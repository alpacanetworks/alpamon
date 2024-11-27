package diskusage

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/shirou/gopsutil/v4/disk"
)

var excludedFileSystems = map[string]bool{
	"tmpfs":    true,
	"devtmpfs": true,
	"proc":     true,
	"sysfs":    true,
	"cgroup":   true,
	"overlay":  true,
}

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

	partitions, err := c.collectDiskPartitions()
	if err != nil {
		checkError.CollectError = err
	}

	metric := base.MetricData{
		Type: base.DISK_USAGE,
		Data: []base.CheckResult{},
	}
	if checkError.CollectError == nil {
		for _, partition := range partitions {
			usage, usageErr := c.collectDiskUsage(partition.Mountpoint)
			if usageErr == nil {
				data := base.CheckResult{
					Timestamp:  time.Now(),
					Device:     partition.Device,
					MountPoint: partition.Mountpoint,
					Usage:      usage.UsedPercent,
					Total:      usage.Total,
					Free:       usage.Free,
					Used:       usage.Used,
				}
				metric.Data = append(metric.Data, data)
			}
		}

		if err := c.saveDiskUsage(ctx, metric.Data); err != nil {
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

func (c *Check) collectDiskPartitions() ([]disk.PartitionStat, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}

	var filteredPartitions []disk.PartitionStat
	for _, partition := range partitions {
		if !excludedFileSystems[partition.Fstype] {
			filteredPartitions = append(filteredPartitions, partition)
		}
	}

	return filteredPartitions, nil
}

func (c *Check) collectDiskUsage(path string) (*disk.UsageStat, error) {
	usage, err := disk.Usage(path)
	if err != nil {
		return nil, err
	}

	return usage, nil
}

func (c *Check) saveDiskUsage(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.DiskUsage.MapCreateBulk(data, func(q *ent.DiskUsageCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetMountPoint(data[i].MountPoint).
			SetUsage(data[i].Usage).
			SetTotal(int64(data[i].Total)).
			SetFree(int64(data[i].Free)).
			SetUsed(int64(data[i].Used))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
