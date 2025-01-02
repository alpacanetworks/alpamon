package cpu

import (
	"context"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
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
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.CPU.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllCPUPerHour(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.CPUPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllMemory(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Memory.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllMemoryPerHour(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.MemoryPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskUsage(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.DiskUsage.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskUsagePerHour(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.DiskIOPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskIO(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = client.DiskIO.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllDiskIOPerHour(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.DiskIOPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllTraffic(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Traffic.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func deleteAllTrafficPerHour(ctx context.Context, client *ent.Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.TrafficPerHour.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
