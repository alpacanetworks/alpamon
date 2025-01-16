package usage

import (
	"context"
	"testing"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setUp() *Check {
	buffer := base.NewCheckBuffer(10)
	args := &base.CheckArgs{
		Type:     base.DAILY_DISK_USAGE,
		Name:     string(base.DAILY_DISK_USAGE) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitTestDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetHourlyDiskUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyDiskUsage.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly disk usage.")

	querySet, err := check.getHourlyDiskUsage(ctx)
	assert.NoError(t, err, "Failed to get hourly disk usage.")
	assert.NotEmpty(t, querySet, "HourlyDiskUsage queryset should not be empty")
}

func TestDeleteHourlyDiskUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyDiskUsage.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetDevice(uuid.NewString()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly disk usage.")

	err = check.deleteHourlyDiskUsage(ctx)
	assert.NoError(t, err, "Failed to delete hourly disk usage.")
}
