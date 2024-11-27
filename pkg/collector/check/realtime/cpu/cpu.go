package cpu

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/shirou/gopsutil/v4/cpu"
)

type Check struct {
	base.BaseCheck
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
	}
}

func (c *Check) Execute(ctx context.Context) {
	var checkError base.CheckError

	usage, err := c.collectCPUUsage()
	if err != nil {
		checkError.CollectError = err
	}

	metric := base.MetricData{
		Type: base.CPU,
		Data: []base.CheckResult{},
	}
	if checkError.CollectError == nil {
		data := base.CheckResult{
			Timestamp: time.Now(),
			Usage:     usage,
		}
		metric.Data = append(metric.Data, data)

		if err := c.saveCPUUsage(ctx, data); err != nil {
			checkError.SaveQueryError = err
		}
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	if checkError.CollectError != nil || checkError.SaveQueryError != nil {
		buffer.FailureQueue <- metric
	} else {
		buffer.SuccessQueue <- metric
	}
}

func (c *Check) collectCPUUsage() (float64, error) {
	usage, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}

	if len(usage) == 0 {
		return 0, fmt.Errorf("no CPU usage data returned")
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
