package usage

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
		Type:     base.HOURLY_DISK_USAGE,
		Name:     string(base.HOURLY_DISK_USAGE) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetDiskUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskUsage.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetMountPoint(uuid.NewString()).
		SetUsage(rand.Float64()).
		SetTotal(int64(rand.Int())).
		SetFree(int64(rand.Int())).
		SetUsed(int64(rand.Int())).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk usage.")

	querySet, err := check.getDiskUsage(ctx)
	assert.NoError(t, err, "Failed to get disk usage.")
	assert.NotEmpty(t, querySet, "Disk usage queryset should not be empty")
}

func TestSaveHourlyDiskUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()
	data := []base.CheckResult{
		{
			Timestamp: time.Now(),
			Device:    uuid.NewString(),
			Peak:      50.0,
			Avg:       50.0,
		},
		{
			Timestamp: time.Now(),
			Device:    uuid.NewString(),
			Peak:      50.0,
			Avg:       50.0,
		},
	}

	err := check.saveHourlyDiskUsage(data, ctx)
	assert.NoError(t, err, "Failed to save hourly disk usage.")
}

func TestDeleteDiskUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskUsage.Create().
		SetTimestamp(time.Now().Add(-2 * time.Hour)).
		SetDevice(uuid.NewString()).
		SetMountPoint(uuid.NewString()).
		SetUsage(rand.Float64()).
		SetTotal(int64(rand.Int())).
		SetFree(int64(rand.Int())).
		SetUsed(int64(rand.Int())).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk usage.")

	err = check.deleteDiskUsage(ctx)
	assert.NoError(t, err, "Failed to delete disk usage.")
}
