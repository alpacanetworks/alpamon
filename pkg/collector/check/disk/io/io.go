package diskio

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/shirou/gopsutil/v4/disk"
)

const (
	checkType = base.DISK_IO
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
	ioCounters, err := c.collectDiskIO()

	metric := base.MetricData{
		Type: checkType,
		Data: []base.CheckResult{},
	}
	if err == nil {
		for name, ioCounter := range ioCounters {
			data := base.CheckResult{
				Timestamp:  time.Now(),
				Device:     name,
				ReadBytes:  ioCounter.ReadBytes,
				WriteBytes: ioCounter.WriteBytes,
			}
			metric.Data = append(metric.Data, data)
		}
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

func (c *Check) collectDiskIO() (map[string]disk.IOCountersStat, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}

	return ioCounters, nil
}
