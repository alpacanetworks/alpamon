package io

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskio"
	"github.com/rs/zerolog/log"
)

type Check struct {
	base.BaseCheck
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
	}
}

func (c *Check) Execute(ctx context.Context) {
	var checkError base.CheckError

	queryset, err := c.getDiskIO(ctx)
	if err != nil {
		checkError.GetQueryError = err
	}

	metric := base.MetricData{
		Type: base.DISK_IO_PER_HOUR,
		Data: []base.CheckResult{},
	}
	if checkError.GetQueryError == nil {
		for _, row := range queryset {
			data := base.CheckResult{
				Timestamp:      time.Now(),
				Device:         row.Device,
				PeakWriteBytes: uint64(row.PeakWriteBytes),
				PeakReadBytes:  uint64(row.PeakReadBytes),
				AvgWriteBytes:  uint64(row.AvgWriteBytes),
				AvgReadBytes:   uint64(row.AvgReadBytes),
			}
			metric.Data = append(metric.Data, data)
		}

		if err := c.saveDiskIOPerHour(ctx, metric.Data); err != nil {
			checkError.SaveQueryError = err
		}

		if err := c.deleteDiskIO(ctx); err != nil {
			checkError.DeleteQueryError = err
		}
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	isFailed := checkError.GetQueryError != nil ||
		checkError.SaveQueryError != nil ||
		checkError.DeleteQueryError != nil
	if isFailed {
		buffer.FailureQueue <- metric
	} else {
		buffer.SuccessQueue <- metric
	}
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
