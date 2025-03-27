package status

import (
	"context"
	"time"

	"github.com/alpacanetworks/alpamon/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon/pkg/scheduler"
)

const (
	statusURL = "/api/servers/servers/-/status/"
)

type Check struct {
	base.BaseCheck
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
	}
}

func (c *Check) Execute(ctx context.Context) error {
	scheduler.Rqueue.Post(statusURL, nil, 80, time.Time{})

	return nil
}
