package usage

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/hourlydiskusage"
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
	metric, err := c.queryHourlyDiskUsage(ctx)
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

func (c *Check) queryHourlyDiskUsage(ctx context.Context) (base.MetricData, error) {
	querySet, err := c.getHourlyDiskUsage(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	var data []base.CheckResult
	for _, row := range querySet {
		data = append(data, base.CheckResult{
			Timestamp: time.Now(),
			Device:    row.Device,
			Peak:      row.Max,
			Avg:       row.AVG,
		})
	}
	metric := base.MetricData{
		Type: base.DAILY_DISK_USAGE,
		Data: data,
	}

	err = c.deleteHourlyDiskUsage(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) getHourlyDiskUsage(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var querySet []base.DiskUsageQuerySet
	err := client.HourlyDiskUsage.Query().
		Where(hourlydiskusage.TimestampGTE(from), hourlydiskusage.TimestampLTE(now)).
		GroupBy(hourlydiskusage.FieldDevice).
		Aggregate(
			ent.Max(hourlydiskusage.FieldPeak),
			ent.Mean(hourlydiskusage.FieldAvg),
		).Scan(ctx, &querySet)
	if err != nil {
		return querySet, err
	}

	return querySet, nil
}

func (c *Check) deleteHourlyDiskUsage(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	from := time.Now().Add(-24 * time.Hour)

	_, err = tx.HourlyDiskUsage.Delete().
		Where(hourlydiskusage.TimestampLTE(from)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
