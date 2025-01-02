package io

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskio"
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
	metric, err := c.queryDiskIO(ctx)
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

func (c *Check) queryDiskIO(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.getDiskIO(ctx)
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

	err = c.saveDiskIOPerHour(data, ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	err = c.deleteDiskIO(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
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
		).Scan(ctx, &queryset)
	if err != nil {
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveDiskIOPerHour(data []base.CheckResult, ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return nil
	}
	defer func() { _ = tx.Rollback() }()

	err = tx.DiskIOPerHour.MapCreateBulk(data, func(q *ent.DiskIOPerHourCreate, i int) {
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

	_ = tx.Commit()

	return nil
}

func (c *Check) deleteDiskIO(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return nil
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err = tx.DiskIO.Delete().
		Where(diskio.TimestampGTE(from), diskio.TimestampLTE(now)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
