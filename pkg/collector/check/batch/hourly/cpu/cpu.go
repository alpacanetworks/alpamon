package cpu

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/cpu"
	"github.com/rs/zerolog/log"
)

type Check struct {
	base.BaseCheck
}

type cpuQuerySet struct {
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

	queryset, err := c.getCPUPeakAndAvg(ctx)
	if err != nil {
		checkError.GetQueryError = err
	}

	metric := base.MetricData{
		Type: base.CPU_PER_HOUR,
		Data: []base.CheckResult{},
	}
	if checkError.GetQueryError == nil {
		data := base.CheckResult{
			Timestamp: time.Now(),
			PeakUsage: queryset[0].Max,
			AvgUsage:  queryset[0].AVG,
		}
		metric.Data = append(metric.Data, data)

		if err := c.saveCPUPeakAndAvg(ctx, data); err != nil {
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

func (c *Check) getCPUPeakAndAvg(ctx context.Context) ([]cpuQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []cpuQuerySet
	err := client.CPU.Query().
		Where(cpu.TimestampGTE(from), cpu.TimestampLTE(now)).
		Aggregate(
			ent.Max(cpu.FieldUsage),
			ent.Mean(cpu.FieldUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveCPUPeakAndAvg(ctx context.Context, data base.CheckResult) error {
	client := c.GetClient()
	if err := client.CPUPerHour.Create().
		SetTimestamp(data.Timestamp).
		SetPeakUsage(data.PeakUsage).
		SetAvgUsage(data.AvgUsage).Exec(ctx); err != nil {
		return err
	}

	return nil
}
