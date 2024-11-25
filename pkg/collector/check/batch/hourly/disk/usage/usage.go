package usage

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/diskusage"
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

	queryset, err := c.getDiskUsage(ctx)
	if err != nil {
		checkError.GetQueryError = err
	}

	metric := base.MetricData{
		Type: base.DISK_USAGE_PER_HOUR,
		Data: []base.CheckResult{},
	}
	if checkError.GetQueryError == nil {
		for _, row := range queryset {
			data := base.CheckResult{
				Timestamp:  time.Now(),
				Device:     row.Device,
				MountPoint: row.MountPoint,
				PeakUsage:  row.Max,
				AvgUsage:   row.AVG,
			}
			metric.Data = append(metric.Data, data)
		}

		if err := c.saveDiskUsagePerHour(ctx, metric.Data); err != nil {
			checkError.SaveQueryError = err
		}

		if err := c.deleteDiskUsage(ctx); err != nil {
			checkError.DeleteQueryError = err
		}
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	isFailed := checkError.GetQueryError != nil ||
		checkError.SaveQueryError != nil ||
		checkError.DeleteQueryError != nil
	if isFailed {
		buffer.FailureQueue <- metric
	} else {
		buffer.SuccessQueue <- metric
	}
}

func (c *Check) getDiskUsage(ctx context.Context) ([]base.DiskUsageQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []base.DiskUsageQuerySet
	err := client.DiskUsage.Query().
		Where(diskusage.TimestampGTE(from), diskusage.TimestampLTE(now)).
		GroupBy(diskusage.FieldDevice, diskusage.FieldMountPoint).
		Aggregate(
			ent.Max(diskusage.FieldUsage),
			ent.Mean(diskusage.FieldUsage),
		).
		Scan(ctx, &queryset)
	if err != nil {
		log.Debug().Msg(err.Error())
		return queryset, err
	}

	return queryset, nil
}

func (c *Check) saveDiskUsagePerHour(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.DiskUsagePerHour.MapCreateBulk(data, func(q *ent.DiskUsagePerHourCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetDevice(data[i].Device).
			SetMountPoint(data[i].MountPoint).
			SetPeakUsage(data[i].PeakUsage).
			SetAvgUsage(data[i].AvgUsage)
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Check) deleteDiskUsage(ctx context.Context) error {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	_, err := client.DiskUsage.Delete().
		Where(diskusage.TimestampGTE(from), diskusage.TimestampLTE(now)).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
