package net

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/trafficperhour"
	"github.com/rs/zerolog/log"
)

type Check struct {
	base.BaseCheck
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	var checkError base.CheckError

	queryset, err := c.getTrafficPerHour(ctx)
	if err != nil {
		checkError.GetQueryError = err
	}

	metric := base.MetricData{
		Type: base.NET_PER_DAY,
		Data: []base.CheckResult{},
	}
	if checkError.GetQueryError == nil {
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

			if err := c.deleteTrafficPerHour(ctx); err != nil {
				checkError.DeleteQueryError = err
			}
		}
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	if checkError.GetQueryError != nil || checkError.DeleteQueryError != nil {
		buffer.FailureQueue <- metric
	} else {
		buffer.SuccessQueue <- metric
	}
}

func (c *Check) getTrafficPerHour(ctx context.Context) ([]base.TrafficQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var queryset []base.TrafficQuerySet
	err := client.TrafficPerHour.Query().
		Where(trafficperhour.TimestampGTE(from), trafficperhour.TimestampLTE(now)).
		GroupBy(trafficperhour.FieldName).
		Aggregate(
			ent.As(ent.Max(trafficperhour.FieldPeakInputPkts), "peak_input_pkts"),
			ent.As(ent.Max(trafficperhour.FieldPeakInputBytes), "peak_input_bytes"),
			ent.As(ent.Max(trafficperhour.FieldPeakOutputPkts), "peak_output_pkts"),
			ent.As(ent.Max(trafficperhour.FieldPeakOutputBytes), "peak_output_bytes"),
			ent.As(ent.Mean(trafficperhour.FieldAvgInputPkts), "avg_input_pkts"),
			ent.As(ent.Mean(trafficperhour.FieldAvgInputBytes), "avg_input_bytes"),
			ent.As(ent.Mean(trafficperhour.FieldAvgOutputPkts), "avg_output_pkts"),
			ent.As(ent.Mean(trafficperhour.FieldAvgOutputBytes), "avg_output_bytes"),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteTrafficPerHour(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	_, err := client.TrafficPerHour.Delete().
		Where(trafficperhour.TimestampGTE(from), trafficperhour.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
