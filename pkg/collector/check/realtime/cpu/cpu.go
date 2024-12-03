package cpu

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/cpu"
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
	metric, err := c.collectAndSaveCPUUsage(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) collectAndSaveCPUUsage(ctx context.Context) (base.MetricData, error) {
	usage, err := c.retryCollectCPUUsage(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	data := base.CheckResult{
		Timestamp: time.Now(),
		Usage:     usage,
	}
	metric := base.MetricData{
		Type: base.CPU,
		Data: []base.CheckResult{data},
	}

	err = c.retrySaveCPUUsage(ctx, data)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryCollectCPUUsage(ctx context.Context) (float64, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxCollectRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		usage, err := c.collectCPUUsage()
		if err == nil {
			return usage, nil
		}

		if attempt < c.retryCount.MaxCollectRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to collect cpu usage: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
	}

	return 0, fmt.Errorf("failed to collect cpu usage")
}

func (c *Check) retrySaveCPUUsage(ctx context.Context, data base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			log.Debug().Msg("asdf")
			break
		}

		err := c.saveCPUUsage(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save cpu usage: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save cpu usage")
}

func (c *Check) collectCPUUsage() (float64, error) {
	usage, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}

	if len(usage) == 0 {
		return 0, fmt.Errorf("no cpu usage data returned")
	}

	return usage[0], nil
}

func (c *Check) saveCPUUsage(ctx context.Context, data base.CheckResult) error {
	client := c.GetClient()
	if err := client.CPU.Create().
		SetTimestamp(data.Timestamp).
		SetUsage(data.Usage).Exec(ctx); err != nil {
		return err
	}

	return nil
}
