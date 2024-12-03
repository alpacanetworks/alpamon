package diskusage

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

var excludedFileSystems = map[string]bool{
	"tmpfs":    true,
	"devtmpfs": true,
	"proc":     true,
	"sysfs":    true,
	"cgroup":   true,
	"overlay":  true,
}

type Check struct {
	base.BaseCheck
	retryCount base.RetryCount
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
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.collectAndSaveDiskUsage(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) collectAndSaveDiskUsage(ctx context.Context) (base.MetricData, error) {
	partitions, err := c.retryCollectDiskPartitions(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	metric := base.MetricData{
		Type: base.DISK_USAGE,
		Data: c.parseDiskUsage(partitions),
	}

	err = c.retrySaveDiskUsage(ctx, metric.Data)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryCollectDiskPartitions(ctx context.Context) ([]disk.PartitionStat, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxCollectRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		partitions, err := c.collectDiskPartitions()
		if err == nil && len(partitions) > 0 {
			return partitions, nil
		}

		if attempt < c.retryCount.MaxCollectRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to collect disk partitions: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to collect disk partitions")
}

func (c *Check) parseDiskUsage(partitions []disk.PartitionStat) []base.CheckResult {
	var data []base.CheckResult
	for _, partition := range partitions {
		usage, err := c.collectDiskUsage(partition.Mountpoint)
		if err == nil {
			data = append(data, base.CheckResult{
				Timestamp:  time.Now(),
				Device:     partition.Device,
				MountPoint: partition.Mountpoint,
				Usage:      usage.UsedPercent,
				Total:      usage.Total,
				Free:       usage.Free,
				Used:       usage.Used,
			})
		}
	}

	return data
}

func (c *Check) retrySaveDiskUsage(ctx context.Context, data []base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveDiskUsage(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save disk usage: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save disk usage")
}

func (c *Check) collectDiskPartitions() ([]disk.PartitionStat, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}

	var filteredPartitions []disk.PartitionStat
	for _, partition := range partitions {
		if !excludedFileSystems[partition.Fstype] {
			filteredPartitions = append(filteredPartitions, partition)
		}
	}

	return filteredPartitions, nil
}

func (c *Check) collectDiskUsage(path string) (*disk.UsageStat, error) {
	usage, err := disk.Usage(path)
	if err != nil {
		return nil, err
	}

	return usage, nil
}

func (c *Check) saveDiskUsage(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.DiskUsage.MapCreateBulk(data, func(q *ent.DiskUsageCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetMountPoint(data[i].MountPoint).
			SetUsage(data[i].Usage).
			SetTotal(int64(data[i].Total)).
			SetFree(int64(data[i].Free)).
			SetUsed(int64(data[i].Used))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
