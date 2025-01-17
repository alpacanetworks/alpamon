package net

import (
	"context"
	"math/rand"
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
		Type:     base.DAILY_NET,
		Name:     string(base.DAILY_NET) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitTestDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetHourlyTraffic(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyTraffic.Create().
		SetTimestamp(time.Now()).
		SetName(uuid.NewString()).
		SetPeakInputPps(rand.Float64()).
		SetPeakInputBps(rand.Float64()).
		SetPeakOutputPps(rand.Float64()).
		SetPeakOutputBps(rand.Float64()).
		SetAvgInputPps(rand.Float64()).
		SetAvgInputBps(rand.Float64()).
		SetAvgOutputPps(rand.Float64()).
		SetAvgOutputBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly traffic.")

	querySet, err := check.getHourlyTraffic(ctx)
	assert.NoError(t, err, "Failed to get hourly traffic.")
	assert.NotEmpty(t, querySet, "HourlyTraffic queryset should not be empty")
}

func TestDeleteHourlyTraffic(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyTraffic.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetName(uuid.NewString()).
		SetPeakInputPps(rand.Float64()).
		SetPeakInputBps(rand.Float64()).
		SetPeakOutputPps(rand.Float64()).
		SetPeakOutputBps(rand.Float64()).
		SetAvgInputPps(rand.Float64()).
		SetAvgInputBps(rand.Float64()).
		SetAvgOutputPps(rand.Float64()).
		SetAvgOutputBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly traffic.")

	err = check.deleteHourlyTraffic(ctx)
	assert.NoError(t, err, "Failed to delete hourly traffic.")
}
