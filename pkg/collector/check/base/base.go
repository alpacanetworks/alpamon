package base

import (
	"time"
)

const (
	CPU        CheckType = "cpu"
	MEM        CheckType = "memory"
	DISK_USAGE CheckType = "disk_usage"
	DISK_IO    CheckType = "disk_io"
	NET        CheckType = "net"
)

type CheckType string

type CheckResult struct {
	Timestamp   time.Time `json:"timestamp"`
	Usage       float64   `json:"usage,omitempty"`
	Name        string    `json:"name,omitempty"`
	Device      string    `json:"device,omitempty"`
	Partition   string    `json:"partition,omitempty"`
	Total       uint64    `json:"total,omitempty"`
	Free        uint64    `json:"free,omitempty"`
	Used        uint64    `json:"used,omitempty"`
	WriteBytes  uint64    `json:"wrtie_bytes,omitempty"`
	ReadBytes   uint64    `json:"read_bytes,omitempty"`
	InputPkts   uint64    `json:"input_pkts,omitempty"`
	InputBytes  uint64    `json:"input_bytes,omitempty"`
	OutputPkts  uint64    `json:"output_pkts,omitempty"`
	OutputBytes uint64    `json:"output_bytes,omitempty"`
}

type MetricData struct {
	Type CheckType     `json:"type"`
	Data []CheckResult `json:"data,omitempty"`
}

type BaseCheck struct {
	name     string
	interval time.Duration
	buffer   *CheckBuffer
}

type CheckBuffer struct {
	SuccessQueue chan MetricData
	FailureQueue chan MetricData
	Capacity     int
}

func NewBaseCheck(name string, interval time.Duration, buffer *CheckBuffer) BaseCheck {
	return BaseCheck{
		name:     name,
		interval: interval,
		buffer:   buffer,
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
