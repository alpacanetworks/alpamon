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
	ctx := context.Background()
	args := &base.CheckArgs{
		Type:     base.NET_PER_DAY,
		Name:     string(base.NET_PER_DAY) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(ctx),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetTrafficPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().TrafficPerHour.Create().
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
	assert.NoError(t, err, "Failed to create traffic per hour.")

	querySet, err := check.getTrafficPerHour(ctx)
	assert.NoError(t, err, "Failed to get traffic per hour.")
	assert.NotEmpty(t, querySet, "TrafficPerHour queryset should not be empty")
}

func TestDeleteTrafficPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().TrafficPerHour.Create().
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
	assert.NoError(t, err, "Failed to create traffic per hour.")

	err = check.deleteTrafficPerHour(ctx)
	assert.NoError(t, err, "Failed to delete traffic per hour.")
}
