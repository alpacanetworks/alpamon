package diskio

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

func setUp(checkType base.CheckType) base.CheckStrategy {
	buffer := base.NewCheckBuffer(10)
	ctx := context.Background()
	args := &base.CheckArgs{
		Type:     checkType,
		Name:     string(checkType) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(ctx),
	}

	check := NewCheck(args)

	return check
}

func TestCollectDiskIO(t *testing.T) {
	check := setUp(base.DISK_IO_COLLECTOR).(*CollectCheck)

	ioCounters, err := check.collectDiskIO()
	assert.NoError(t, err, "Failed to get disk io.")

	assert.NotEmpty(t, ioCounters, "Disk IO should not be empty")
	for name, ioCounter := range ioCounters {
		assert.NotEmpty(t, name, "Device name should not be empty")
		assert.True(t, ioCounter.ReadBytes > 0, "Read bytes should be non-negative.")
		assert.True(t, ioCounter.WriteBytes > 0, "Write bytes should be non-negative.")
	}
}

func TestSaveDiskIO(t *testing.T) {
	check := setUp(base.DISK_IO_COLLECTOR).(*CollectCheck)
	ctx := context.Background()

	ioCounters, err := check.collectDiskIO()
	assert.NoError(t, err, "Failed to get disk io.")

	data := check.parseDiskIO(ioCounters)

	err = check.saveDiskIO(data, ctx)
	assert.NoError(t, err, "Failed to save cpu usage.")
}

func TestGetDiskIO(t *testing.T) {
	check := setUp(base.DISK_IO).(*SendCheck)
	ctx := context.Background()

	err := check.GetClient().DiskIO.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetReadBps(rand.Float64()).
		SetWriteBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create disk io.")

	querySet, err := check.getDiskIO(ctx)
	assert.NoError(t, err, "Failed to get disk io queryset.")
	assert.NotEmpty(t, querySet, "Disk IO queryset should not be empty")
}
