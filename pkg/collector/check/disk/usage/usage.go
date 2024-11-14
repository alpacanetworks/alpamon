package diskusage

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/shirou/gopsutil/v4/disk"
)

const (
	checkType = base.DISK_USAGE
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

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer),
	}
}

func (c *Check) Execute(ctx context.Context) {
	partitions, err := c.collectDiskPartitions()

	metric := base.MetricData{
		Type: checkType,
		Data: []base.CheckResult{},
	}
	if err == nil {
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
