package cpu

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/cpuperhour"
	"github.com/rs/zerolog/log"
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

	queryset, err := c.getCPUPerHour(ctx)
	if err != nil {
		checkError.GetQueryError = err
	}

	metric := base.MetricData{
		Type: base.CPU_PER_DAY,
		Data: []base.CheckResult{},
	}
	if checkError.GetQueryError == nil {
		data := base.CheckResult{
			Timestamp: time.Now(),
			PeakUsage: queryset[0].Max,
			AvgUsage:  queryset[0].AVG,
		}
		metric.Data = append(metric.Data, data)
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	if checkError.GetQueryError != nil {
		buffer.FailureQueue <- metric
	} else {
		buffer.SuccessQueue <- metric
	}
}

func (c *Check) getCPUPerHour(ctx context.Context) ([]base.CPUQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	var queryset []base.CPUQuerySet
	err := client.CPUPerHour.Query().
		Where(cpuperhour.TimestampGTE(from), cpuperhour.TimestampLTE(now)).
		Aggregate(
			ent.Max(cpuperhour.FieldPeakUsage),
			ent.Mean(cpuperhour.FieldAvgUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}