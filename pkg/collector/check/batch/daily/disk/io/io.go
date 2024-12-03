package io

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskioperhour"
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
			MaxGetRetries:    3,
			MaxDeleteRetries: 2,
			MaxRetryTime:     base.MAX_RETRY_TIMES,
			Delay:            base.DEFAULT_DELAY,
		},
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.queryDiskIOPerHour(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryDiskIOPerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetDiskIOPerHour(ctx)
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
		Type: base.DISK_IO_PER_DAY,
		Data: data,
	}

	err = c.retryDeleteDiskIOPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetDiskIOPerHour(ctx context.Context) ([]base.DiskIOQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getDiskIOPerHour(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get disk io per hour queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get disk io per hour queryset")
}

func (c *Check) retryDeleteDiskIOPerHour(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteDiskIOPerHour(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete disk io per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete disk io per hour")
}

func (c *Check) getDiskIOPerHour(ctx context.Context) ([]base.DiskIOQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var queryset []base.DiskIOQuerySet
	err := client.DiskIOPerHour.Query().
		Where(diskioperhour.TimestampGTE(from), diskioperhour.TimestampLTE(now)).
		GroupBy(diskioperhour.FieldDevice).
		Aggregate(
			ent.As(ent.Max(diskioperhour.FieldPeakReadBytes), "peak_read_bytes"),
			ent.As(ent.Max(diskioperhour.FieldPeakWriteBytes), "peak_write_bytes"),
			ent.As(ent.Mean(diskioperhour.FieldAvgReadBytes), "avg_read_bytes"),
			ent.As(ent.Mean(diskioperhour.FieldAvgWriteBytes), "avg_write_bytes"),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteDiskIOPerHour(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	_, err := client.DiskIOPerHour.Delete().
		Where(diskioperhour.TimestampGTE(from), diskioperhour.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
