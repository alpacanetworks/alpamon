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
	queryset, err := c.getDiskUsage(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	var data []base.CheckResult
	for _, row := range queryset {
		data = append(data, base.CheckResult{
			Timestamp: time.Now(),
			Device:    row.Device,
			PeakUsage: row.Max,
			AvgUsage:  row.AVG,
		})
	}
	metric := base.MetricData{
		Type: base.DISK_USAGE_PER_HOUR,
		Data: data,
	}

	err = c.saveDiskUsagePerHour(data, ctx)
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

	var queryset []base.DiskUsageQuerySet
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
		}).Scan(ctx, &queryset)
	if err != nil {
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveDiskUsagePerHour(data []base.CheckResult, ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.DiskUsagePerHour.MapCreateBulk(data, func(q *ent.DiskUsagePerHourCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetPeakUsage(data[i].PeakUsage).
			SetAvgUsage(data[i].AvgUsage)
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
	defer tx.Rollback()

	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err = tx.DiskUsage.Delete().
		Where(diskusage.TimestampGTE(from), diskusage.TimestampLTE(now)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
