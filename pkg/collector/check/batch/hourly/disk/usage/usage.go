package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskusage"
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
	metric, err := c.queryDiskUsage(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryDiskUsage(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetDiskUsage(ctx)
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
		Type: base.DISK_USAGE_PER_HOUR,
		Data: data,
	}

	err = c.retrySaveDiskUsagePerHour(ctx, data)
	if err != nil {
		return base.MetricData{}, err
	}

	err = c.retryDeleteDiskUsage(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetDiskUsage(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getDiskUsage(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get disk usage queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get disk usage queryset")
}

func (c *Check) retrySaveDiskUsagePerHour(ctx context.Context, data []base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveDiskUsagePerHour(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save disk usage per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save disk usage per hour")
}

func (c *Check) retryDeleteDiskUsage(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteDiskUsage(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete disk usage: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete disk usage")
}

func (c *Check) getDiskUsage(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []base.DiskUsageQuerySet
	err := client.DiskUsage.Query().
		Where(diskusage.TimestampGTE(from), diskusage.TimestampLTE(now)).
		GroupBy(diskusage.FieldDevice, diskusage.FieldMountPoint).
		Aggregate(
			ent.Max(diskusage.FieldUsage),
			ent.Mean(diskusage.FieldUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveDiskUsagePerHour(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.DiskUsagePerHour.MapCreateBulk(data, func(q *ent.DiskUsagePerHourCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetMountPoint(data[i].MountPoint).
			SetPeakUsage(data[i].PeakUsage).
			SetAvgUsage(data[i].AvgUsage)
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Check) deleteDiskUsage(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err := client.DiskUsage.Delete().
		Where(diskusage.TimestampGTE(from), diskusage.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
