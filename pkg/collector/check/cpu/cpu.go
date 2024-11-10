package cpu

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/shirou/gopsutil/v4/cpu"
)

const (
	checkType = base.CPU
)

type Check struct {
	base.BaseCheck
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer),
	}
}

func (c *Check) Execute(ctx context.Context) {
	usage, err := c.collectCPUUsage()

	metric := base.MetricData{
		Type: checkType,
		Data: []base.CheckResult{},
	}
	if err == nil {
		data := base.CheckResult{
			Timestamp: time.Now(),
			Usage:     usage,
		}
		metric.Data = append(metric.Data, data)
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	if err != nil {
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
