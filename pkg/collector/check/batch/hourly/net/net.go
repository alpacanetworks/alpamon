package net

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/traffic"
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
			MaxGetRetries:    base.GET_MAX_RETRIES,
			MaxSaveRetries:   base.SAVE_MAX_RETRIES,
			MaxDeleteRetries: base.DELETE_MAX_RETRIES,
			MaxRetryTime:     base.MAX_RETRY_TIMES,
			Delay:            base.DEFAULT_DELAY,
		},
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.queryTraffic(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryTraffic(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetTraffic(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	var data []base.CheckResult
	for _, row := range queryset {
		data = append(data, base.CheckResult{
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
		})
	}
	metric := base.MetricData{
		Type: base.NET_PER_HOUR,
		Data: data,
	}

	err = c.retrySaveTrafficPerHour(ctx, data)
	if err != nil {
		return base.MetricData{}, err
	}

	err = c.retryDeleteTraffic(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetTraffic(ctx context.Context) ([]base.TrafficQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getTraffic(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get traffic queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get traffic queryset")
}

func (c *Check) retrySaveTrafficPerHour(ctx context.Context, data []base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveTrafficPerHour(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save traffic per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save traffic per hour")
}

func (c *Check) retryDeleteTraffic(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteTraffic(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete traffic: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete traffic")
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
