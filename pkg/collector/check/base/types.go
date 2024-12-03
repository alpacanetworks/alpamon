package base

import (
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
)

const (
	CPU                 CheckType     = "cpu"
	CPU_PER_HOUR        CheckType     = "cpu_per_hour"
	CPU_PER_DAY         CheckType     = "cpu_per_day"
	MEM                 CheckType     = "memory"
	MEM_PER_HOUR        CheckType     = "memory_per_hour"
	MEM_PER_DAY         CheckType     = "memory_per_day"
	DISK_USAGE          CheckType     = "disk_usage"
	DISK_USAGE_PER_HOUR CheckType     = "disk_usage_per_hour"
	DISK_USAGE_PER_DAY  CheckType     = "disk_usage_per_day"
	DISK_IO             CheckType     = "disk_io"
	DISK_IO_PER_HOUR    CheckType     = "disk_io_per_hour"
	DISK_IO_PER_DAY     CheckType     = "disk_io_per_day"
	NET                 CheckType     = "net"
	NET_PER_HOUR        CheckType     = "net_per_hour"
	NET_PER_DAY         CheckType     = "net_per_day"
	CLEANUP             CheckType     = "cleanup"
	MAX_RETRIES         int           = 5
	MAX_RETRY_TIMES     time.Duration = 1 * time.Minute
	COLLECT_MAX_RETRIES int           = 3
	GET_MAX_RETRIES     int           = 2
	SAVE_MAX_RETRIES    int           = 2
	DELETE_MAX_RETRIES  int           = 1
	DEFAULT_DELAY       time.Duration = 1 * time.Second
)

type CheckType string

type CheckArgs struct {
	Type     CheckType
	Name     string
	Interval time.Duration
	Buffer   *CheckBuffer
	Client   *ent.Client
}

type RetryCount struct {
	MaxCollectRetries int
	MaxGetRetries     int
	MaxSaveRetries    int
	MaxDeleteRetries  int
	MaxRetryTime      time.Duration
	Delay             time.Duration
}

type CPUQuerySet struct {
	Max float64
	AVG float64
}

type MemoryQuerySet struct {
	Max float64
	AVG float64
}

type DiskIOQuerySet struct {
	Device         string  `json:"device" db:"device"`
	PeakReadBytes  float64 `json:"peak_read_bytes"`
	PeakWriteBytes float64 `json:"peak_write_bytes"`
	AvgReadBytes   float64 `json:"avg_read_bytes"`
	AvgWriteBytes  float64 `json:"avg_write_bytes"`
}

type DiskUsageQuerySet struct {
	Device     string  `json:"device"`
	MountPoint string  `json:"mount_point"`
	Max        float64 `json:"max"`
	AVG        float64 `json:"avg"`
}

type TrafficQuerySet struct {
	Name            string  `json:"name"`
	PeakInputPkts   float64 `json:"peak_input_pkts"`
	PeakInputBytes  float64 `json:"peak_input_bytes"`
	PeakOutputPkts  float64 `json:"peak_output_pkts"`
	PeakOutputBytes float64 `json:"peak_output_bytes"`
	AvgInputPkts    float64 `json:"avg_input_pkts"`
	AvgInputBytes   float64 `json:"avg_input_bytes"`
	AvgOutputPkts   float64 `json:"avg_output_pkts"`
	AvgOutputBytes  float64 `json:"avg_output_bytes"`
}

type CheckResult struct {
	Timestamp       time.Time `json:"timestamp"`
	Usage           float64   `json:"usage,omitempty"`
	Name            string    `json:"name,omitempty"`
	Device          string    `json:"device,omitempty"`
	MountPoint      string    `json:"mount_point,omitempty"`
	Total           uint64    `json:"total,omitempty"`
	Free            uint64    `json:"free,omitempty"`
	Used            uint64    `json:"used,omitempty"`
	WriteBytes      uint64    `json:"write_bytes,omitempty"`
	ReadBytes       uint64    `json:"read_bytes,omitempty"`
	InputPkts       uint64    `json:"input_pkts,omitempty"`
	InputBytes      uint64    `json:"input_bytes,omitempty"`
	OutputPkts      uint64    `json:"output_pkts,omitempty"`
	OutputBytes     uint64    `json:"output_bytes,omitempty"`
	PeakUsage       float64   `json:"peak_usage,omitempty"`
	AvgUsage        float64   `json:"avg_usage,omitempty"`
	PeakWriteBytes  uint64    `json:"peak_write_bytes,omitempty"`
	PeakReadBytes   uint64    `json:"peak_read_bytes,omitempty"`
	AvgWriteBytes   uint64    `json:"avg_write_bytes,omitempty"`
	AvgReadBytes    uint64    `json:"avg_read_bytes,omitempty"`
	PeakInputPkts   uint64    `json:"peak_input_pkts,omitempty"`
	PeakInputBytes  uint64    `json:"peak_input_bytes,omitempty"`
	PeakOutputPkts  uint64    `json:"peak_output_pkts,omitempty"`
	PeakOutputBytes uint64    `json:"peak_output_bytes,omitempty"`
	AvgInputPkts    uint64    `json:"avg_input_pkts,omitempty"`
	AvgInputBytes   uint64    `json:"avg_input_bytes,omitempty"`
	AvgOutputPkts   uint64    `json:"avg_output_pkts,omitempty"`
	AvgOutputBytes  uint64    `json:"avg_output_bytes,omitempty"`
}

type MetricData struct {
	Type CheckType     `json:"type"`
	Data []CheckResult `json:"data,omitempty"`
}

type CheckBuffer struct {
	SuccessQueue chan MetricData
	FailureQueue chan MetricData
	Capacity     int
}
