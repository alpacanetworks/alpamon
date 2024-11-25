package check

import (
	"context"
	"fmt"
	"time"

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
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
)

type CheckStrategy interface {
	Execute(ctx context.Context)
	GetInterval() time.Duration
	GetName() string
	GetBuffer() *base.CheckBuffer
	GetClient() *ent.Client
}

type CheckFactory interface {
	CreateCheck(checkType base.CheckType, name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) (CheckStrategy, error)
}

type DefaultCheckFactory struct{}

func (f *DefaultCheckFactory) CreateCheck(checkType base.CheckType, name string, interval time.Duration, buffer *base.CheckBuffer, client *ent.Client) (CheckStrategy, error) {
	switch checkType {
	case base.CPU:
		return cpu.NewCheck(name, interval, buffer, client), nil
	case base.CPU_PER_HOUR:
		return cpuhourly.NewCheck(name, interval, buffer, client), nil
	case base.CPU_PER_DAY:
		return cpudaily.NewCheck(name, interval, buffer, client), nil
	case base.MEM:
		return memory.NewCheck(name, interval, buffer, client), nil
	case base.MEM_PER_HOUR:
		return memoryhourly.NewCheck(name, interval, buffer, client), nil
	case base.MEM_PER_DAY:
		return memorydaily.NewCheck(name, interval, buffer, client), nil
	case base.DISK_USAGE:
		return diskusage.NewCheck(name, interval, buffer, client), nil
	case base.DISK_USAGE_PER_HOUR:
		return diskusagehourly.NewCheck(name, interval, buffer, client), nil
	case base.DISK_USAGE_PER_DAY:
		return diskusagedaily.NewCheck(name, interval, buffer, client), nil
	case base.DISK_IO:
		return diskio.NewCheck(name, interval, buffer, client), nil
	case base.DISK_IO_PER_HOUR:
		return diskiohourly.NewCheck(name, interval, buffer, client), nil
	case base.DISK_IO_PER_DAY:
		return diskiodaily.NewCheck(name, interval, buffer, client), nil
	case base.NET:
		return net.NewCheck(name, interval, buffer, client), nil
	case base.NET_PER_HOUR:
		return nethourly.NewCheck(name, interval, buffer, client), nil
	case base.NET_PER_DAY:
		return netdaily.NewCheck(name, interval, buffer, client), nil
	default:
		return nil, fmt.Errorf("unknown check type: %s", checkType)
	}
}
