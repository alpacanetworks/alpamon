package base

import (
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
)

const (
	CPU                 CheckType = "cpu"
	CPU_PER_HOUR        CheckType = "cpu_per_hour"
	CPU_PER_DAY         CheckType = "cpu_per_day"
	MEM                 CheckType = "memory"
	MEM_PER_HOUR        CheckType = "memory_per_hour"
	MEM_PER_DAY         CheckType = "memory_per_day"
	DISK_USAGE          CheckType = "disk_usage"
	DISK_USAGE_PER_HOUR CheckType = "disk_usage_per_hour"
	DISK_USAGE_PER_DAY  CheckType = "disk_usage_per_day"
	DISK_IO             CheckType = "disk_io"
	DISK_IO_COLLECTOR   CheckType = "disk_io_collector"
	DISK_IO_PER_HOUR    CheckType = "disk_io_per_hour"
	DISK_IO_PER_DAY     CheckType = "disk_io_per_day"
	NET                 CheckType = "net"
	NET_COLLECTOR       CheckType = "net_collector"
	NET_PER_HOUR        CheckType = "net_per_hour"
	NET_PER_DAY         CheckType = "net_per_day"
	CLEANUP             CheckType = "cleanup"
)

type CheckType string

type CheckArgs struct {
	Type     CheckType
	Name     string
	Interval time.Duration
	Buffer   *CheckBuffer
	Client   *ent.Client
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
	Device       string  `json:"device" db:"device"`
	PeakReadBps  float64 `json:"peak_read_bps"`
	PeakWriteBps float64 `json:"peak_write_bps"`
	AvgReadBps   float64 `json:"avg_read_bps"`
	AvgWriteBps  float64 `json:"avg_write_bps"`
}

type DiskUsageQuerySet struct {
	Device string  `json:"device"`
	Max    float64 `json:"max"`
	AVG    float64 `json:"avg"`
}

type TrafficQuerySet struct {
	Name          string  `json:"name"`
	PeakInputPps  float64 `json:"peak_input_pps"`
	PeakInputBps  float64 `json:"peak_input_bps"`
	PeakOutputPps float64 `json:"peak_output_pps"`
	PeakOutputBps float64 `json:"peak_output_bps"`
	AvgInputPps   float64 `json:"avg_input_pps"`
	AvgInputBps   float64 `json:"avg_input_bps"`
	AvgOutputPps  float64 `json:"avg_output_pps"`
	AvgOutputBps  float64 `json:"avg_output_bps"`
}

type CheckResult struct {
	Timestamp     time.Time `json:"timestamp"`
	Usage         float64   `json:"usage,omitempty"`
	Name          string    `json:"name,omitempty"`
	Device        string    `json:"device,omitempty"`
	MountPoint    string    `json:"mount_point,omitempty"`
	Total         uint64    `json:"total,omitempty"`
	Free          uint64    `json:"free,omitempty"`
	Used          uint64    `json:"used,omitempty"`
	WriteBps      *float64  `json:"write_bps,omitempty"`
	ReadBps       *float64  `json:"read_bps,omitempty"`
	InputPps      *float64  `json:"input_pps,omitempty"`
	InputBps      *float64  `json:"input_bps,omitempty"`
	OutputPps     *float64  `json:"output_pps,omitempty"`
	OutputBps     *float64  `json:"output_bps,omitempty"`
	Peak          float64   `json:"peak,omitempty"`
	Avg           float64   `json:"avg,omitempty"`
	PeakWriteBps  float64   `json:"peak_write_bps,omitempty"`
	PeakReadBps   float64   `json:"peak_read_bps,omitempty"`
	AvgWriteBps   float64   `json:"avg_write_bps,omitempty"`
	AvgReadBps    float64   `json:"avg_read_bps,omitempty"`
	PeakInputPps  float64   `json:"peak_input_pps,omitempty"`
	PeakInputBps  float64   `json:"peak_input_bps,omitempty"`
	PeakOutputPps float64   `json:"peak_output_pps,omitempty"`
	PeakOutputBps float64   `json:"peak_output_bps,omitempty"`
	AvgInputPps   float64   `json:"avg_input_pps,omitempty"`
	AvgInputBps   float64   `json:"avg_input_bps,omitempty"`
	AvgOutputPps  float64   `json:"avg_output_pps,omitempty"`
	AvgOutputBps  float64   `json:"avg_output_bps,omitempty"`
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
