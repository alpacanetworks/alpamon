package diskio

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/shirou/gopsutil/v4/disk"
)

type Check struct {
	base.BaseCheck
	lastMetric map[string]disk.IOCountersStat
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck:  base.NewBaseCheck(args),
		lastMetric: make(map[string]disk.IOCountersStat),
	}
}

func (c *Check) Execute(ctx context.Context) error {
	metric, err := c.collectAndSaveDiskIO(ctx)
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

func (c *Check) collectAndSaveDiskIO(ctx context.Context) (base.MetricData, error) {
	ioCounters, err := c.collectDiskIO()
	if err != nil {
		return base.MetricData{}, err
	}

	metric := base.MetricData{
		Type: base.DISK_IO,
		Data: c.parseDiskIO(ioCounters),
	}

	err = c.saveDiskIO(metric.Data, ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) parseDiskIO(ioCounters map[string]disk.IOCountersStat) []base.CheckResult {
	var data []base.CheckResult
	for name, ioCounter := range ioCounters {
		var readBytes, writeBytes uint64

		if lastCounter, exist := c.lastMetric[name]; exist {
			readBytes = ioCounter.ReadBytes - lastCounter.ReadBytes
			writeBytes = ioCounter.WriteBytes - lastCounter.WriteBytes
		} else {
			readBytes = 0
			writeBytes = 0
		}

		c.lastMetric[name] = ioCounter
		data = append(data, base.CheckResult{
			Timestamp:  time.Now(),
			Device:     name,
			ReadBytes:  &readBytes,
			WriteBytes: &writeBytes,
		})
	}

	return data
}

func (c *Check) collectDiskIO() (map[string]disk.IOCountersStat, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}

	return ioCounters, nil
}

func (c *Check) saveDiskIO(data []base.CheckResult, ctx context.Context) error {
	client := c.GetClient()
	err := client.DiskIO.MapCreateBulk(data, func(q *ent.DiskIOCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetReadBytes(int64(*data[i].ReadBytes)).
			SetWriteBytes(int64(*data[i].WriteBytes))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
