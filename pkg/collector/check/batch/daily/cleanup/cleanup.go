package cpu

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
)

var (
	tables = []base.CheckType{
		base.CPU,
		base.CPU_PER_HOUR,
		base.MEM,
		base.MEM_PER_HOUR,
		base.DISK_USAGE,
		base.DISK_USAGE_PER_HOUR,
		base.DISK_IO,
		base.DISK_IO_PER_HOUR,
		base.NET,
		base.NET_PER_HOUR,
	}
	deleteQueryMap = map[base.CheckType]deleteQuery{
		base.CPU:                 deleteAllCPU,
		base.CPU_PER_HOUR:        deleteAllCPUPerHour,
		base.MEM:                 deleteAllMemory,
		base.MEM_PER_HOUR:        deleteAllMemoryPerHour,
		base.DISK_USAGE:          deleteAllDiskUsage,
		base.DISK_USAGE_PER_HOUR: deleteAllDiskUsagePerHour,
		base.DISK_IO:             deleteAllDiskIO,
		base.DISK_IO_PER_HOUR:    deleteAllDiskIOPerHour,
		base.NET:                 deleteAllTraffic,
		base.NET_PER_HOUR:        deleteAllTrafficPerHour,
	}
)

type deleteQuery func(context.Context, *ent.Client) error

type Check struct {
	base.BaseCheck
	retryCount base.RetryCount
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
		retryCount: base.RetryCount{
			MaxDeleteRetries: base.MAX_RETRIES,
			MaxRetryTime:     base.MAX_RETRY_TIMES,
			Delay:            base.DEFAULT_DELAY,
		},
	}
}

func (c *Check) Execute(ctx context.Context) {
	start := time.Now()

	for attempt := 0; attempt <= c.retryCount.MaxDeleteRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		if err := c.deleteAllMetric(ctx); err != nil {
			if attempt < c.retryCount.MaxDeleteRetries {
				backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return
				}
			}
		}
		break
	}

	if ctx.Err() != nil {
		return
	}
}

func (c *Check) deleteAllMetric(ctx context.Context) error {
	for _, table := range tables {
		if query, exist := deleteQueryMap[table]; exist {
			if err := query(ctx, c.GetClient()); err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteAllCPU(ctx context.Context, client *ent.Client) error {
	_, err := client.CPU.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllCPUPerHour(ctx context.Context, client *ent.Client) error {
	_, err := client.CPUPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllMemory(ctx context.Context, client *ent.Client) error {
	_, err := client.Memory.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllMemoryPerHour(ctx context.Context, client *ent.Client) error {
	_, err := client.MemoryPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllDiskUsage(ctx context.Context, client *ent.Client) error {
	_, err := client.DiskUsage.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllDiskUsagePerHour(ctx context.Context, client *ent.Client) error {
	_, err := client.DiskIOPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllDiskIO(ctx context.Context, client *ent.Client) error {
	_, err := client.DiskIO.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllDiskIOPerHour(ctx context.Context, client *ent.Client) error {
	_, err := client.DiskIOPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllTraffic(ctx context.Context, client *ent.Client) error {
	_, err := client.Traffic.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteAllTrafficPerHour(ctx context.Context, client *ent.Client) error {
	_, err := client.TrafficPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
