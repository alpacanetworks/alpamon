package transporter

import (
	"fmt"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/scheduler"
)

var checkTypeUrlMap = map[base.CheckType]string{
	base.CPU:        "/api/metrics/cpu/",
	base.MEM:        "/api/metrics/memory/",
	base.DISK_USAGE: "/api/metrics/disk-usage/",
	base.DISK_IO:    "/api/metrics/disk-io/",
	base.NET:        "/api/metrics/traffic/",
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
	case base.CPU:
		_, _, err = t.session.Post(checkTypeUrlMap[base.CPU], data.Data[0], 10)
	case base.MEM:
		_, _, err = t.session.Post(checkTypeUrlMap[base.MEM], data.Data[0], 10)
	case base.DISK_USAGE:
		_, _, err = t.session.Post(checkTypeUrlMap[base.DISK_USAGE], data.Data, 10)
	case base.DISK_IO:
		_, _, err = t.session.Post(checkTypeUrlMap[base.DISK_IO], data.Data, 10)
	case base.NET:
		_, _, err = t.session.Post(checkTypeUrlMap[base.NET], data.Data, 10)
	default:
		err = fmt.Errorf("unknown check type: %s", checkType)
	}

	if err != nil {
		return err
	}

	return nil
}
