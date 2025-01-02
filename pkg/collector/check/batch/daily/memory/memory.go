package memory

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/memoryperhour"
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
	metric, err := c.queryMemoryPerHour(ctx)
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

func (c *Check) queryMemoryPerHour(ctx context.Context) (base.MetricData, error) {
	queryset, err := c.getMemoryPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	data := base.CheckResult{
		Timestamp: time.Now(),
		PeakUsage: queryset[0].Max,
		AvgUsage:  queryset[0].AVG,
	}
	metric := base.MetricData{
		Type: base.MEM_PER_DAY,
		Data: []base.CheckResult{data},
	}

	err = c.deleteMemoryPerHour(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) getMemoryPerHour(ctx context.Context) ([]base.MemoryQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var queryset []base.MemoryQuerySet
	err := client.MemoryPerHour.Query().
		Where(memoryperhour.TimestampGTE(from), memoryperhour.TimestampLTE(now)).
		Aggregate(
			ent.Max(memoryperhour.FieldPeakUsage),
			ent.Mean(memoryperhour.FieldAvgUsage),
		).Scan(ctx, &queryset)
	if err != nil {
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) deleteMemoryPerHour(ctx context.Context) error {
	tx, err := c.GetClient().Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	from := now.Add(-24 * time.Hour)

	_, err = tx.MemoryPerHour.Delete().
		Where(memoryperhour.TimestampGTE(from), memoryperhour.TimestampLTE(now)).Exec(ctx)
	if err != nil {
		return err
	}

	_ = tx.Commit()

	return nil
}
