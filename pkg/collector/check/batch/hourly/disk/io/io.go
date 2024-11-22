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

type diskIOQuerySet struct {
	Device         string  `json:"device" db:"device"`
	PeakReadBytes  float64 `json:"peak_read_bytes"`
	PeakWriteBytes float64 `json:"peak_write_bytes"`
	AvgReadBytes   float64 `json:"avg_read_bytes"`
	AvgWriteBytes  float64 `json:"avg_write_bytes"`
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	queryset, err := c.getDiskIOPeakAndAvg(ctx)
	metric := base.MetricData{
		Type: base.DISK_IO_PER_HOUR,
		Data: []base.CheckResult{},
	}

	if err == nil {
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
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	if err != nil {
		buffer.FailureQueue <- metric
	} else {
		buffer.SuccessQueue <- metric
	}
}

func (c *Check) getDiskIOPeakAndAvg(ctx context.Context) ([]diskIOQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []diskIOQuerySet
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