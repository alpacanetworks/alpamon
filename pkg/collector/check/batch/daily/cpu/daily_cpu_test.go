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
		Type:     base.CPU_PER_DAY,
		Name:     string(base.CPU_PER_DAY) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   db.InitDB(ctx),
	}

	check := NewCheck(args).(*Check)

	return check
}

func TestGetCPUPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().CPUPerHour.Create().
		SetTimestamp(time.Now()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create cpu usage per hour.")

	querySet, err := check.getCPUPerHour(ctx)
	assert.NoError(t, err, "Failed to get cpu usage per hour.")
	assert.NotEmpty(t, querySet, "CPUPerHour queryset should not be empty")
}

func TestDeleteCPUPerHour(t *testing.T) {
	check := setUp()
	ctx := context.Background()

	err := check.GetClient().CPUPerHour.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetPeak(50.0).
		SetAvg(50.0).Exec(ctx)
	assert.NoError(t, err, "Failed to create cpu usage per hour.")

	err = check.deleteCPUPerHour(ctx)
	assert.NoError(t, err, "Failed to delete cpu usage per hour.")
}
