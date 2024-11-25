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
	DISK_IO_PER_HOUR    CheckType = "disk_io_per_hour"
	DISK_IO_PER_DAY     CheckType = "disk_io_per_day"
	NET                 CheckType = "net"
	NET_PER_HOUR        CheckType = "net_per_hour"
	NET_PER_DAY         CheckType = "net_per_day"
)

type CheckType string

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

type CheckError struct {
	CollectError     error
	GetQueryError    error
	SaveQueryError   error
	DeleteQueryError error
}

type MetricData struct {
	Type CheckType     `json:"type"`
	Data []CheckResult `json:"data,omitempty"`
}

type BaseCheck struct {
	name     string
	interval time.Duration
	buffer   *CheckBuffer
	client   *ent.Client
}

type CheckBuffer struct {
	SuccessQueue chan MetricData
	FailureQueue chan MetricData
	Capacity     int
}

func NewBaseCheck(name string, interval time.Duration, buffer *CheckBuffer, client *ent.Client) BaseCheck {
	return BaseCheck{
		name:     name,
		interval: interval,
		buffer:   buffer,
		client:   client,
	}
}

func NewCheckBuffer(capacity int) *CheckBuffer {
	return &CheckBuffer{
		SuccessQueue: make(chan MetricData, capacity),
		FailureQueue: make(chan MetricData, capacity),
		Capacity:     capacity,
	}
}

func (c *BaseCheck) GetName() string {
	return c.name
}

func (c *BaseCheck) GetInterval() time.Duration {
	return c.interval
}

func (c *BaseCheck) GetBuffer() *CheckBuffer {
	return c.buffer
}

func (c *BaseCheck) GetClient() *ent.Client {
	return c.client
}
