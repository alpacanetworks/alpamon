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
		Type:     base.DISK_USAGE_PER_DAY,
		Name:     string(base.DISK_USAGE_PER_DAY) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetDiskUsagePerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskUsagePerHour.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk usage per hour.")

	querySet, err := check.getDiskUsagePerHour(ctx)
	assert.NoError(t, err, "Failed to get disk usage per hour.")
	assert.NotEmpty(t, querySet, "DiskUsagePerHour queryset should not be empty")
}

func TestDeleteDiskUsagePerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().DiskUsagePerHour.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetDevice(uuid.NewString()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk usage per hour.")

	err = check.deleteDiskUsagePerHour(ctx)
	assert.NoError(t, err, "Failed to delete disk usage per hour.")
}
