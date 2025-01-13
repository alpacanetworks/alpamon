package memory

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
		Type:     base.MEM_PER_DAY,
		Name:     string(base.MEM_PER_DAY) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetMemoryPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().MemoryPerHour.Create().
		SetTimestamp(time.Now()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create memory usage per hour.")

	querySet, err := check.getMemoryPerHour(ctx)
	assert.NoError(t, err, "Failed to get memory usage per hour.")
	assert.NotEmpty(t, querySet, "MemoryPerHour queryset should not be empty")
}

func TestDeleteMemoryPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().MemoryPerHour.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create memory usage per hour.")

	err = check.deleteMemoryPerHour(ctx)
	assert.NoError(t, err, "Failed to delete memory usage per hour.")
}
