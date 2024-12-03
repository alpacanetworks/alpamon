package net

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacanetworks/alpamon-go/pkg/collector/check/base"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/net"
)

type Check struct {
	base.BaseCheck
	retryCount base.RetryCount
	lastMetric map[string]net.IOCountersStat
}

func NewCheck(args *base.CheckArgs) base.CheckStrategy {
	return &Check{
		BaseCheck: base.NewBaseCheck(args),
		retryCount: base.RetryCount{
			MaxCollectRetries: base.COLLECT_MAX_RETRIES,
			MaxSaveRetries:    base.SAVE_MAX_RETRIES,
			MaxRetryTime:      base.MAX_RETRY_TIMES,
			Delay:             base.DEFAULT_DELAY,
		},
		lastMetric: make(map[string]net.IOCountersStat),
	}
}

func (c *Check) Execute(ctx context.Context) {
	metric, err := c.collectAndSaveTraffic(ctx)
	if err != nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	buffer := c.GetBuffer()
	buffer.SuccessQueue <- metric
}

func (c *Check) collectAndSaveTraffic(ctx context.Context) (base.MetricData, error) {
	ioCounters, interfaces, err := c.retryCollectTraffic(ctx)
	if err != nil {
		return base.MetricData{}, err
	}

	metric := base.MetricData{
		Type: base.NET,
		Data: c.parseTraffic(ioCounters, interfaces),
	}

	err = c.retrySaveTraffic(ctx, metric.Data)
	if err != nil {
		return base.MetricData{}, err
	}

	return metric, nil
}

func (c *Check) retryCollectTraffic(ctx context.Context) ([]net.IOCountersStat, map[string]net.InterfaceStat, error) {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxCollectRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		ioCounters, ioErr := c.collectIOCounters()
		interfaces, ifaceErr := c.collectInterfaces()

		if ioErr == nil && ifaceErr == nil {
			return ioCounters, interfaces, nil
		}

		if attempt < c.retryCount.MaxCollectRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to collect traffic: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		}
	}

	return nil, nil, fmt.Errorf("failed to collect traffic")
}

func (c *Check) parseTraffic(ioCOunters []net.IOCountersStat, interfaces map[string]net.InterfaceStat) []base.CheckResult {
	var data []base.CheckResult
	for _, ioCounter := range ioCOunters {
		if _, ok := interfaces[ioCounter.Name]; ok {
			var inputPkts, inputBytes, outputPkts, outputBytes uint64

			if lastCounter, exists := c.lastMetric[ioCounter.Name]; exists {
				inputPkts = ioCounter.PacketsRecv - lastCounter.PacketsRecv
				inputBytes = ioCounter.BytesRecv - lastCounter.BytesRecv
				outputPkts = ioCounter.PacketsSent - lastCounter.PacketsSent
				outputBytes = ioCounter.BytesSent - lastCounter.BytesSent
			} else {
				inputPkts = ioCounter.PacketsRecv
				inputBytes = ioCounter.BytesRecv
				outputPkts = ioCounter.PacketsSent
				outputBytes = ioCounter.BytesSent
			}

			c.lastMetric[ioCounter.Name] = ioCounter
			data = append(data, base.CheckResult{
				Timestamp:   time.Now(),
				Name:        ioCounter.Name,
				InputPkts:   inputPkts,
				InputBytes:  inputBytes,
				OutputPkts:  outputPkts,
				OutputBytes: outputBytes,
			})
		}
	}
	return data
}

func (c *Check) retrySaveTraffic(ctx context.Context, data []base.CheckResult) error {
	start := time.Now()
	for attempt := 0; attempt <= c.retryCount.MaxSaveRetries; attempt++ {
		if time.Since(start) >= c.retryCount.MaxRetryTime {
			break
		}

		err := c.saveTraffic(ctx, data)
		if err == nil {
			return nil
		}

		if attempt < c.retryCount.MaxSaveRetries {
			backoff := utils.CalculateBackOff(c.retryCount.Delay, attempt)
			select {
			case <-time.After(backoff):
				log.Debug().Msgf("Retry to save traffic: %d attempt", attempt)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to save traffic")
}

func (c *Check) collectInterfaces() (map[string]net.InterfaceStat, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	interfaces := map[string]net.InterfaceStat{}
	for _, iface := range ifaces {
		mac := iface.HardwareAddr
		if mac == "" {
			continue
		}
		interfaces[iface.Name] = iface
	}

	return interfaces, nil
}

func (c *Check) collectIOCounters() ([]net.IOCountersStat, error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	return ioCounters, nil
}

func (c *Check) saveTraffic(ctx context.Context, data []base.CheckResult) error {
	client := c.GetClient()
	err := client.Traffic.MapCreateBulk(data, func(q *ent.TrafficCreate, i int) {
		q.SetTimestamp(data[i].Timestamp).
			SetName(data[i].Name).
			SetInputPkts(int64(data[i].InputPkts)).
			SetInputBytes(int64(data[i].InputBytes)).
			SetOutputPkts(int64(data[i].OutputPkts)).
			SetOutputBytes(int64(data[i].OutputBytes))
	}).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
