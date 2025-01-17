package io

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
		Type:     base.DAILY_DISK_IO,
		Name:     string(base.DAILY_DISK_IO) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitTestDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetHourlyDiskIO(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyDiskIO.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetPeakReadBps(rand.Float64()).
		SetPeakWriteBps(rand.Float64()).
		SetAvgReadBps(rand.Float64()).
		SetAvgWriteBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly disk io.")

	querySet, err := check.getHourlyDiskIO(ctx)
	assert.NoError(t, err, "Failed to get hourly disk io.")
	assert.NotEmpty(t, querySet, "HourlyDiskIO queryset should not be empty")
}

func TestDeleteHourlyDiskIO(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().HourlyDiskIO.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetDevice(uuid.NewString()).
		SetPeakReadBps(rand.Float64()).
		SetPeakWriteBps(rand.Float64()).
		SetAvgReadBps(rand.Float64()).
		SetAvgWriteBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create hourly disk io.")

	err = check.deleteHourlyDiskIO(ctx)
	assert.NoError(t, err, "Failed to delete hourly disk io.")
}
