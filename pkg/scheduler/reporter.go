package scheduler

import (
	"encoding/json"
	"fmt"
	"github.com/alpacanetworks/alpamon-go/pkg/config"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/alpacanetworks/alpamon-go/pkg/version"
	"github.com/rs/zerolog/log"
	"math"
	"net/http"
	"sync"
	"time"
)

const (
	startUpEventURL = "/api/events/events/"
)

func NewReporter(index int, session *Session) *Reporter {
	return &Reporter{
		name:    fmt.Sprintf("Reporter-%d", index),
		session: session,
		counters: &counters{
			success: 0,
			failure: 0,
			ignored: 0,
			delay:   0.0,
			latency: 0.0,
		},
	}
}

func StartReporters(session *Session) {
	newRequestQueue() // init RequestQueue

	wg := sync.WaitGroup{}
	for i := 0; i < config.GlobalSettings.HTTPThreads; i++ {
		wg.Add(1)
		reporter := NewReporter(i, session)
		go func() {
			defer wg.Done()
			reporter.Run()
		}()
	}

	reportStartupEvent()
}

func reportStartupEvent() {
	eventData, _ := json.Marshal(map[string]string{
		"reporter":    "alpamon",
		"record":      "started",
		"description": fmt.Sprintf("alpamon-go %s started running.", version.Version),
	})

	Rqueue.Post(startUpEventURL, eventData, 10, time.Time{})
}

func (r *Reporter) query(entry PriorityEntry) {
	t1 := time.Now()
	resp, statusCode, err := r.session.Request(entry.method, entry.url, entry.data, 5)
	t2 := time.Now()

	r.counters.delay = r.counters.delay*0.9 + (t2.Sub(entry.due).Seconds())*0.1
	r.counters.latency = r.counters.latency*0.9 + (t2.Sub(t1).Seconds())*0.1

	var success bool
	if err != nil {
		log.Error().Err(err).Msgf("%s %s", entry.method, entry.url)
		success = false
	} else if utils.IsSuccessStatusCode(statusCode) {
		success = true
	} else {
		if statusCode == http.StatusBadRequest {
			log.Error().Err(err).Msgf("%d Bad Request: %s", statusCode, resp)
		} else {
			log.Debug().Msgf("%s %s Error: %d %s", entry.method, entry.url, statusCode, resp)
		}
		success = false
	}

	if success {
		r.counters.success++
	} else {
		r.counters.failure++
		if entry.retry > 0 {
			backoff := time.Duration(math.Pow(2, float64(RetryLimit-entry.retry))) * time.Second
			entry.due = entry.due.Add(backoff)
			entry.retry--
			err = Rqueue.queue.Offer(entry)
			if err != nil {
				r.counters.ignored++
				time.Sleep(1 * time.Second)
			}
		} else {
			r.counters.ignored++
		}
	}
}

func (r *Reporter) Run() {
	for {
		Rqueue.cond.L.Lock()
		for Rqueue.queue.Size() == 0 {
			Rqueue.cond.Wait()
		}
		entry, err := Rqueue.queue.Get()
		Rqueue.cond.L.Unlock()
		if err != nil {
			continue
		}

		if !entry.expiry.IsZero() && entry.expiry.Before(time.Now()) {
			r.counters.ignored++
		} else if !entry.due.IsZero() && entry.due.After(time.Now()) {
			err = Rqueue.queue.Offer(entry)
			if err != nil {
				r.counters.ignored++
				time.Sleep(1 * time.Second)
			}
		} else {
			r.query(entry)
		}
	}
}

// TODO : GetReporterStats
func GetReporterStats() {}
