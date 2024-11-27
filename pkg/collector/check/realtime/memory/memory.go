package memory

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/shirou/gopsutil/v4/mem"
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

	usage, err := c.collectMemoryUsage()
	if err != nil {
		checkError.CollectError = err
	}

	metric := base.MetricData{
		Type: base.MEM,
		Data: []base.CheckResult{},
	}
	if checkError.CollectError == nil {
		data := base.CheckResult{
			Timestamp: time.Now(),
			Usage:     usage,
		}
		metric.Data = append(metric.Data, data)

		if err := c.saveMemoryUsage(ctx, data); err != nil {
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
