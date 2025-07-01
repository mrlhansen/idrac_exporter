package collector

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

var mu sync.Mutex
var collectors = map[string]*Collector{}

type Collector struct {
	// Internal variables
	client     *Client
	registry   *prometheus.Registry
	collected  *sync.Cond
	collecting bool
	errors     atomic.Uint64
	builder    *strings.Builder

	// Exporter
	ExporterBuildInfo         *prometheus.Desc
	ExporterScrapeErrorsTotal *prometheus.Desc

	// System
	SystemPowerOn         *prometheus.Desc
	SystemHealth          *prometheus.Desc
	SystemIndicatorLED    *prometheus.Desc // This attribute is deprecated in Redfish
	SystemIndicatorActive *prometheus.Desc
	SystemMemorySize      *prometheus.Desc
	SystemCpuCount        *prometheus.Desc
	SystemBiosInfo        *prometheus.Desc
	SystemMachineInfo     *prometheus.Desc

	// Sensors
	SensorsTemperature *prometheus.Desc
	SensorsFanHealth   *prometheus.Desc
	SensorsFanSpeed    *prometheus.Desc

	// Power supply
	PowerSupplyHealth            *prometheus.Desc
	PowerSupplyOutputWatts       *prometheus.Desc
	PowerSupplyInputWatts        *prometheus.Desc
	PowerSupplyCapacityWatts     *prometheus.Desc
	PowerSupplyInputVoltage      *prometheus.Desc
	PowerSupplyEfficiencyPercent *prometheus.Desc

	// Power control
	PowerControlConsumedWatts    *prometheus.Desc
	PowerControlCapacityWatts    *prometheus.Desc
	PowerControlMinConsumedWatts *prometheus.Desc
	PowerControlMaxConsumedWatts *prometheus.Desc
	PowerControlAvgConsumedWatts *prometheus.Desc
	PowerControlInterval         *prometheus.Desc

	// System event log
	EventLogEntry *prometheus.Desc

	// Storage
	StorageInfo                 *prometheus.Desc
	StorageHealth               *prometheus.Desc
	StorageDriveInfo            *prometheus.Desc
	StorageDriveHealth          *prometheus.Desc
	StorageDriveCapacity        *prometheus.Desc
	StorageDriveLifeLeft        *prometheus.Desc
	StorageDriveIndicatorActive *prometheus.Desc
	StorageControllerInfo       *prometheus.Desc
	StorageControllerHealth     *prometheus.Desc
	StorageControllerSpeed      *prometheus.Desc
	StorageVolumeInfo           *prometheus.Desc
	StorageVolumeHealth         *prometheus.Desc
	StorageVolumeMediaSpan      *prometheus.Desc
	StorageVolumeCapacity       *prometheus.Desc

	// Memory modules
	MemoryModuleInfo     *prometheus.Desc
	MemoryModuleHealth   *prometheus.Desc
	MemoryModuleCapacity *prometheus.Desc
	MemoryModuleSpeed    *prometheus.Desc

	// Network
	NetworkInterfaceHealth *prometheus.Desc
	NetworkPortHealth      *prometheus.Desc
	NetworkPortSpeed       *prometheus.Desc
	NetworkPortLinkUp      *prometheus.Desc

	// Processors
	CpuInfo         *prometheus.Desc
	CpuHealth       *prometheus.Desc
	CpuVoltage      *prometheus.Desc
	CpuMaxSpeed     *prometheus.Desc
	CpuCurrentSpeed *prometheus.Desc
	CpuTotalCores   *prometheus.Desc
	CpuTotalThreads *prometheus.Desc

	// Dell OEM
	DellBatteryRollupHealth       *prometheus.Desc
	DellEstimatedSystemAirflowCFM *prometheus.Desc
	DellControllerBatteryHealth   *prometheus.Desc

	// Update Service
	FirmwareInfo *prometheus.Desc
}

