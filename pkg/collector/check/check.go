package check

import (
	"context"
	"fmt"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	cpudaily "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/daily/cpu"
	diskiodaily "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/daily/disk/io"
	diskusagedaily "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/daily/disk/usage"
	memorydaily "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/daily/memory"
	netdaily "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/daily/net"
	cpuhourly "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/hourly/cpu"
	diskiohourly "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/hourly/disk/io"
	diskusagehourly "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/hourly/disk/usage"
	memoryhourly "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/hourly/memory"
	nethourly "github.com/alpacanetworks/alpamon-go/pkg/collector/check/batch/hourly/net"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/realtime/cpu"
	diskio "github.com/alpacanetworks/alpamon-go/pkg/collector/check/realtime/disk/io"
	diskusage "github.com/alpacanetworks/alpamon-go/pkg/collector/check/realtime/disk/usage"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/realtime/memory"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/realtime/net"
)

var checkFactories = map[base.CheckType]newCheck{
	base.CPU:                 cpu.NewCheck,
	base.CPU_PER_HOUR:        cpuhourly.NewCheck,
	base.CPU_PER_DAY:         cpudaily.NewCheck,
	base.MEM:                 memory.NewCheck,
	base.MEM_PER_HOUR:        memoryhourly.NewCheck,
	base.MEM_PER_DAY:         memorydaily.NewCheck,
	base.DISK_USAGE:          diskusage.NewCheck,
	base.DISK_USAGE_PER_HOUR: diskusagehourly.NewCheck,
	base.DISK_USAGE_PER_DAY:  diskusagedaily.NewCheck,
	base.DISK_IO:             diskio.NewCheck,
	base.DISK_IO_PER_HOUR:    diskiohourly.NewCheck,
	base.DISK_IO_PER_DAY:     diskiodaily.NewCheck,
	base.NET:                 net.NewCheck,
	base.NET_PER_HOUR:        nethourly.NewCheck,
	base.NET_PER_DAY:         netdaily.NewCheck,
}

type Check interface {
	Execute(ctx context.Context)
}

type CheckFactory interface {
	CreateCheck(args *base.CheckArgs) (base.CheckStrategy, error)
}

type newCheck func(args *base.CheckArgs) base.CheckStrategy

type DefaultCheckFactory struct{}

func (f *DefaultCheckFactory) CreateCheck(args *base.CheckArgs) (base.CheckStrategy, error) {
	if factory, exists := checkFactories[args.Type]; exists {
		return factory(args), nil
	}

	return nil, fmt.Errorf("unknown check type: %s", args.Type)
}
