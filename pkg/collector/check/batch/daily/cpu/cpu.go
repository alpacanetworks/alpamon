package cpu

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/cpuperhour"
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
	metric, err := c.queryCPUPerHour(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryCPUPerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetCPUPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	data := base.CheckResult{
		Timestamp: time.Now(),
		PeakUsage: queryset[0].Max,
		AvgUsage:  queryset[0].AVG,
	}
	metric := base.MetricData{
		Type: base.CPU_PER_DAY,
		Data: []base.CheckResult{data},
	}

	err = c.retryDeleteCPUPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetCPUPerHour(ctx context.Context) ([]base.CPUQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getCPUPerHour(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get cpu usage per hour queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get cpu usage per hour queryset")
}

func (c *Check) retryDeleteCPUPerHour(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteCPUPerHour(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete cpu usage per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete cpu usage per hour")
}

func (c *Check) getCPUPerHour(ctx context.Context) ([]base.CPUQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var queryset []base.CPUQuerySet
	err := client.CPUPerHour.Query().
		Where(cpuperhour.TimestampGTE(from), cpuperhour.TimestampLTE(now)).
		Aggregate(
			ent.Max(cpuperhour.FieldPeakUsage),
			ent.Mean(cpuperhour.FieldAvgUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteCPUPerHour(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	_, err := client.CPUPerHour.Delete().
		Where(cpuperhour.TimestampGTE(from), cpuperhour.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
