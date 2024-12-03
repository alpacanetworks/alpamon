package diskio

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/disk"
)

type Check struct {
	base.BaseCheck
	retryCount base.RetryCount
	lastMetric map[string]disk.IOCountersStat
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
		retryCount: base.RetryCount{
			MaxCollectRetries: base.COLLECT_MAX_RETRIES,
			MaxSaveRetries:    base.SAVE_MAX_RETRIES,
			MaxRetryTime:      base.MAX_RETRY_TIMES,
			Delay:             base.DEFAULT_DELAY,
		},
		lastMetric: make(map[string]disk.IOCountersStat),
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.collectAndSaveDiskIO(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) collectAndSaveDiskIO(ctx context.Context) (base.MetricData, error) {
	ioCounters, err := c.retryCollectDiskIO(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	metric := base.MetricData{
		Type: base.DISK_IO,
		Data: c.parseDiskIO(ioCounters),
	}

	err = c.retrySaveDiskIO(ctx, metric.Data)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryCollectDiskIO(ctx context.Context) (map[string]disk.IOCountersStat, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxCollectRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		ioCounters, err := c.collectDiskIO()

		if err == nil {
			return ioCounters, nil
		}

		if attempt < c.retryCount.MaxCollectRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to collect disk io: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to collect disk io")
}

func (c *Check) parseDiskIO(ioCounters map[string]disk.IOCountersStat) []base.CheckResult {
	var data []base.CheckResult
	for name, ioCounter := range ioCounters {
		var readBytes, writeBytes uint64

		if lastCounter, exist := c.lastMetric[name]; exist {
			readBytes = ioCounter.ReadBytes - lastCounter.ReadBytes
			writeBytes = ioCounter.WriteBytes - lastCounter.WriteBytes
		} else {
			readBytes = ioCounter.ReadBytes
			writeBytes = ioCounter.WriteBytes
		}

		c.lastMetric[name] = ioCounter
		data = append(data, base.CheckResult{
			Timestamp:  time.Now(),
			Device:     name,
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
		})
	}

	return data
}

func (c *Check) retrySaveDiskIO(ctx context.Context, data []base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveDiskIO(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save disk io: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save disk io")
}

func (c *Check) collectDiskIO() (map[string]disk.IOCountersStat, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}

	return ioCounters, nil
}

func (c *Check) saveDiskIO(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.DiskIO.MapCreateBulk(data, func(q *ent.DiskIOCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetReadBytes(int64(data[i].ReadBytes)).
			SetWriteBytes(int64(data[i].WriteBytes))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
