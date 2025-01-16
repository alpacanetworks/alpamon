package usage

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskusage"
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
	metric, err := c.queryDiskUsage(ctx)
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

func (c *Check) queryDiskUsage(ctx context.Context) (base.MetricData, error) {
	querySet, err := c.getDiskUsage(ctx)
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
		Type: base.HOURLY_DISK_USAGE,
		Data: data,
	}

	err = c.saveHourlyDiskUsage(data, ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	err = c.deleteDiskUsage(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) getDiskUsage(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	client := c.GetClient()

	var querySet []base.DiskUsageQuerySet
	err := client.DiskUsage.Query().
		Modify(func(s *sql.Selector) {
			now := time.Now()
			from := now.Add(-1 * time.Hour)
			usageExpr := "(CAST(SUM(used) AS FLOAT) * 100.0) / NULLIF(SUM(total), 0)"
			t := sql.Table(diskusage.Table)

			subq := sql.Select(
				"device",
				"timestamp",
				sql.As(usageExpr, "usage"),
			).
				From(t).
				Where(
					sql.And(
						sql.GTE(t.C(diskusage.FieldTimestamp), from),
						sql.LTE(t.C(diskusage.FieldTimestamp), now),
					),
				).
				GroupBy("device", "timestamp")

			*s = *sql.Select(
				"device",
				sql.As(sql.Max("usage"), "max"),
				sql.As(sql.Avg("usage"), "avg"),
			).From(subq).GroupBy("device")
		}).Scan(ctx, &querySet)
	if err != nil {
		return querySet, err
	}

	return querySet, nil
}

func (c *Check) saveHourlyDiskUsage(data []base.CheckResult, ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	err = tx.HourlyDiskUsage.MapCreateBulk(data, func(q *ent.HourlyDiskUsageCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetPeak(data[i].Peak).
			SetAvg(data[i].Avg)
	}).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func (c *Check) deleteDiskUsage(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	from := time.Now().Add(-1 * time.Hour)

	_, err = tx.DiskUsage.Delete().
		Where(diskusage.TimestampLTE(from)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
