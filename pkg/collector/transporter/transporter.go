package transporter

import (
	"fmt"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
)

var checkTypeUrlMap = map[base.CheckType]string{
	base.CPU:                 "/api/metrics/realtime/cpu/",
	base.CPU_PER_HOUR:        "/api/metrics/hourly/cpu/",
	base.CPU_PER_DAY:         "/api/metrics/daily/cpu/",
	base.MEM:                 "/api/metrics/realtime/memory/",
	base.MEM_PER_HOUR:        "/api/metrics/hourly/memory/",
	base.MEM_PER_DAY:         "/api/metrics/daily/memory/",
	base.DISK_USAGE:          "/api/metrics/realtime/disk-usage/",
	base.DISK_USAGE_PER_HOUR: "/api/metrics/hourly/disk-usage/",
	base.DISK_USAGE_PER_DAY:  "/api/metrics/daily/disk-usage/",
	base.DISK_IO:             "/api/metrics/realtime/disk-io/",
	base.DISK_IO_PER_HOUR:    "/api/metrics/hourly/disk-io/",
	base.DISK_IO_PER_DAY:     "/api/metrics/daily/disk-io/",
	base.NET:                 "/api/metrics/realtime/traffic/",
	base.NET_PER_HOUR:        "/api/metrics/hourly/traffic/",
	base.NET_PER_DAY:         "/api/metrics/daily/traffic/",
}

type TransportStrategy interface {
	Send(data base.MetricData) error
}

type TransporterFactory interface {
	CreateTransporter(session *scheduler.Session) (TransportStrategy, error)
}

type DefaultTransporterFactory struct{}

type Transporter struct {
	session *scheduler.Session
}

// TODO: Support for various transporters will be required in the future
func (f *DefaultTransporterFactory) CreateTransporter(session *scheduler.Session) (TransportStrategy, error) {
	return NewTransporter(session), nil
}

func NewTransporter(session *scheduler.Session) *Transporter {
	return &Transporter{
		session: session,
	}
}

func (t *Transporter) Send(data base.MetricData) error {
	checkType := data.Type

	var err error
	switch checkType {
	case base.CPU, base.CPU_PER_HOUR, base.CPU_PER_DAY,
		base.MEM, base.MEM_PER_HOUR, base.MEM_PER_DAY:
		_, _, err = t.session.Post(checkTypeUrlMap[checkType], data.Data[0], 10)
	case base.DISK_USAGE, base.DISK_USAGE_PER_HOUR, base.DISK_USAGE_PER_DAY,
		base.DISK_IO, base.DISK_IO_PER_HOUR, base.DISK_IO_PER_DAY,
		base.NET, base.NET_PER_HOUR, base.NET_PER_DAY:
		_, _, err = t.session.Post(checkTypeUrlMap[checkType], data.Data, 10)
	default:
		err = fmt.Errorf("unknown check type: %s", checkType)
	}

	if err != nil {
		return err
	}

	return nil
}
