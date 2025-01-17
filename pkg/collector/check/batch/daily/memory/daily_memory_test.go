package memory

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
		Type:     base.DAILY_MEM_USAGE,
		Name:     string(base.DAILY_MEM_USAGE) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitTestDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetHourlyMemoryUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyMemoryUsage.Create().
		SetTimestamp(time.Now()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly memory usage.")

	querySet, err := check.getHourlyMemoryUsage(ctx)
	assert.NoError(t, err, "Failed to get hourly memory usage.")
	assert.NotEmpty(t, querySet, "HouryMemoryUsage queryset should not be empty")
}

func TestDeleteHourlyMemoryUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyMemoryUsage.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly memory usage.")

	err = check.deleteHourlyMemoryUsage(ctx)
	assert.NoError(t, err, "Failed to delete hourly memory usage.")
}
