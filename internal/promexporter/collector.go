package promexporter

import (
	"fmt"
	"sync"
	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/redfish"
)

var mapMu sync.Mutex
var collectors = map[string]*metricsCollector{}

type metricsCollector struct {
	*redfish.Client
	store      *MetricsStore
	collected  *sync.Cond
	collecting bool
	reachable  bool
	retries    uint
}

// CollectMetrics collect the metrics using the underlying redfish.Client, storing them in the MetricsStore.
// It returns the collected metrics in OpenMetrics format along any error encountered in the process.
//
// If two or more goroutines try to collect the metrics concurrently,
// the first one will actually make the Redfish calls and store the new data,
// the others will simply gather the metrics previously stored by the first goroutine
func (c *metricsCollector) CollectMetrics() (string, error) {
	c.collected.L.Lock()

	// If a collection is already in progress wait for it to complete and return the cached data
	if c.collecting {
		c.collected.Wait()
		metrics := c.store.Gather()
		c.collected.L.Unlock()
		return metrics, nil
	}

	// Set c.collecting to true and let other goroutines enter in critical section
	c.collecting = true
	c.collected.L.Unlock()

	// Defer set c.collecting to false and wake waiting goroutines
	defer func() {
		c.collected.L.Lock()
		c.collected.Broadcast()
		c.collecting = false
		c.collected.L.Unlock()
	}()

	c.store.Reset()
	//
	// if config.Config.Collect.System {
	// 	err := c.RefreshSystem(c.store);
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }
	// // if config.Config.Collect.Sensors {
	// // 	err := c.RefreshSensors(c.store);
	// // 	if err != nil {
	// // 		return "", err
	// // 	}
	// // }
	// if config.Config.Collect.Power {
	// 	err := c.RefreshPower(c.store);
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }
	// if config.Config.Collect.SEL {
	// 	err := c.RefreshIdracSel(c.store);
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }
	// if config.Config.Collect.Storage {
	// 	err := c.RefreshStorage(c.store);
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }
	// if config.Config.Collect.Memory {
	// 	err := c.RefreshMemory(c.store);
	// 	if err != nil {
	// 		return "", err
	// 	}
	// }

	return c.store.Gather(), nil
}

func getCollector(target string) (*metricsCollector, error) {
	mapMu.Lock()
	collector, ok := collectors[target]
	if !ok {
		collector = &metricsCollector{
			collected: sync.NewCond(new(sync.Mutex)),
			store:     NewMetricsStore(config.Config.MetricsPrefix),
		}
		collectors[target] = collector
	}
	mapMu.Unlock()

	// Do not act concurrently on the same host
	collector.collected.L.Lock()
	defer collector.collected.L.Unlock()

	// If Redfish host is nil, and we did not exceed max retries, try to instantiate a new Redfish host
	if collector.Client == nil && collector.retries < config.Config.Retries {
		hostCfg := config.Config.GetHostCfg(target)
		c, err := redfish.NewClient(hostCfg)
		if err != nil {
			collector.retries++
			return collector, err
		} else {
			collector.Client = c
			collector.reachable = true
		}
	}

	if collector.Client == nil && collector.retries >= config.Config.Retries {
		return nil, fmt.Errorf("collector unreachable after %d retries", collector.retries)
	}

	return collector, nil
}