func NewCollector() *Collector {
	prefix := config.Config.MetricsPrefix

	collector := &Collector{
		ExporterBuildInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "exporter", "build_info"),
			"Constant metric with build information for the exporter",
			nil, prometheus.Labels{
				"version":   version.Version,
				"revision":  version.Revision,
				"goversion": runtime.Version(),
			},
		),
		ExporterScrapeErrorsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "exporter", "scrape_errors_total"),
			"Total number of errors encountered while scraping target",
			nil, nil,
		),
		SystemPowerOn: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "power_on"),
			"Power state of the system",
			nil, nil,
		),
		SystemHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "health"),
			"Health status of the system",
			[]string{"status"}, nil,
		),
		SystemIndicatorLED: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "indicator_led_on"),
			"Indicator LED state of the system",
			[]string{"state"}, nil,
		),
		SystemIndicatorActive: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "indicator_active"),
			"State of the system location indicator",
			nil, nil,
		),
		SystemMemorySize: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "memory_size_bytes"),
			"Total memory size of the system in bytes",
			nil, nil,
		),
		SystemCpuCount: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "cpu_count"),
			"Total number of CPUs in the system",
			[]string{"model"}, nil,
		),
		SystemBiosInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "bios_info"),
			"Information about the BIOS",
			[]string{"version"}, nil,
		),
		SystemMachineInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "machine_info"),
			"Information about the machine",
			[]string{"manufacturer", "model", "serial", "sku", "hostname"}, nil,
		),
		SensorsTemperature: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "sensors", "temperature"),
			"Sensors reporting temperature measurements",
			[]string{"id", "name", "units"}, nil,
		),
		SensorsFanHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "sensors", "fan_health"),
			"Health status for fans",
			[]string{"id", "name", "status"}, nil,
		),
		SensorsFanSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "sensors", "fan_speed"),
			"Sensors reporting fan speed measurements",
			[]string{"id", "name", "units"}, nil,
		),
		PowerSupplyHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "health"),
			"Power supply health status",
			[]string{"id", "status"}, nil,
		),
		PowerSupplyOutputWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "output_watts"),
			"Power supply output in watts",
			[]string{"id"}, nil,
		),
		PowerSupplyInputWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "input_watts"),
			"Power supply input in watts",
			[]string{"id"}, nil,
		),
		PowerSupplyCapacityWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "capacity_watts"),
			"Power supply capacity in watts",
			[]string{"id"}, nil,
		),
		PowerSupplyInputVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "input_voltage"),
			"Power supply input voltage",
			[]string{"id"}, nil,
		),
		PowerSupplyEfficiencyPercent: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "efficiency_percent"),
			"Power supply efficiency in percentage",
			[]string{"id"}, nil,
		),
		PowerControlConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "consumed_watts"),
			"Consumption of power control system in watts",
			[]string{"id", "name"}, nil,
		),
		PowerControlCapacityWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "capacity_watts"),
			"Capacity of power control system in watts",
			[]string{"id", "name"}, nil,
		),
		PowerControlMinConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "min_consumed_watts"),
			"Minimum consumption of power control system during the reported interval",
			[]string{"id", "name"}, nil,
		),
		PowerControlMaxConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "max_consumed_watts"),
			"Maximum consumption of power control system during the reported interval",
			[]string{"id", "name"}, nil,
		),
		PowerControlAvgConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "avg_consumed_watts"),
			"Average consumption of power control system during the reported interval",
			[]string{"id", "name"}, nil,
		),
		PowerControlInterval: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "interval_in_minutes"),
			"Interval for measurements of power control system",
			[]string{"id", "name"}, nil,
		),
		EventLogEntry: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "events", "log_entry"),
			"Entry from the system event log",
			[]string{"id", "message", "severity"}, nil,
		),
		StorageInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage", "info"),
			"Information about storage sub systems",
			[]string{"id", "name"}, nil,
		),
		StorageHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage", "health"),
			"Health status for storage sub systems",
			[]string{"id", "status"}, nil,
		),
		StorageDriveInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "info"),
			"Information about disk drives",
			[]string{"id", "storage_id", "manufacturer", "mediatype", "model", "name", "protocol", "serial", "slot"}, nil,
		),
		StorageDriveHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "health"),
			"Health status for disk drives",
			[]string{"id", "storage_id", "status"}, nil,
		),
		StorageDriveCapacity: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "capacity_bytes"),
			"Capacity of disk drives in bytes",
			[]string{"id", "storage_id"}, nil,
		),
		StorageDriveLifeLeft: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "life_left_percent"),
			"Predicted life left in percent",
			[]string{"id", "storage_id"}, nil,
		),
		StorageDriveIndicatorActive: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "indicator_active"),
			"State of the drive location indicator",
			[]string{"id", "storage_id"}, nil,
		),
		StorageControllerInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_controller", "info"),
			"Information about storage controllers",
			[]string{"id", "storage_id", "manufacturer", "model", "name", "firmware"}, nil,
		),
		StorageControllerHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_controller", "health"),
			"Health status for storage controllers",
			[]string{"id", "storage_id", "status"}, nil,
		),
		StorageControllerSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_controller", "speed_mbps"),
			"Speed of storage controllers in Mbps",
			[]string{"id", "storage_id"}, nil,
		),
		StorageVolumeInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "info"),
			"Information about virtual volumes",
			[]string{"id", "storage_id", "name", "volumetype", "raidtype"}, nil,
		),
		StorageVolumeHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "health"),
			"Health status for virtual volumes",
			[]string{"id", "storage_id", "status"}, nil,
		),
		StorageVolumeMediaSpan: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "media_span_count"),
			"Number of media spanned by virtual volumes",
			[]string{"id", "storage_id"}, nil,
		),
		StorageVolumeCapacity: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "capacity_bytes"),
			"Capacity of virtual volumes in bytes",
			[]string{"id", "storage_id"}, nil,
		),
		MemoryModuleInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "info"),
			"Information about memory modules",
			[]string{"id", "ecc", "manufacturer", "type", "name", "serial", "rank"}, nil,
		),
		MemoryModuleHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "health"),
			"Health status for memory modules",
			[]string{"id", "status"}, nil,
		),
		MemoryModuleCapacity: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "capacity_bytes"),
			"Capacity of memory modules in bytes",
			[]string{"id"}, nil,
		),
		MemoryModuleSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "speed_mhz"),
			"Speed of memory modules in Mhz",
			[]string{"id"}, nil,
		),
		NetworkInterfaceHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_interface", "health"),
			"Health status for network interfaces",
			[]string{"id", "status"}, nil,
		),
		NetworkPortHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_port", "health"),
			"Health status for network ports",
			[]string{"id", "interface_id", "status"}, nil,
		),
		NetworkPortSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_port", "speed_mbps"),
			"Link speed of network ports in Mbps",
			[]string{"id", "interface_id"}, nil,
		),
		NetworkPortLinkUp: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_port", "link_up"),
			"Link status of network ports (up or down)",
			[]string{"id", "interface_id", "status"}, nil,
		),
		CpuInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "info"),
			"Information about the CPU",
			[]string{"id", "socket", "manufacturer", "model", "arch"}, nil,
		),
		CpuHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "health"),
			"Health status of the CPU",
			[]string{"id", "status"}, nil,
		),
		CpuVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "voltage"),
			"Current voltage of the CPU",
			[]string{"id"}, nil,
		),
		CpuMaxSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "max_speed_mhz"),
			"Maximum speed of the CPU in Mhz",
			[]string{"id"}, nil,
		),
		CpuCurrentSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "current_speed_mhz"),
			"Current speed of the CPU in Mhz",
			[]string{"id"}, nil,
		),
		CpuTotalCores: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "total_cores"),
			"Total number of CPU cores",
			[]string{"id"}, nil,
		),
		CpuTotalThreads: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "total_threads"),
			"Total number of CPU threads",
			[]string{"id"}, nil,
		),
		DellBatteryRollupHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "dell", "battery_rollup_health"),
			"Health rollup status for the batteries",
			[]string{"status"}, nil,
		),
		DellEstimatedSystemAirflowCFM: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "dell", "estimated_system_airflow_cfm"),
			"Estimated system airflow in cubic feet per minute",
			nil, nil,
		),
		DellControllerBatteryHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "dell", "controller_battery_health"),
			"Health status of storage controller battery",
			[]string{"id", "storage_id", "name", "status"}, nil,
		),
		FirmwareInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "firmware", "info"),
			"The version of the firmware",
			[]string{"name", "version", "state"}, nil,
		),
	}

	collector.builder = new(strings.Builder)
	collector.collected = sync.NewCond(new(sync.Mutex))
	collector.registry = prometheus.NewRegistry()
	collector.registry.Register(collector)

	return collector
}

