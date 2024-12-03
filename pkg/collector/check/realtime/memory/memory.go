package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/mem"
)

type Check struct {
	base.BaseCheck
	retryCount base.RetryCount
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
		retryCount: base.RetryCount{
			MaxCollectRetries: base.COLLECT_MAX_RETRIES,
			MaxSaveRetries:    base.SAVE_MAX_RETRIES,
			MaxRetryTime:      base.MAX_RETRY_TIMES,
			Delay:             base.DEFAULT_DELAY,
		},
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.collectAndSaveMemoryUsage(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) collectAndSaveMemoryUsage(ctx context.Context) (base.MetricData, error) {
	usage, err := c.retryCollectMemoryUsage(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	data := base.CheckResult{
		Timestamp: time.Now(),
		Usage:     usage,
	}
	metric := base.MetricData{
		Type: base.MEM,
		Data: []base.CheckResult{data},
	}

	err = c.retrySaveMemoryUsage(ctx, data)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryCollectMemoryUsage(ctx context.Context) (float64, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxCollectRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		usage, err := c.collectMemoryUsage()
		if err == nil {
			return usage, nil
		}

		if attempt < c.retryCount.MaxCollectRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to collect memory usage: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
	}

	return 0, fmt.Errorf("failed to collect memory usage")
}

func (c *Check) retrySaveMemoryUsage(ctx context.Context, data base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveMemoryUsage(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save memory usage: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save memory usage")
}

func (c *Check) collectMemoryUsage() (float64, error) {
	memory, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	return memory.UsedPercent, nil
}

func (c *Check) saveMemoryUsage(ctx context.Context, data base.CheckResult) error {
	client := c.GetClient()
	if err := client.Memory.Create().
		SetTimestamp(data.Timestamp).
		SetUsage(data.Usage).Exec(ctx); err != nil {
		return err
	}

	return nil
}
