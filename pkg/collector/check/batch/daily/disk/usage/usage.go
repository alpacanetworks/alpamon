package usage

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskusageperhour"
)

type Check struct {
	base.BaseCheck
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
	}
}

func (c *Check) Execute(ctx context.Context) error {
	metric, err := c.queryDiskUsagePerHour(ctx)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric

	return nil
}

func (c *Check) queryDiskUsagePerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.getDiskUsagePerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	var data []base.CheckResult
	for _, row := range queryset {
		data = append(data, base.CheckResult{
			Timestamp: time.Now(),
			Device:    row.Device,
			Peak:      row.Max,
			Avg:       row.AVG,
		})
	}
	metric := base.MetricData{
		Type: base.DISK_USAGE_PER_DAY,
		Data: data,
	}

	err = c.deleteDiskUsagePerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) getDiskUsagePerHour(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var queryset []base.DiskUsageQuerySet
	err := client.DiskUsagePerHour.Query().
		Where(diskusageperhour.TimestampGTE(from), diskusageperhour.TimestampLTE(now)).
		GroupBy(diskusageperhour.FieldDevice).
		Aggregate(
			ent.Max(diskusageperhour.FieldPeak),
			ent.Mean(diskusageperhour.FieldAvg),
		).Scan(ctx, &queryset)
	if err != nil {
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteDiskUsagePerHour(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	from := time.Now().Add(-24 * time.Hour)

	_, err = tx.DiskUsagePerHour.Delete().
		Where(diskusageperhour.TimestampLTE(from)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
