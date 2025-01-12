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
		Type:     base.DISK_IO_PER_DAY,
		Name:     string(base.DISK_IO_PER_DAY) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(ctx),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetDiskIOPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskIOPerHour.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetPeakReadBps(rand.Float64()).
		SetPeakWriteBps(rand.Float64()).
		SetAvgReadBps(rand.Float64()).
		SetAvgWriteBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk io per hour.")

	querySet, err := check.getDiskIOPerHour(ctx)
	assert.NoError(t, err, "Failed to get disk io per hour.")
	assert.NotEmpty(t, querySet, "DiskIOPerHour queryset should not be empty")
}

func TestDeleteDiskIOPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskIOPerHour.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetDevice(uuid.NewString()).
		SetPeakReadBps(rand.Float64()).
		SetPeakWriteBps(rand.Float64()).
		SetAvgReadBps(rand.Float64()).
		SetAvgWriteBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk io per hour.")

	err = check.deleteDiskIOPerHour(ctx)
	assert.NoError(t, err, "Failed to delete disk io per hour.")
}
