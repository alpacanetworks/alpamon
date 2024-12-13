package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
)

const (
	MAX_RETRIES     int           = 5
	MAX_RETRY_TIMES time.Duration = 1 * time.Minute
	DEFAULT_DELAY   time.Duration = 1 * time.Second
)

type Scheduler struct {
	tasks     sync.Map
	retryConf RetryConf
	taskQueue chan *ScheduledTask
	stopChan  chan struct{}
}

type ScheduledTask struct {
	check    base.CheckStrategy
	nextRun  time.Time
	interval time.Duration
}

type RetryConf struct {
	MaxRetries   int
	MaxRetryTime time.Duration
	Delay        time.Duration
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		retryConf: RetryConf{
			MaxRetries:   MAX_RETRIES,
			MaxRetryTime: MAX_RETRY_TIMES,
			Delay:        DEFAULT_DELAY,
		},
		taskQueue: make(chan *ScheduledTask),
		stopChan:  make(chan struct{}),
	}
}

func (s *Scheduler) AddTask(check base.CheckStrategy) {
	interval := check.GetInterval()
	task := &ScheduledTask{
		check:    check,
		nextRun:  time.Now().Add(interval),
		interval: interval,
	}
	s.tasks.Store(check.GetName(), task)
}

func (s *Scheduler) Start(ctx context.Context, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go s.worker(ctx)
	}

	go s.dispatcher(ctx)
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
	close(s.taskQueue)
}

func (s *Scheduler) dispatcher(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			now := time.Now()
			s.tasks.Range(func(key, value interface{}) bool {
				task, ok := value.(*ScheduledTask)
				if !ok {
					return true
				}

				if now.After(task.nextRun) {
					s.taskQueue <- task
				}
				return true
			})
		}
	}
}

func (s *Scheduler) worker(ctx context.Context) {
	for task := range s.taskQueue {
		select {
		case <-ctx.Done():
			return
		default:
			s.executeTask(ctx, task)
		}
	}
}

func (s *Scheduler) executeTask(ctx context.Context, task *ScheduledTask) {
	defer func() {
		task.nextRun = time.Now().Add(task.interval)
	}()

	for attempt := 0; attempt <= s.retryConf.MaxRetries; attempt++ {
		err := task.check.Execute(ctx)
		if err != nil {
			if attempt < s.retryConf.MaxRetries {
				backoff := utils.CalculateBackOff(s.retryConf.Delay, attempt)
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return
				}
			}
			return
		}
		break
	}

}
