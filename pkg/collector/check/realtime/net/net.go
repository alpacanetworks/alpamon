package net

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/shirou/gopsutil/v4/net"
)

type Check struct {
	base.BaseCheck
	lastMetric map[string]net.IOCountersStat
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck:  base.NewBaseCheck(args),
		lastMetric: make(map[string]net.IOCountersStat),
	}
}

func (c *Check) Execute(ctx context.Context) error {
	metric, err := c.collectAndSaveTraffic(ctx)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric

	return nil
}

func (c *Check) collectAndSaveTraffic(ctx context.Context) (base.MetricData, error) {
	ioCounters, interfaces, err := c.collectTraffic()
	if err != nil {
		return base.MetricData{}, err
	}

	metric := base.MetricData{
		Type: base.NET,
		Data: c.parseTraffic(ioCounters, interfaces),
	}

	err = c.saveTraffic(metric.Data, ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) collectTraffic() ([]net.IOCountersStat, map[string]net.InterfaceStat, error) {
	ioCounters, err := c.collectIOCounters()
	if err != nil {
		return nil, nil, err
	}

	interfaces, err := c.collectInterfaces()
	if err != nil {
		return nil, nil, err
	}

	return ioCounters, interfaces, nil
}

func (c *Check) parseTraffic(ioCOunters []net.IOCountersStat, interfaces map[string]net.InterfaceStat) []base.CheckResult {
	var data []base.CheckResult
	for _, ioCounter := range ioCOunters {
		if _, ok := interfaces[ioCounter.Name]; ok {
			var inputPps, inputBps, outputPps, outputBps float64

			if lastCounter, exists := c.lastMetric[ioCounter.Name]; exists {
				inputPps, outputPps = utils.CalculatePps(ioCounter, lastCounter, c.GetInterval())
				inputBps, outputBps = utils.CalculateBps(ioCounter, lastCounter, c.GetInterval())
			} else {
				inputPps = 0
				inputBps = 0
				outputPps = 0
				outputBps = 0
			}

			c.lastMetric[ioCounter.Name] = ioCounter
			data = append(data, base.CheckResult{
				Timestamp: time.Now(),
				Name:      ioCounter.Name,
				InputPps:  &inputPps,
				InputBps:  &inputBps,
				OutputPps: &outputPps,
				OutputBps: &outputBps,
			})
		}
	}
	return data
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

func (c *Check) saveTraffic(data []base.CheckResult, ctx context.Context) error {
	client := c.GetClient()
	err := client.Traffic.MapCreateBulk(data, func(q *ent.TrafficCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetName(data[i].Name).
			SetInputPps(*data[i].InputPps).
			SetInputBps(*data[i].InputBps).
			SetOutputPps(*data[i].OutputPps).
			SetOutputBps(*data[i].OutputBps)
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
