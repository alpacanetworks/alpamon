package memory

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/memory"
	"github.com/rs/zerolog/log"
)

type Check struct {
	base.BaseCheck
}

type memoryQuerySet struct {
	Max float64
	AVG float64
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	var checkError base.CheckError

	queryset, err := c.getMemoryPeakAndAvg(ctx)
	if err != nil {
		checkError.GetQueryError = err
	}

	metric := base.MetricData{
		Type: base.MEM_PER_HOUR,
		Data: []base.CheckResult{},
	}
	if checkError.GetQueryError == nil {
		data := base.CheckResult{
			Timestamp: time.Now(),
			PeakUsage: queryset[0].Max,
			AvgUsage:  queryset[0].AVG,
		}
		metric.Data = append(metric.Data, data)

		if err := c.saveMemoryPeakAndAvg(ctx, data); err != nil {
			checkError.SaveQueryError = err
		}
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	if checkError.GetQueryError != nil || checkError.SaveQueryError != nil {
		buffer.FailureQueue <- metric
	} else {
		buffer.SuccessQueue <- metric
	}
}

func (c *Check) getMemoryPeakAndAvg(ctx context.Context) ([]memoryQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []memoryQuerySet
	err := client.Memory.Query().
		Where(memory.TimestampGTE(from), memory.TimestampLTE(now)).
		Aggregate(
			ent.Max(memory.FieldUsage),
			ent.Mean(memory.FieldUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveMemoryPeakAndAvg(ctx context.Context, data base.CheckResult) error {
	client := c.GetClient()
	if err := client.MemoryPerHour.Create().
		SetTimestamp(data.Timestamp).
		SetPeakUsage(data.PeakUsage).
		SetAvgUsage(data.AvgUsage).Exec(ctx); err != nil {
		return err
	}

	return nil
}
