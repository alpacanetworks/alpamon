package diskusage

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
		Type:     base.DISK_USAGE,
		Name:     string(base.DISK_USAGE) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitTestDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestCollectDiskPartitions(t *testing.T) {
	check := setUp()

	partitions, err := check.collectDiskPartitions()

	assert.NoError(t, err, "Failed to get disk partitions.")
	assert.NotEmpty(t, partitions, "Disk partitions should not be empty")
}

func TestCollectDiskUsage(t *testing.T) {
	check := setUp()

	partitions, err := check.collectDiskPartitions()
	assert.NoError(t, err, "Failed to get disk partitions.")

	assert.NotEmpty(t, partitions, "Disk partitions should not be empty")
	for _, partition := range partitions {
		usage, err := check.collectDiskUsage(partition.Mountpoint)
		assert.NoError(t, err, "Failed to get disk usage.")
		assert.GreaterOrEqual(t, usage.UsedPercent, 0.0, "Disk usage should be non-negative.")
		assert.LessOrEqual(t, usage.UsedPercent, 100.0, "Disk usage should not exceed 100%.")
	}
}

func TestSaveDiskUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	partitions, err := check.collectDiskPartitions()
	assert.NoError(t, err, "Failed to get disk partitions.")

	err = check.saveDiskUsage(check.parseDiskUsage(partitions), ctx)
	assert.NoError(t, err, "Failed to save disk usage.")
}
