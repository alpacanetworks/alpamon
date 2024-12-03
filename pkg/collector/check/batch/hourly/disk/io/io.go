package io

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskio"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/rs/zerolog/log"
)

type Check struct {
	base.BaseCheck
	retryCount base.RetryCount
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
		retryCount: base.RetryCount{
			MaxGetRetries:    base.GET_MAX_RETRIES,
			MaxSaveRetries:   base.SAVE_MAX_RETRIES,
			MaxDeleteRetries: base.DELETE_MAX_RETRIES,
			MaxRetryTime:     base.MAX_RETRY_TIMES,
			Delay:            base.DEFAULT_DELAY,
		},
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.queryDiskIO(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryDiskIO(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetDiskIO(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	var data []base.CheckResult
	for _, row := range queryset {
		data = append(data, base.CheckResult{
			Timestamp:      time.Now(),
			Device:         row.Device,
			PeakWriteBytes: uint64(row.PeakWriteBytes),
			PeakReadBytes:  uint64(row.PeakReadBytes),
			AvgWriteBytes:  uint64(row.AvgWriteBytes),
			AvgReadBytes:   uint64(row.AvgReadBytes),
		})
	}
	metric := base.MetricData{
		Type: base.DISK_IO_PER_HOUR,
		Data: data,
	}

	err = c.retrySaveDiskIOPerHour(ctx, data)
	if err != nil {
		return base.MetricData{}, err
	}

	err = c.retryDeleteDiskIO(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetDiskIO(ctx context.Context) ([]base.DiskIOQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getDiskIO(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get disk io queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get disk io queryset")
}

func (c *Check) retrySaveDiskIOPerHour(ctx context.Context, data []base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveDiskIOPerHour(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save disk io per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save disk io per hour")
}

func (c *Check) retryDeleteDiskIO(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteDiskIO(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete disk io: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete disk io")
}

func (c *Check) getDiskIO(ctx context.Context) ([]base.DiskIOQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []base.DiskIOQuerySet
	err := client.DiskIO.Query().
		Where(diskio.TimestampGTE(from), diskio.TimestampLTE(now)).
		GroupBy(diskio.FieldDevice).
		Aggregate(
			ent.As(ent.Max(diskio.FieldReadBytes), "peak_read_bytes"),
			ent.As(ent.Max(diskio.FieldWriteBytes), "peak_write_bytes"),
			ent.As(ent.Mean(diskio.FieldReadBytes), "avg_read_bytes"),
			ent.As(ent.Mean(diskio.FieldWriteBytes), "avg_write_bytes"),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveDiskIOPerHour(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.DiskIOPerHour.MapCreateBulk(data, func(q *ent.DiskIOPerHourCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetPeakReadBytes(int64(data[i].PeakReadBytes)).
			SetPeakWriteBytes(int64(data[i].PeakWriteBytes)).
			SetAvgReadBytes(int64(data[i].AvgReadBytes)).
			SetAvgWriteBytes(int64(data[i].AvgWriteBytes))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Check) deleteDiskIO(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err := client.DiskIO.Delete().
		Where(diskio.TimestampGTE(from), diskio.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
