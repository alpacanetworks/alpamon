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
		Type:     base.NET_PER_HOUR,
		Name:     string(base.NET_PER_HOUR) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetTraffic(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().Traffic.Create().
		SetTimestamp(time.Now()).
		SetName(uuid.NewString()).
		SetInputPps(rand.Float64()).
		SetInputBps(rand.Float64()).
		SetOutputPps(rand.Float64()).
		SetOutputBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create traffic.")

	querySet, err := check.getTraffic(ctx)
	assert.NoError(t, err, "Failed to get traffic.")
	assert.NotEmpty(t, querySet, "Traffic queryset should not be empty")
}

func TestSaveTrafficPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()
	data := []base.CheckResult{
		{
			Timestamp:     time.Now(),
			Name:          uuid.NewString(),
			PeakInputPps:  rand.Float64(),
			PeakInputBps:  rand.Float64(),
			AvgInputPps:   rand.Float64(),
			AvgInputBps:   rand.Float64(),
			PeakOutputPps: rand.Float64(),
			PeakOutputBps: rand.Float64(),
			AvgOutputPps:  rand.Float64(),
			AvgOutputBps:  rand.Float64(),
		},
		{
			Timestamp:     time.Now(),
			Name:          uuid.NewString(),
			PeakInputPps:  rand.Float64(),
			PeakInputBps:  rand.Float64(),
			AvgInputPps:   rand.Float64(),
			AvgInputBps:   rand.Float64(),
			PeakOutputPps: rand.Float64(),
			PeakOutputBps: rand.Float64(),
			AvgOutputPps:  rand.Float64(),
			AvgOutputBps:  rand.Float64(),
		},
	}

	err := check.saveTrafficPerHour(data, ctx)
	assert.NoError(t, err, "Failed to save traffic per hour.")
}

func TestDeleteTraffic(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().Traffic.Create().
		SetTimestamp(time.Now().Add(-2 * time.Hour)).
		SetName(uuid.NewString()).
		SetInputPps(rand.Float64()).
		SetInputBps(rand.Float64()).
		SetOutputPps(rand.Float64()).
		SetOutputBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create traffic.")

	err = check.deleteTraffic(ctx)
	assert.NoError(t, err, "Failed to delete traffic.")
}
