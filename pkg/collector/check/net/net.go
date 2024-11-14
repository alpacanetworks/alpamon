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
	interfaces, _ := c.collectInterfaces()

	metric := base.MetricData{
		Type: checkType,
		Data: []base.CheckResult{},
	}
	if err == nil {
		for _, ioCounter := range ioCounters {
			if _, ok := interfaces[ioCounter.Name]; ok {
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

func (c *Check) collectInterfaces() (map[string]net.InterfaceStat, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	interfaces := map[string]net.InterfaceStat{}
	for _, iface := range ifaces {
		mac := iface.HardwareAddr
		if mac == "" {
			continue
		}
		interfaces[iface.Name] = iface
	}

	return interfaces, nil
}

func (c *Check) collectIOCounters() ([]net.IOCountersStat, error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	return ioCounters, nil
}
