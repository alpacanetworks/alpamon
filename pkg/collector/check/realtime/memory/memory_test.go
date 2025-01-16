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
		Type:     base.MEM,
		Name:     string(base.MEM) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitTestDB(),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestCollectMemoryUsage(t *testing.T) {
	check := setUp()

	usage, err := check.collectMemoryUsage()

	assert.NoError(t, err, "Failed to get memory usage.")
	assert.GreaterOrEqual(t, usage, 0.0, "Memory usage should be non-negative.")
	assert.LessOrEqual(t, usage, 100.0, "Memory usage should not exceed 100%.")
}

func TestSaveMemoryUsage(t *testing.T) {
	check := setUp()
	ctx := context.Background()
	data := base.CheckResult{
		Timestamp: time.Now(),
		Usage:     50.0,
	}

	err := check.saveMemoryUsage(data, ctx)

	assert.NoError(t, err, "Failed to save memory usage.")
}
