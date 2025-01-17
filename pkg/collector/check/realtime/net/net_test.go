package net

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type NetCheckSuite struct {
	suite.Suite
	client *ent.Client
}

func (suite *NetCheckSuite) SetupSuite() {
	suite.client = db.InitTestDB()
}

func setUp(checkType base.CheckType, client *ent.Client) base.CheckStrategy {
	buffer := base.NewCheckBuffer(10)
	args := &base.CheckArgs{
		Type:     checkType,
		Name:     string(checkType) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   client,
	}

	check := NewCheck(args)

	return check
}

func (suite *NetCheckSuite) TestCollectIOCounters() {
	check := setUp(base.NET_COLLECTOR, suite.client).(*CollectCheck)

	ioCounters, err := check.collectIOCounters()
	assert.NoError(suite.T(), err, "Failed to get network IO.")
	assert.NotEmpty(suite.T(), ioCounters, "Network IO should not be empty")
}

func (suite *NetCheckSuite) TestCollectInterfaces() {
	check := setUp(base.NET_COLLECTOR, suite.client).(*CollectCheck)

	interfaces, err := check.collectInterfaces()
	assert.NoError(suite.T(), err, "Failed to get interfaces.")
	assert.NotEmpty(suite.T(), interfaces, "Interfaces should not be empty")
}

func (suite *NetCheckSuite) TestSaveTraffic() {
	check := setUp(base.NET_COLLECTOR, suite.client).(*CollectCheck)
	ctx := context.Background()

	ioCounters, interfaces, err := check.collectTraffic()
	assert.NoError(suite.T(), err, "Failed to get traffic.")
	assert.NotEmpty(suite.T(), ioCounters, "Network IO should not be empty")
	assert.NotEmpty(suite.T(), interfaces, "Interfaces should not be empty")

	data := check.parseTraffic(ioCounters, interfaces)

	err = check.saveTraffic(data, ctx)
	assert.NoError(suite.T(), err, "Failed to save traffic.")
}

func (suite *NetCheckSuite) TestGetTraffic() {
	check := setUp(base.NET, suite.client).(*SendCheck)
	ctx := context.Background()

	err := check.GetClient().Traffic.Create().
		SetTimestamp(time.Now()).
		SetName(uuid.NewString()).
		SetInputPps(rand.Float64()).
		SetInputBps(rand.Float64()).
		SetOutputPps(rand.Float64()).
		SetOutputBps(rand.Float64()).Exec(ctx)
	assert.NoError(suite.T(), err, "Failed to create traffic.")

	querySet, err := check.getTraffic(ctx)
	assert.NoError(suite.T(), err, "Failed to get traffic queryset.")
	assert.NotEmpty(suite.T(), querySet, "Traffic queryset should not be empty")
}

func TestNetCheckSuite(t *testing.T) {
	suite.Run(t, new(NetCheckSuite))
}
