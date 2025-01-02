package transporter

import (
	"fmt"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
)

const (
	CPU                 string = "/api/metrics/realtime/cpu/"
	CPU_PER_HOUR        string = "/api/metrics/hourly/cpu/"
	CPU_PER_DAY         string = "/api/metrics/daily/cpu/"
	MEM                 string = "/api/metrics/realtime/memory/"
	MEM_PER_HOUR        string = "/api/metrics/hourly/memory/"
	MEM_PER_DAY         string = "/api/metrics/daily/memory/"
	DISK_USAGE          string = "/api/metrics/realtime/disk-usage/"
	DISK_USAGE_PER_HOUR string = "/api/metrics/hourly/disk-usage/"
	DISK_USAGE_PER_DAY  string = "/api/metrics/daily/disk-usage/"
	DISK_IO             string = "/api/metrics/realtime/disk-io/"
	DISK_IO_PER_HOUR    string = "/api/metrics/hourly/disk-io/"
	DISK_IO_PER_DAY     string = "/api/metrics/daily/disk-io/"
	NET                 string = "/api/metrics/realtime/traffic/"
	NET_PER_HOUR        string = "/api/metrics/hourly/traffic/"
	NET_PER_DAY         string = "/api/metrics/daily/traffic/"
)

type URLResolver struct {
	checkTypeToURL map[base.CheckType]string
}

func NewURLResolver() *URLResolver {
	return &URLResolver{
		checkTypeToURL: map[base.CheckType]string{
			base.CPU:                 CPU,
			base.CPU_PER_HOUR:        CPU_PER_HOUR,
			base.CPU_PER_DAY:         CPU_PER_DAY,
			base.MEM:                 MEM,
			base.MEM_PER_HOUR:        MEM_PER_HOUR,
			base.MEM_PER_DAY:         MEM_PER_DAY,
			base.DISK_USAGE:          DISK_USAGE,
			base.DISK_USAGE_PER_HOUR: DISK_USAGE_PER_HOUR,
			base.DISK_USAGE_PER_DAY:  DISK_USAGE_PER_DAY,
			base.DISK_IO:             DISK_IO,
			base.DISK_IO_PER_HOUR:    DISK_IO_PER_HOUR,
			base.DISK_IO_PER_DAY:     DISK_IO_PER_DAY,
			base.NET:                 NET,
			base.NET_PER_HOUR:        NET_PER_HOUR,
			base.NET_PER_DAY:         NET_PER_DAY,
		},
	}
}

func (r *URLResolver) ResolveURL(checkType base.CheckType) (string, error) {
	url, exists := r.checkTypeToURL[checkType]
	if !exists {
		return "", fmt.Errorf("unknown check type: %s", checkType)
	}

	return url, nil
}
