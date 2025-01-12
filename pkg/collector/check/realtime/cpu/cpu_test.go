package cpu

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
	ctx := context.Background()
	args := &base.CheckArgs{
		Type:     base.CPU,
		Name:     string(base.CPU) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(ctx),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestCollectCPUUsage(t *testing.T) {
	check := setUp()

	usage, err := check.collectCPUUsage()

	assert.NoError(t, err, "Failed to get cpu usage.")
	assert.GreaterOrEqual(t, usage, 0.0, "CPU usage should be non-negative.")
	assert.LessOrEqual(t, usage, 100.0, "CPU usage should not exceed 100%.")
}

func TestSaveCPUUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()
	data := base.CheckResult{
		Timestamp: time.Now(),
		Usage:     50.0,
	}

	err := check.saveCPUUsage(data, ctx)

	assert.NoError(t, err, "Failed to save cpu usage.")
}
