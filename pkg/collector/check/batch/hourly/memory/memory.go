package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/memory"
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
	metric, err := c.queryMemoryUsage(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) queryMemoryUsage(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.retryGetMemory(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	data := base.CheckResult{
		Timestamp: time.Now(),
		PeakUsage: queryset[0].Max,
		AvgUsage:  queryset[0].AVG,
	}
	metric := base.MetricData{
		Type: base.MEM_PER_HOUR,
		Data: []base.CheckResult{data},
	}

	err = c.retrySaveMemoryPerHour(ctx, data)
	if err != nil {
		return base.MetricData{}, err
	}

	err = c.retryDeleteMemory(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryGetMemory(ctx context.Context) ([]base.MemoryQuerySet, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxGetRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		queryset, err := c.getMemory(ctx)
		if err == nil {
			return queryset, nil
		}

		if attempt < c.retryCount.MaxGetRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to get memory queryset: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed to get memory queryset")
}

func (c *Check) retrySaveMemoryPerHour(ctx context.Context, data base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveMemoryPerHour(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save memory usage per hour: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save memory usage per hour")
}

func (c *Check) retryDeleteMemory(ctx context.Context) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.deleteMemory(ctx)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxDeleteRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to delete memory usage: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to delete memory usage")
}

func (c *Check) getMemory(ctx context.Context) ([]base.MemoryQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []base.MemoryQuerySet
	err := client.Memory.Query().
		Where(memory.TimestampGTE(from), memory.TimestampLTE(now)).
		Aggregate(
			ent.Max(memory.FieldUsage),
			ent.Mean(memory.FieldUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveMemoryPerHour(ctx context.Context, data base.CheckResult) error {
	client := c.GetClient()
	if err := client.MemoryPerHour.Create().
		SetTimestamp(data.Timestamp).
		SetPeakUsage(data.PeakUsage).
		SetAvgUsage(data.AvgUsage).Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Check) deleteMemory(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err := client.Memory.Delete().
		Where(memory.TimestampGTE(from), memory.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
