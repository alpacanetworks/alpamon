package io

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskioperhour"
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
	metric, err := c.queryDiskIOPerHour(ctx)
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

func (c *Check) queryDiskIOPerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.getDiskIOPerHour(ctx)
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

	err = c.deleteDiskIOPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
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
		).Scan(ctx, &queryset)
	if err != nil {
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteDiskIOPerHour(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	from := now.Add(-24 * time.Hour)

	_, err = tx.DiskIOPerHour.Delete().
		Where(diskioperhour.TimestampGTE(from), diskioperhour.TimestampLTE(now)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
