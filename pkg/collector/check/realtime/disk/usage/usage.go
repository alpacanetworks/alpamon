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
	"autofs":   true,
	"devfs":    true,
}

type Check struct {
	base.BaseCheck
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
	}
}

func (c *Check) Execute(ctx context.Context) error {
	metric, err := c.collectAndSaveDiskUsage(ctx)
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

func (c *Check) collectAndSaveDiskUsage(ctx context.Context) (base.MetricData, error) {
	partitions, err := c.collectDiskPartitions()
	if err != nil {
		return base.MetricData{}, err
	}

	metric := base.MetricData{
		Type: base.DISK_USAGE,
		Data: c.parseDiskUsage(partitions),
	}

	err = c.saveDiskUsage(metric.Data, ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) parseDiskUsage(partitions []disk.PartitionStat) []base.CheckResult {
	var data []base.CheckResult
	for _, partition := range partitions {
		usage, err := c.collectDiskUsage(partition.Mountpoint)
		if err == nil {
			data = append(data, base.CheckResult{
				Timestamp:  time.Now(),
				Device:     partition.Device,
				MountPoint: partition.Mountpoint,
				Usage:      usage.UsedPercent,
				Total:      usage.Total,
				Free:       usage.Free,
				Used:       usage.Used,
			})
		}
	}

	return data
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

func (c *Check) saveDiskUsage(data []base.CheckResult, ctx context.Context) error {
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
