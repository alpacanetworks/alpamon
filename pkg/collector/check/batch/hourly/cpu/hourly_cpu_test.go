package cpu

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
		Type:     base.CPU_PER_HOUR,
		Name:     string(base.CPU_PER_HOUR) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetCPU(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().CPU.Create().
		SetTimestamp(time.Now()).
		SetUsage(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create cpu usage.")

	querySet, err := check.getCPU(ctx)
	assert.NoError(t, err, "Failed to get cpu usage.")
	assert.NotEmpty(t, querySet, "CPU queryset should not be empty")
}

func TestSaveCPUPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()
	data := base.CheckResult{
		Timestamp: time.Now(),
		Peak:      50.0,
		Avg:       50.0,
	}

	err := check.saveCPUPerHour(data, ctx)
	assert.NoError(t, err, "Failed to save cpu usage per hour.")
}

func TestDeleteCPU(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().CPU.Create().
		SetTimestamp(time.Now().Add(-2 * time.Hour)).
		SetUsage(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create cpu usage.")

	err = check.deleteCPU(ctx)
	assert.NoError(t, err, "Failed to delete cpu usage.")
}