func (collector *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.ExporterBuildInfo
	ch <- collector.ExporterScrapeErrorsTotal
	ch <- collector.SystemPowerOn
	ch <- collector.SystemHealth
	ch <- collector.SystemIndicatorLED
	ch <- collector.SystemIndicatorActive
	ch <- collector.SystemMemorySize
	ch <- collector.SystemCpuCount
	ch <- collector.SystemBiosInfo
	ch <- collector.SystemMachineInfo
	ch <- collector.SensorsTemperature
	ch <- collector.SensorsFanHealth
	ch <- collector.SensorsFanSpeed
	ch <- collector.PowerSupplyHealth
	ch <- collector.PowerSupplyOutputWatts
	ch <- collector.PowerSupplyInputWatts
	ch <- collector.PowerSupplyCapacityWatts
	ch <- collector.PowerSupplyInputVoltage
	ch <- collector.PowerSupplyEfficiencyPercent
	ch <- collector.PowerControlConsumedWatts
	ch <- collector.PowerControlCapacityWatts
	ch <- collector.PowerControlMinConsumedWatts
	ch <- collector.PowerControlMaxConsumedWatts
	ch <- collector.PowerControlAvgConsumedWatts
	ch <- collector.PowerControlInterval
	ch <- collector.EventLogEntry
	ch <- collector.StorageInfo
	ch <- collector.StorageHealth
	ch <- collector.StorageDriveInfo
	ch <- collector.StorageDriveHealth
	ch <- collector.StorageDriveCapacity
	ch <- collector.StorageDriveLifeLeft
	ch <- collector.StorageDriveIndicatorActive
	ch <- collector.StorageControllerInfo
	ch <- collector.StorageControllerHealth
	ch <- collector.StorageControllerSpeed
	ch <- collector.StorageVolumeInfo
	ch <- collector.StorageVolumeHealth
	ch <- collector.StorageVolumeMediaSpan
	ch <- collector.StorageVolumeCapacity
	ch <- collector.MemoryModuleInfo
	ch <- collector.MemoryModuleHealth
	ch <- collector.MemoryModuleCapacity
	ch <- collector.MemoryModuleSpeed
	ch <- collector.NetworkInterfaceHealth
	ch <- collector.NetworkPortHealth
	ch <- collector.NetworkPortSpeed
	ch <- collector.NetworkPortLinkUp
	ch <- collector.CpuInfo
	ch <- collector.CpuHealth
	ch <- collector.CpuVoltage
	ch <- collector.CpuMaxSpeed
	ch <- collector.CpuCurrentSpeed
	ch <- collector.CpuTotalCores
	ch <- collector.CpuTotalThreads
	ch <- collector.DellBatteryRollupHealth
	ch <- collector.DellEstimatedSystemAirflowCFM
	ch <- collector.DellControllerBatteryHealth
}

