package memory

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/memory"
)

type Check struct {
	base.BaseCheck
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
	}
}

func (c *Check) Execute(ctx context.Context) error {
	metric, err := c.queryMemoryUsage(ctx)
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

func (c *Check) queryMemoryUsage(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.getMemory(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	data := base.CheckResult{
		Timestamp: time.Now(),
		PeakUsage: queryset[0].Max,
		AvgUsage:  queryset[0].AVG,
	}
	metric := base.MetricData{
		Type: base.MEM_PER_HOUR,
		Data: []base.CheckResult{data},
	}

	err = c.saveMemoryPerHour(data, ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	err = c.deleteMemory(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) getMemory(ctx context.Context) ([]base.MemoryQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []base.MemoryQuerySet
	err := client.Memory.Query().
		Where(memory.TimestampGTE(from), memory.TimestampLTE(now)).
		Aggregate(
			ent.Max(memory.FieldUsage),
			ent.Mean(memory.FieldUsage),
		).Scan(ctx, &queryset)
	if err != nil {
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveMemoryPerHour(data base.CheckResult, ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	err = tx.MemoryPerHour.Create().
		SetTimestamp(data.Timestamp).
		SetPeakUsage(data.PeakUsage).
		SetAvgUsage(data.AvgUsage).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}

func (c *Check) deleteMemory(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err = tx.Memory.Delete().
		Where(memory.TimestampGTE(from), memory.TimestampLTE(now)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
