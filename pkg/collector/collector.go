package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/transporter"
	session "github.com/alpacanetworks/alpamon-go/pkg/scheduler"
)

type Collector struct {
	transporter transporter.TransportStrategy
	scheduler   *scheduler.Scheduler
	buffer      *base.CheckBuffer
	errorChan   chan error
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

func NewCollector(session *session.Session, checkFactory check.CheckFactory, transporterFactory transporter.TransporterFactory) (*Collector, error) {
	transporter, err := transporterFactory.CreateTransporter(session)
	if err != nil {
		return nil, err
	}

	checkBuffer := base.NewCheckBuffer(10)

	collector := &Collector{
		transporter: transporter,
		scheduler:   scheduler.NewScheduler(),
		buffer:      checkBuffer,
		errorChan:   make(chan error, 10),
		stopChan:    make(chan struct{}),
	}

	checkTypes := map[base.CheckType]string{
		base.CPU:        "cpu",
		base.MEM:        "memory",
		base.DISK_USAGE: "disk_usage",
		base.DISK_IO:    "disk_io",
		base.NET:        "net",
	}
	for checkType, name := range checkTypes {
		check, err := checkFactory.CreateCheck(checkType, name, time.Duration(time.Duration.Seconds(5)), checkBuffer)
		if err != nil {
			return nil, err
		}
		if err := collector.scheduler.AddTask(check); err != nil {
			return nil, err
		}
	}

	return collector, nil
}

func (c *Collector) Start(ctx context.Context) error {
	go c.scheduler.Start(ctx)

	for i := 0; i < 5; i++ {
		c.wg.Add(1)
		go c.successQueueWorker(ctx)
	}

	c.wg.Add(1)
	go c.failureQueueWorker(ctx)

	return nil
}

func (c *Collector) successQueueWorker(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case metric := <-c.buffer.SuccessQueue:
			if err := c.transporter.Send(metric); err != nil {
				select {
				case c.buffer.FailureQueue <- metric:
				default:
					c.errorChan <- fmt.Errorf("failed to move metric to failure queue: %v", err)
				}
			}
		}
	}

}

func (c *Collector) failureQueueWorker(ctx context.Context) {
	defer c.wg.Done()

	retryTicker := time.NewTicker(5 * time.Second)
	defer retryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-retryTicker.C:
			metric := <-c.buffer.FailureQueue
			if err := c.transporter.Send(metric); err != nil {
				c.buffer.FailureQueue <- metric
			}
		}
	}
}

func (c *Collector) Stop() {
	close(c.stopChan)
	c.scheduler.Stop()
	c.wg.Wait()

	close(c.buffer.SuccessQueue)
	close(c.buffer.FailureQueue)
	close(c.errorChan)
}

func (c *Collector) Errors() <-chan error {
	return c.errorChan
}
