package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskusageperhour"
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
	metric, err := c.queryDiskUsagePerHour(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryDiskUsagePerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetDiskUsagePerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	var data []base.CheckResult
	for _, row := range queryset {
		data = append(data, base.CheckResult{
			Timestamp:  time.Now(),
			Device:     row.Device,
			MountPoint: row.MountPoint,
			PeakUsage:  row.Max,
			AvgUsage:   row.AVG,
		})
	}
	metric := base.MetricData{
		Type: base.DISK_USAGE_PER_DAY,
		Data: data,
	}

	err = c.retryDeleteDiskUsagePerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetDiskUsagePerHour(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getDiskUsagePerHour(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get disk usage per hour queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get disk usage per hour queryset")
}

func (c *Check) retryDeleteDiskUsagePerHour(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteDiskUsagePerHour(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete disk usage per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete disk usage per hour")
}

func (c *Check) getDiskUsagePerHour(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var queryset []base.DiskUsageQuerySet
	err := client.DiskUsagePerHour.Query().
		Where(diskusageperhour.TimestampGTE(from), diskusageperhour.TimestampLTE(now)).
		GroupBy(diskusageperhour.FieldDevice, diskusageperhour.FieldMountPoint).
		Aggregate(
			ent.Max(diskusageperhour.FieldPeakUsage),
			ent.Mean(diskusageperhour.FieldAvgUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteDiskUsagePerHour(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	_, err := client.DiskUsagePerHour.Delete().
		Where(diskusageperhour.TimestampGTE(from), diskusageperhour.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
