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
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	var checkError base.CheckError

	ioCounters, err := c.collectDiskIO()
	if err != nil {
		checkError.CollectError = err
	}

	metric := base.MetricData{
		Type: base.DISK_IO,
		Data: []base.CheckResult{},
	}
	if checkError.CollectError == nil {
		for name, ioCounter := range ioCounters {
			data := base.CheckResult{
				Timestamp:  time.Now(),
				Device:     name,
				ReadBytes:  ioCounter.ReadBytes,
				WriteBytes: ioCounter.WriteBytes,
			}
			metric.Data = append(metric.Data, data)
		}

		if err := c.saveDiskIO(ctx, metric.Data); err != nil {
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

func (c *Check) collectDiskIO() (map[string]disk.IOCountersStat, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}

	return ioCounters, nil
}

func (c *Check) saveDiskIO(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.DiskIO.MapCreateBulk(data, func(q *ent.DiskIOCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetReadBytes(int64(data[i].ReadBytes)).
			SetWriteBytes(int64(data[i].WriteBytes))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
