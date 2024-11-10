package net

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/shirou/gopsutil/v4/net"
)

const (
	checkType = base.NET
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
	ioCounters, err := c.collectIOCounters()

	var metric base.MetricData
	if err != nil {
		metric = base.MetricData{
			Type: checkType,
			Data: []base.CheckResult{},
		}
	} else {
		for _, ioCounter := range ioCounters {
			data := base.CheckResult{
				Timestamp:   time.Now(),
				Name:        ioCounter.Name,
				InputPkts:   ioCounter.PacketsRecv,
				InputBytes:  ioCounter.BytesRecv,
				OutputPkts:  ioCounter.PacketsSent,
				OutputBytes: ioCounter.BytesSent,
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

func (c *Check) collectIOCounters() ([]net.IOCountersStat, error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	return ioCounters, nil
}
