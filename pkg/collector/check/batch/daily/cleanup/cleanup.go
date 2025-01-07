package cpu

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/cpu"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/cpuperhour"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskio"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskioperhour"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskusage"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskusageperhour"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/memory"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/memoryperhour"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/traffic"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/trafficperhour"
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

type deleteQuery func(context.Context, *ent.Client, time.Time) error

type Check struct {
	base.BaseCheck
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
	}
}

func (c *Check) Execute(ctx context.Context) error {
	err := c.deleteAllMetric(ctx)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

func (c *Check) deleteAllMetric(ctx context.Context) error {
	now := time.Now()
	for _, table := range tables {
		if query, exist := deleteQueryMap[table]; exist {
			if err := query(ctx, c.GetClient(), now); err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteAllCPU(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.CPU.Delete().
		Where(cpu.TimestampLTE(now.Add(-1 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllCPUPerHour(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.CPUPerHour.Delete().
		Where(cpuperhour.TimestampLTE(now.Add(-24 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllMemory(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Memory.Delete().
		Where(memory.TimestampLTE(now.Add(-1 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllMemoryPerHour(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.MemoryPerHour.Delete().
		Where(memoryperhour.TimestampLTE(now.Add(-24 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskUsage(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.DiskUsage.Delete().
		Where(diskusage.TimestampLTE(now.Add(-1 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskUsagePerHour(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.DiskUsagePerHour.Delete().
		Where(diskusageperhour.TimestampLTE(now.Add(-24 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskIO(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = client.DiskIO.Delete().
		Where(diskio.TimestampLTE(now.Add(-1 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskIOPerHour(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.DiskIOPerHour.Delete().
		Where(diskioperhour.TimestampLTE(now.Add(-24 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllTraffic(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Traffic.Delete().
		Where(traffic.TimestampLTE(now.Add(-1 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllTrafficPerHour(ctx context.Context, client *ent.Client, now time.Time) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.TrafficPerHour.Delete().
		Where(trafficperhour.TimestampLTE(now.Add(-24 * time.Hour))).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