func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup

	collector.client.redfish.RefreshSession()
	collect := &config.Config.Collect

	if collect.System {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshSystem(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Sensors {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshSensors(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Power {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshPower(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Network {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshNetwork(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Events {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshEventLog(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Storage {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshStorage(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Memory {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshMemory(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Processors {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshProcessors(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Extra {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshDell(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	if collect.Firmware {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshFirmware(collector, ch)
			if !ok {
				collector.errors.Add(1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	ch <- prometheus.MustNewConstMetric(collector.ExporterBuildInfo, prometheus.UntypedValue, 1)
	ch <- prometheus.MustNewConstMetric(collector.ExporterScrapeErrorsTotal, prometheus.CounterValue, float64(collector.errors.Load()))
}

func (collector *Collector) Gather() (string, error) {
	collector.collected.L.Lock()

	// If a collection is already in progress wait for it to complete and return the cached data
	if collector.collecting {
		collector.collected.Wait()
		metrics := collector.builder.String()
		collector.collected.L.Unlock()
		return metrics, nil
	}

	// Set collecting to true and let other goroutines enter in critical section
	collector.collecting = true
	collector.collected.L.Unlock()

	// Defer set collecting to false and wake waiting goroutines
	defer func() {
		collector.collected.L.Lock()
		collector.collected.Broadcast()
		collector.collecting = false
		collector.collected.L.Unlock()
	}()

	// Collect metrics
	collector.builder.Reset()

	m, err := collector.registry.Gather()
	if err != nil {
		return "", err
	}

	for i := range m {
		expfmt.MetricFamilyToText(collector.builder, m[i])
	}

	return collector.builder.String(), nil
}

// Resets an existing collector of the given target
func Reset(target string) {
	mu.Lock()
	_, ok := collectors[target]
	if ok {
		delete(collectors, target)
	}
	mu.Unlock()
}

func GetCollector(target string) (*Collector, error) {
	mu.Lock()
	collector, ok := collectors[target]
	if !ok {
		collector = NewCollector()
		collectors[target] = collector
	}
	mu.Unlock()

	// Do not act concurrently on the same host
	collector.collected.L.Lock()
	defer collector.collected.L.Unlock()

	// Try to instantiate a new Redfish host
	if collector.client == nil {
		host := config.Config.GetHostCfg(target)
		if host == nil {
			return nil, fmt.Errorf("failed to get host information")
		}
		c := NewClient(host)
		if c == nil {
			return nil, fmt.Errorf("failed to instantiate new client")
		} else {
			collector.client = c
		}
	}

	return collector, nil
}
