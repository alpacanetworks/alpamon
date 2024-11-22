package net

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/traffic"
	"github.com/rs/zerolog/log"
)

type Check struct {
	base.BaseCheck
}

type trafficQuerySet struct {
	Name            string  `json:"name"`
	PeakInputPkts   float64 `json:"peak_input_pkts"`
	PeakInputBytes  float64 `json:"peak_input_bytes"`
	PeakOutputPkts  float64 `json:"peak_output_pkts"`
	PeakOutputBytes float64 `json:"peak_output_bytes"`
	AvgInputPkts    float64 `json:"avg_input_pkts"`
	AvgInputBytes   float64 `json:"avg_input_bytes"`
	AvgOutputPkts   float64 `json:"avg_output_pkts"`
	AvgOutputBytes  float64 `json:"avg_output_bytes"`
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	queryset, err := c.getTrafficPeakAndAvg(ctx)
	metric := base.MetricData{
		Type: base.NET_PER_HOUR,
		Data: []base.CheckResult{},
	}

	if err == nil {
		for _, row := range queryset {
			data := base.CheckResult{
				Timestamp:       time.Now(),
				Name:            row.Name,
				PeakInputPkts:   uint64(row.PeakInputPkts),
				PeakInputBytes:  uint64(row.PeakInputBytes),
				PeakOutputPkts:  uint64(row.PeakOutputPkts),
				PeakOutputBytes: uint64(row.PeakOutputBytes),
				AvgInputPkts:    uint64(row.AvgInputPkts),
				AvgInputBytes:   uint64(row.AvgInputBytes),
				AvgOutputPkts:   uint64(row.AvgOutputPkts),
				AvgOutputBytes:  uint64(row.AvgOutputBytes),
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

func (c *Check) getTrafficPeakAndAvg(ctx context.Context) ([]trafficQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []trafficQuerySet
	err := client.Traffic.Query().
		Where(traffic.TimestampGTE(from), traffic.TimestampLTE(now)).
		GroupBy(traffic.FieldName).
		Aggregate(
			ent.As(ent.Max(traffic.FieldInputPkts), "peak_input_pkts"),
			ent.As(ent.Max(traffic.FieldInputBytes), "peak_input_bytes"),
			ent.As(ent.Max(traffic.FieldOutputPkts), "peak_output_pkts"),
			ent.As(ent.Max(traffic.FieldOutputBytes), "peak_output_bytes"),
			ent.As(ent.Mean(traffic.FieldInputPkts), "avg_input_pkts"),
			ent.As(ent.Mean(traffic.FieldInputBytes), "avg_input_bytes"),
			ent.As(ent.Mean(traffic.FieldOutputPkts), "avg_output_pkts"),
			ent.As(ent.Mean(traffic.FieldOutputBytes), "avg_output_bytes"),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}
