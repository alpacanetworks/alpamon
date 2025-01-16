package cpu

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
		Type:     base.DAILY_CPU_USAGE,
		Name:     string(base.DAILY_CPU_USAGE) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetHourlyCPUUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyCPUUsage.Create().
		SetTimestamp(time.Now()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly cpu usage.")

	querySet, err := check.getHourlyCPUUsage(ctx)
	assert.NoError(t, err, "Failed to get hourly cpu usage.")
	assert.NotEmpty(t, querySet, "HourlyCPUUsage queryset should not be empty")
}

func TestDeleteHourlyCPUUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyCPUUsage.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly cpu usage.")

	err = check.deleteHourlyCPUUsage(ctx)
	assert.NoError(t, err, "Failed to delete hourly cpu usage.")
}
