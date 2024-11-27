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

var (
	confURL = "/api/metrics/config/"
)

type Collector struct {
	transporter transporter.TransportStrategy
	scheduler   *scheduler.Scheduler
	buffer      *base.CheckBuffer
	errorChan   chan error
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

type collectorArgs struct {
	session          *session.Session
	client           *ent.Client
	conf             []collectConf
	checkFactory     check.CheckFactory
	transportFactory transporter.TransporterFactory
}

type collectConf struct {
	Type     base.CheckType
	Interval int
}

func InitCollector(session *session.Session, client *ent.Client) *Collector {
	checkFactory := &check.DefaultCheckFactory{}
	transporterFactory := &transporter.DefaultTransporterFactory{}

	var conf []collectConf
	resp, statusCode, err := session.Get(confURL, 10)
	if statusCode == http.StatusOK {
		err = json.Unmarshal(resp, &conf)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal collection config")
			os.Exit(1)
		}
	} else {
		log.Error().Err(err).Msgf("HTTP %d: Failed to get collection config", statusCode)
		os.Exit(1)
	}

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

func NewCollector(args collectorArgs) (*Collector, error) {
	transporter, err := args.transportFactory.CreateTransporter(args.session)
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

	for _, entry := range args.conf {
		duration := time.Duration(entry.Interval) * time.Minute
		name := string(entry.Type) + "_" + uuid.NewString()
		checkArgs := base.CheckArgs{
			Type:     entry.Type,
			Name:     name,
			Interval: time.Duration(duration.Minutes() * float64(time.Minute)),
			Buffer:   checkBuffer,
			Client:   args.client,
		}

		check, err := args.checkFactory.CreateCheck(&checkArgs)
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
