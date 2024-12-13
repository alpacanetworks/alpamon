package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/scheduler"
	"github.com/alpacanetworks/alpamon-go/pkg/collector/transporter"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	session "github.com/alpacanetworks/alpamon-go/pkg/scheduler"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	confURL       = "/api/metrics/config/"
	maxRetryCount = 5
	delay         = 1 * time.Second
)

type Collector struct {
	transporter transporter.TransportStrategy
	scheduler   *scheduler.Scheduler
	buffer      *base.CheckBuffer
	errorChan   chan error
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

type collectConf struct {
	Type     base.CheckType
	Interval int
}

type collectorArgs struct {
	session          *session.Session
	client           *ent.Client
	conf             []collectConf
	checkFactory     check.CheckFactory
	transportFactory transporter.TransporterFactory
}

func InitCollector(session *session.Session, client *ent.Client) *Collector {
	conf, err := fetchConfig(session)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch collector config")
		os.Exit(1)
	}

	checkFactory := &check.DefaultCheckFactory{}
	urlResolver := transporter.NewURLResolver()
	transporterFactory := transporter.NewDefaultTransporterFactory(urlResolver)
	args := collectorArgs{
		session:          session,
		client:           client,
		conf:             conf,
		checkFactory:     checkFactory,
		transportFactory: transporterFactory,
	}

	collector, err := NewCollector(args)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create collector")
		os.Exit(1)
	}

	return collector
}

func fetchConfig(session *session.Session) ([]collectConf, error) {
	resp, statusCode, err := session.Get(confURL, 10)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get collection config: %d status code", statusCode)
	}

	var conf []collectConf
	err = json.Unmarshal(resp, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return conf, nil
}

func NewCollector(args collectorArgs) (*Collector, error) {
	transporter, err := args.transportFactory.CreateTransporter(args.session)
	if err != nil {
		return nil, err
	}

	checkBuffer := base.NewCheckBuffer(len(args.conf) * 2)
	collector := &Collector{
		transporter: transporter,
		scheduler:   scheduler.NewScheduler(),
		buffer:      checkBuffer,
		errorChan:   make(chan error, 10),
		stopChan:    make(chan struct{}),
	}

	err = collector.initTasks(args)
	if err != nil {
		return nil, err
	}

	return collector, nil
}

func (c *Collector) initTasks(args collectorArgs) error {
	for _, entry := range args.conf {
		duration := time.Duration(entry.Interval) * time.Second
		name := string(entry.Type) + "_" + uuid.NewString()
		checkArgs := base.CheckArgs{
			Type:     entry.Type,
			Name:     name,
			Interval: time.Duration(duration.Minutes() * float64(time.Minute)),
			Buffer:   c.buffer,
			Client:   args.client,
		}

		check, err := args.checkFactory.CreateCheck(&checkArgs)
		if err != nil {
			return err
		}
		c.scheduler.AddTask(check)
	}
	return nil
}

func (c *Collector) Start(ctx context.Context) error {
	go c.scheduler.Start(ctx, c.buffer.Capacity)

	for i := 0; i < c.buffer.Capacity; i++ {
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
			err := c.retryWithBackoff(ctx, metric)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to check metric: %s", metric.Type)
			}
		}
	}
}

func (c *Collector) retryWithBackoff(ctx context.Context, metric base.MetricData) error {
	retryCount := 0
	for retryCount < maxRetryCount {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(1<<retryCount) * delay):
			err := c.transporter.Send(metric)
			if err != nil {
				retryCount++
				continue
			}

			return nil
		}
	}

	return fmt.Errorf("max retries exceeded for metric %s", metric.Type)
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
