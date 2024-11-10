package check

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/cpu"
	diskio "github.com/alpacanetworks/alpamon-go/pkg/collector/check/disk/io"
	diskusage "github.com/alpacanetworks/alpamon-go/pkg/collector/check/disk/usage"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/memory"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/net"
)

type CheckStrategy interface {
	Execute(ctx context.Context)
	GetInterval() time.Duration
	GetName() string
	GetBuffer() *base.CheckBuffer
}

type CheckFactory interface {
	CreateCheck(checkType base.CheckType, name string, interval time.Duration, buffer *base.CheckBuffer) (CheckStrategy, error)
}

type DefaultCheckFactory struct{}

func (f *DefaultCheckFactory) CreateCheck(checkType base.CheckType, name string, interval time.Duration, buffer *base.CheckBuffer) (CheckStrategy, error) {
	switch checkType {
	case base.CPU:
		return cpu.NewCheck(name, interval, buffer), nil
	case base.MEM:
		return memory.NewCheck(name, interval, buffer), nil
	case base.DISK_USAGE:
		return diskusage.NewCheck(name, interval, buffer), nil
	case base.DISK_IO:
		return diskio.NewCheck(name, interval, buffer), nil
	case base.NET:
		return net.NewCheck(name, interval, buffer), nil
	default:
		return nil, fmt.Errorf("unknown check type: %s", checkType)
	}
}
