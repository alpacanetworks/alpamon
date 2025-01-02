package net

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/trafficperhour"
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
	metric, err := c.queryTrafficPerHour(ctx)
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

func (c *Check) queryTrafficPerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.getTrafficPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	var data []base.CheckResult
	for _, row := range queryset {
		data = append(data, base.CheckResult{
			Timestamp:     time.Now(),
			Name:          row.Name,
			PeakInputPps:  row.PeakInputPps,
			PeakInputBps:  row.PeakInputBps,
			PeakOutputPps: row.PeakOutputPps,
			PeakOutputBps: row.PeakOutputBps,
			AvgInputPps:   row.AvgInputPps,
			AvgInputBps:   row.AvgInputBps,
			AvgOutputPps:  row.AvgOutputPps,
			AvgOutputBps:  row.AvgOutputBps,
		})
	}
	metric := base.MetricData{
		Type: base.NET_PER_DAY,
		Data: data,
	}

	err = c.deleteTrafficPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
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
			ent.As(ent.Max(trafficperhour.FieldPeakInputPps), "peak_input_pps"),
			ent.As(ent.Max(trafficperhour.FieldPeakInputBps), "peak_input_bps"),
			ent.As(ent.Max(trafficperhour.FieldPeakOutputPps), "peak_output_pps"),
			ent.As(ent.Max(trafficperhour.FieldPeakOutputBps), "peak_output_bps"),
			ent.As(ent.Mean(trafficperhour.FieldAvgInputPps), "avg_input_pps"),
			ent.As(ent.Mean(trafficperhour.FieldAvgInputBps), "avg_input_bps"),
			ent.As(ent.Mean(trafficperhour.FieldAvgOutputPps), "avg_output_pps"),
			ent.As(ent.Mean(trafficperhour.FieldAvgOutputBps), "avg_output_bps"),
		).Scan(ctx, &queryset)
	if err != nil {
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteTrafficPerHour(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	from := now.Add(-24 * time.Hour)

	_, err = tx.TrafficPerHour.Delete().
		Where(trafficperhour.TimestampGTE(from), trafficperhour.TimestampLTE(now)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
