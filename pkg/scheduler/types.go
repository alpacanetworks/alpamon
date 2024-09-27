package scheduler

import (
	"github.com/adrianbrad/queue"
	"github.com/hashicorp/go-retryablehttp"
	"sync"
	"time"
)

type Session struct {
	BaseURL       string
	Client        *retryablehttp.Client
	authorization string
}

// queue //
type PriorityEntry struct {
	priority int
	method   string
	url      string
	data     interface{}
	due      time.Time
	expiry   time.Time
	retry    int
}

type RequestQueue struct {
	queue *queue.Priority[PriorityEntry]
	cond  *sync.Cond
}

// reporter //
type Reporter struct {
	name     string
	session  *Session
	counters *counters
}

type counters struct {
	success int
	failure int
	ignored int
	delay   float64
	latency float64
}
