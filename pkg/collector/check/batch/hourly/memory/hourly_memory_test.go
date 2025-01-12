package memory

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
		Type:     base.MEM_PER_HOUR,
		Name:     string(base.MEM_PER_HOUR) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(ctx),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetMemory(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().CPU.Create().
		SetTimestamp(time.Now()).
		SetUsage(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create memory usage.")

	querySet, err := check.getMemory(ctx)
	assert.NoError(t, err, "Failed to get memory usage.")
	assert.NotEmpty(t, querySet, "Memory queryset should not be empty")
}

func TestSaveMemoryPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()
	data := base.CheckResult{
		Timestamp: time.Now(),
		Peak:      50.0,
		Avg:       50.0,
	}

	err := check.saveMemoryPerHour(data, ctx)
	assert.NoError(t, err, "Failed to save memory usage per hour.")
}

func TestDeleteMemory(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().Memory.Create().
		SetTimestamp(time.Now().Add(-2 * time.Hour)).
		SetUsage(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create memory usage.")

	err = check.deleteMemory(ctx)
	assert.NoError(t, err, "Failed to delete memory usage.")
}
