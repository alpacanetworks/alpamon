package net

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/shirou/gopsutil/v4/net"
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

	ioCounters, err := c.collectIOCounters()
	interfaces, _ := c.collectInterfaces()
	if err != nil {
		checkError.CollectError = err
	}

	metric := base.MetricData{
		Type: base.NET,
		Data: []base.CheckResult{},
	}
	if checkError.CollectError == nil {
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

		if err := c.saveTraffic(ctx, metric.Data); err != nil {
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

func (c *Check) saveTraffic(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.Traffic.MapCreateBulk(data, func(q *ent.TrafficCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetName(data[i].Name).
			SetInputPkts(int64(data[i].InputPkts)).
			SetInputBytes(int64(data[i].InputBytes)).
			SetOutputPkts(int64(data[i].OutputPkts)).
			SetOutputBytes(int64(data[i].OutputBytes))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
