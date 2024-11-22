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

type diskUsageQuerySet struct {
	Device     string  `json:"device"`
	MountPoint string  `json:"mount_point"`
	Max        float64 `json:"max"`
	AVG        float64 `json:"avg"`
}

func NewCheck(name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) *Check {
	return &Check{
		BaseCheck: base.NewBaseCheck(name, interval, buffer, client),
	}
}

func (c *Check) Execute(ctx context.Context) {
	queryset, err := c.getDiskUsagePeakAndAvg(ctx)
	metric := base.MetricData{
		Type: base.DISK_USAGE_PER_HOUR,
		Data: []base.CheckResult{},
	}

	if err == nil {
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

func (c *Check) getDiskUsagePeakAndAvg(ctx context.Context) ([]diskUsageQuerySet, error) {
	client := c.GetClient()
	now := time.Now()
	from := now.Add(-1 * time.Hour)

	var queryset []diskUsageQuerySet
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
