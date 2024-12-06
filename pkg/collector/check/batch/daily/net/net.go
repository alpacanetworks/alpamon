package net

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/trafficperhour"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/rs/zerolog/log"
)

type Check struct {
	base.BaseCheck
	retryCount base.RetryCount
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
		retryCount: base.RetryCount{
			MaxGetRetries:    3,
			MaxDeleteRetries: 2,
			MaxRetryTime:     base.MAX_RETRY_TIMES,
			Delay:            base.DEFAULT_DELAY,
		},
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.queryTrafficPerHour(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryTrafficPerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetTrafficPerHour(ctx)
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

	err = c.retryDeleteTrafficPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetTrafficPerHour(ctx context.Context) ([]base.TrafficQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getTrafficPerHour(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get traffic per hour queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get traffic per hour queryset")
}

func (c *Check) retryDeleteTrafficPerHour(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteTrafficPerHour(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete traffic per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete traffic per hour")
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
