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
	ctx := context.Background()
	args := &base.CheckArgs{
		Type:     base.DISK_IO_PER_HOUR,
		Name:     string(base.DISK_IO_PER_HOUR) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(ctx),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetDiskIO(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskIO.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetReadBps(rand.Float64()).
		SetWriteBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk io.")

	querySet, err := check.getDiskIO(ctx)
	assert.NoError(t, err, "Failed to get disk io.")
	assert.NotEmpty(t, querySet, "Disk io queryset should not be empty")
}

func TestSaveDiskIOPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()
	data := []base.CheckResult{
		{
			Timestamp:    time.Now(),
			Device:       uuid.NewString(),
			PeakWriteBps: rand.Float64(),
			PeakReadBps:  rand.Float64(),
			AvgWriteBps:  rand.Float64(),
			AvgReadBps:   rand.Float64(),
		},
		{
			Timestamp:    time.Now(),
			Device:       uuid.NewString(),
			PeakWriteBps: rand.Float64(),
			PeakReadBps:  rand.Float64(),
			AvgWriteBps:  rand.Float64(),
			AvgReadBps:   rand.Float64(),
		},
	}

	err := check.saveDiskIOPerHour(data, ctx)
	assert.NoError(t, err, "Failed to save disk io per hour.")
}

func TestDeleteDiskIO(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskIO.Create().
		SetTimestamp(time.Now().Add(-2 * time.Hour)).
		SetDevice(uuid.NewString()).
		SetReadBps(rand.Float64()).
		SetWriteBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk io.")

	err = check.deleteDiskIO(ctx)
	assert.NoError(t, err, "Failed to delete disk io.")
}
