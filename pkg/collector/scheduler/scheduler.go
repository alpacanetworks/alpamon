package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
)

type Scheduler struct {
	tasks    map[string]*ScheduledTask
	mu       sync.RWMutex
	stopChan chan struct{}
}

type ScheduledTask struct {
	check    base.CheckStrategy
	nextRun  time.Time
	interval time.Duration
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:    make(map[string]*ScheduledTask),
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) AddTask(check base.CheckStrategy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	interval := check.GetInterval()
	s.tasks[check.GetName()] = &ScheduledTask{
		check:    check,
		nextRun:  time.Now().Add(interval),
		interval: interval,
	}
	return nil
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.RLock()
			now := time.Now()
			for _, task := range s.tasks {
				if now.After(task.nextRun) {
					go s.executeTask(ctx, task)
				}
			}
			s.mu.RUnlock()
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
}

func (s *Scheduler) executeTask(ctx context.Context, task *ScheduledTask) {
	defer func() {
		s.mu.Lock()
		task.nextRun = time.Now().Add(task.interval)
		s.mu.Unlock()
	}()

	task.check.Execute(ctx)
}
