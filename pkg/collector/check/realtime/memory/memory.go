package memory

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/shirou/gopsutil/v4/mem"
)

type Check struct {
	base.BaseCheck
}

type MemoryCheckError struct {
	CollectError error
	QueryError   error
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	var checkError MemoryCheckError

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
			checkError.QueryError = err
		}
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	if checkError.CollectError != nil || checkError.QueryError != nil {
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
