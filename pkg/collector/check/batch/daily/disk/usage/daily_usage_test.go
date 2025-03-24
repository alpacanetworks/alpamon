package usage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alpacanetworks/alpamon/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon/pkg/db"
	"github.com/alpacanetworks/alpamon/pkg/db/ent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DailyDiskUsageCheckSuite struct {
	suite.Suite
	client *ent.Client
	check  *Check
	ctx    context.Context
}

func (suite *DailyDiskUsageCheckSuite) SetupSuite() {
	suite.client = db.InitTestDB()
	buffer := base.NewCheckBuffer(10)
	args := &base.CheckArgs{
		Type:     base.DAILY_DISK_USAGE,
		Name:     string(base.DAILY_DISK_USAGE) + "_" + uuid.NewString(),
		Interval: time.Duration(1 * time.Second),
		Buffer:   buffer,
		Client:   suite.client,
	}
	suite.check = NewCheck(args).(*Check)
	suite.ctx = context.Background()
}

func (suite *DailyDiskUsageCheckSuite) TearDownSuite() {
	err := os.Remove("alpamon.db")
	suite.Require().NoError(err, "failed to delete test db file")
}

func (suite *DailyDiskUsageCheckSuite) TestGetHourlyDiskUsage() {
	err := suite.check.GetClient().HourlyDiskUsage.Create().
		SetTimestamp(time.Now()).
		SetDevice(uuid.NewString()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(suite.ctx)
	assert.NoError(suite.T(), err, "Failed to create hourly disk usage.")

	querySet, err := suite.check.getHourlyDiskUsage(suite.ctx)
	assert.NoError(suite.T(), err, "Failed to get hourly disk usage.")
	assert.NotEmpty(suite.T(), querySet, "HourlyDiskUsage queryset should not be empty")
}

func (suite *DailyDiskUsageCheckSuite) TestDeleteHourlyDiskUsage() {
	err := suite.check.GetClient().HourlyDiskUsage.Create().
		SetTimestamp(time.Now().Add(-25 * time.Hour)).
		SetDevice(uuid.NewString()).
		SetPeak(50.0).
		SetAvg(50.0).Exec(suite.ctx)
	assert.NoError(suite.T(), err, "Failed to create hourly disk usage.")

	err = suite.check.deleteHourlyDiskUsage(suite.ctx)
	assert.NoError(suite.T(), err, "Failed to delete hourly disk usage.")
}

func TestDailyDiskUsageCheckSuite(t *testing.T) {
	t.Setenv("GOMAXPROCS", "1")
	suite.Run(t, new(DailyDiskUsageCheckSuite))
}
