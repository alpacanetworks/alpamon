package net

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

func TestCollectIOCounters(t *testing.T) {
	check := setUp(base.NET_COLLECTOR).(*CollectCheck)

	ioCounters, err := check.collectIOCounters()
	assert.NoError(t, err, "Failed to get network IO.")
	assert.NotEmpty(t, ioCounters, "Network IO should not be empty")
}

func TestCollectInterfaces(t *testing.T) {
	check := setUp(base.NET_COLLECTOR).(*CollectCheck)

	interfaces, err := check.collectInterfaces()
	assert.NoError(t, err, "Failed to get interfaces.")
	assert.NotEmpty(t, interfaces, "Interfaces should not be empty")
}

func TestSaveTraffic(t *testing.T) {
	check := setUp(base.NET_COLLECTOR).(*CollectCheck)
	ctx := context.Background()

	ioCounters, interfaces, err := check.collectTraffic()
	assert.NoError(t, err, "Failed to get traffic.")
	assert.NotEmpty(t, ioCounters, "Network IO should not be empty")
	assert.NotEmpty(t, interfaces, "Interfaces should not be empty")

	data := check.parseTraffic(ioCounters, interfaces)

	err = check.saveTraffic(data, ctx)
	assert.NoError(t, err, "Failed to save traffic.")
}

func TestGetTraffic(t *testing.T) {
	check := setUp(base.NET).(*SendCheck)
	ctx := context.Background()

	err := check.GetClient().Traffic.Create().
		SetTimestamp(time.Now()).
		SetName(uuid.NewString()).
		SetInputPps(rand.Float64()).
		SetInputBps(rand.Float64()).
		SetOutputPps(rand.Float64()).
		SetOutputBps(rand.Float64()).Exec(ctx)
	assert.NoError(t, err, "Failed to create traffic.")

	querySet, err := check.getTraffic(ctx)
	assert.NoError(t, err, "Failed to get traffic queryset.")
	assert.NotEmpty(t, querySet, "Traffic queryset should not be empty")
}
