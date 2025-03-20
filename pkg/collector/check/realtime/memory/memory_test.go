package memory

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MemoryCheckSuite struct {
	suite.Suite
	client *ent.Client
	check  *Check
	ctx    context.Context
}

func (suite *MemoryCheckSuite) SetupSuite() {
	suite.client = db.InitTestDB()
	buffer := base.NewCheckBuffer(10)
	args := &base.CheckArgs{
		Type:     base.MEM,
		Name:     string(base.MEM) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   suite.client,
	}
	suite.check = NewCheck(args).(*Check)
	suite.ctx = context.Background()
}

func (suite *MemoryCheckSuite) TearDownSuite() {
	err := os.Remove("alpamon.db")
	suite.Require().NoError(err, "failed to delete test db file")
}

func (suite *MemoryCheckSuite) TestCollectMemoryUsage() {
	usage, err := suite.check.collectMemoryUsage()

	assert.NoError(suite.T(), err, "Failed to get memory usage.")
	assert.GreaterOrEqual(suite.T(), usage, 0.0, "Memory usage should be non-negative.")
	assert.LessOrEqual(suite.T(), usage, 100.0, "Memory usage should not exceed 100%.")
}

func (suite *MemoryCheckSuite) TestSaveMemoryUsage() {
	data := base.CheckResult{
		Timestamp: time.Now(),
		Usage:     50.0,
	}

	err := suite.check.saveMemoryUsage(data, suite.ctx)

	assert.NoError(suite.T(), err, "Failed to save memory usage.")
}

func TestMemoryCheckSuite(t *testing.T) {
	t.Setenv("GOMAXPROCS", "1")
	suite.Run(t, new(MemoryCheckSuite))
}
