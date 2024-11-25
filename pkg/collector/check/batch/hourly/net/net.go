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

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	var checkError base.CheckError

	queryset, err := c.getTraffic(ctx)
	if err != nil {
		checkError.GetQueryError = err
	}

	metric := base.MetricData{
		Type: base.NET_PER_HOUR,
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
		}

		if err := c.saveTrafficPerHour(ctx, metric.Data); err != nil {
			checkError.SaveQueryError = err
		}

		if err := c.deleteTraffic(ctx); err != nil {
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

func (c *Check) getTraffic(ctx context.Context) ([]base.TrafficQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []base.TrafficQuerySet
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

func (c *Check) saveTrafficPerHour(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.TrafficPerHour.MapCreateBulk(data, func(q *ent.TrafficPerHourCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetName(data[i].Name).
			SetPeakInputPkts(int64(data[i].PeakInputPkts)).
			SetPeakInputBytes(int64(data[i].PeakInputBytes)).
			SetPeakOutputPkts(int64(data[i].PeakOutputPkts)).
			SetPeakOutputBytes(int64(data[i].PeakOutputBytes)).
			SetAvgInputPkts(int64(data[i].AvgInputPkts)).
			SetAvgInputBytes(int64(data[i].AvgInputBytes)).
			SetAvgOutputPkts(int64(data[i].AvgOutputPkts)).
			SetAvgOutputBytes(int64(data[i].AvgOutputBytes))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Check) deleteTraffic(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err := client.Traffic.Delete().
		Where(traffic.TimestampGTE(from), traffic.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
