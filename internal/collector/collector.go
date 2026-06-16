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
	NetworkAdapterInfo      *prometheus.Desc
	NetworkAdapterHealth    *prometheus.Desc
	NetworkPortHealth       *prometheus.Desc
	NetworkPortMaxSpeed     *prometheus.Desc
	NetworkPortCurrentSpeed *prometheus.Desc
	NetworkPortLinkUp       *prometheus.Desc

	// Processors
	CpuInfo         *prometheus.Desc
	CpuHealth       *prometheus.Desc
	CpuVoltage      *prometheus.Desc
	CpuMaxSpeed     *prometheus.Desc
	CpuCurrentSpeed *prometheus.Desc
	CpuTotalCores   *prometheus.Desc
	CpuTotalThreads *prometheus.Desc

	// BMC
	ManagerInfo   *prometheus.Desc
	ManagerHealth *prometheus.Desc

	// Dell OEM
	DellBatteryRollupHealth       *prometheus.Desc
	DellEstimatedSystemAirflowCFM *prometheus.Desc
	DellControllerBatteryHealth   *prometheus.Desc

	// PDU
	PduInfo            *prometheus.Desc
	PduHealth          *prometheus.Desc
	PduPowerWatts      *prometheus.Desc
	PduPowerApparentVA *prometheus.Desc
	PduPowerFactor     *prometheus.Desc
	PduEnergyKWh       *prometheus.Desc
}

func NewCollector(labels prometheus.Labels) *Collector {
	prefix := config.Config.MetricsPrefix

	// Merge host labels with exporter build info labels
	buildLabels := prometheus.Labels{
		"version":   version.Version,
		"revision":  version.Revision,
		"goversion": runtime.Version(),
	}
	for k, v := range labels {
		buildLabels[k] = v
	}

	collector := &Collector{
		ExporterBuildInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "exporter", "build_info"),
			"Constant metric with build information for the exporter",
			nil, buildLabels,
		),
		ExporterScrapeErrorsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "exporter", "scrape_errors_total"),
			"Total number of errors encountered while scraping target",
			nil, labels,
		),
		SystemPowerOn: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "power_on"),
			"Power state of the system",
			nil, labels,
		),
		SystemHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "health"),
			"Health status of the system",
			[]string{"status"}, labels,
		),
		SystemIndicatorLED: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "indicator_led_on"),
			"Indicator LED state of the system",
			[]string{"state"}, labels,
		),
		SystemIndicatorActive: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "indicator_active"),
			"State of the system location indicator",
			nil, labels,
		),
		SystemMemorySize: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "memory_size_bytes"),
			"Total memory size of the system in bytes",
			nil, labels,
		),
		SystemCpuCount: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "cpu_count"),
			"Total number of CPUs in the system",
			[]string{"model"}, labels,
		),
		SystemBiosInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "bios_info"),
			"Information about the BIOS",
			[]string{"version"}, labels,
		),
		SystemMachineInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "system", "machine_info"),
			"Information about the machine",
			[]string{"manufacturer", "model", "serial", "sku", "hostname"}, labels,
		),
		SensorsTemperature: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "sensors", "temperature"),
			"Sensors reporting temperature measurements",
			[]string{"id", "name", "units"}, labels,
		),
		SensorsFanHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "sensors", "fan_health"),
			"Health status for fans",
			[]string{"id", "name", "status"}, labels,
		),
		SensorsFanSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "sensors", "fan_speed"),
			"Sensors reporting fan speed measurements",
			[]string{"id", "name", "units"}, labels,
		),
		PowerSupplyHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "health"),
			"Power supply health status",
			[]string{"id", "status"}, labels,
		),
		PowerSupplyOutputWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "output_watts"),
			"Power supply output in watts",
			[]string{"id"}, labels,
		),
		PowerSupplyInputWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "input_watts"),
			"Power supply input in watts",
			[]string{"id"}, labels,
		),
		PowerSupplyCapacityWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "capacity_watts"),
			"Power supply capacity in watts",
			[]string{"id"}, labels,
		),
		PowerSupplyInputVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "input_voltage"),
			"Power supply input voltage",
			[]string{"id"}, labels,
		),
		PowerSupplyEfficiencyPercent: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_supply", "efficiency_percent"),
			"Power supply efficiency in percentage",
			[]string{"id"}, labels,
		),
		PowerControlConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "consumed_watts"),
			"Consumption of power control system in watts",
			[]string{"id", "name"}, labels,
		),
		PowerControlCapacityWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "capacity_watts"),
			"Capacity of power control system in watts",
			[]string{"id", "name"}, labels,
		),
		PowerControlMinConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "min_consumed_watts"),
			"Minimum consumption of power control system during the reported interval",
			[]string{"id", "name"}, labels,
		),
		PowerControlMaxConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "max_consumed_watts"),
			"Maximum consumption of power control system during the reported interval",
			[]string{"id", "name"}, labels,
		),
		PowerControlAvgConsumedWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "avg_consumed_watts"),
			"Average consumption of power control system during the reported interval",
			[]string{"id", "name"}, labels,
		),
		PowerControlInterval: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "power_control", "interval_in_minutes"),
			"Interval for measurements of power control system",
			[]string{"id", "name"}, labels,
		),
		EventLogEntry: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "events", "log_entry"),
			"Entry from the system event log",
			[]string{"id", "message", "severity"}, labels,
		),
		StorageInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage", "info"),
			"Information about storage sub systems",
			[]string{"id", "name"}, labels,
		),
		StorageHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage", "health"),
			"Health status for storage sub systems",
			[]string{"id", "status"}, labels,
		),
		StorageDriveInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "info"),
			"Information about disk drives",
			[]string{"id", "storage_id", "manufacturer", "mediatype", "model", "name", "protocol", "serial", "slot"}, labels,
		),
		StorageDriveHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "health"),
			"Health status for disk drives",
			[]string{"id", "storage_id", "status"}, labels,
		),
		StorageDriveCapacity: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "capacity_bytes"),
			"Capacity of disk drives in bytes",
			[]string{"id", "storage_id"}, labels,
		),
		StorageDriveLifeLeft: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "life_left_percent"),
			"Predicted life left in percent",
			[]string{"id", "storage_id"}, labels,
		),
		StorageDriveIndicatorActive: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_drive", "indicator_active"),
			"State of the drive location indicator",
			[]string{"id", "storage_id"}, labels,
		),
		StorageControllerInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_controller", "info"),
			"Information about storage controllers",
			[]string{"id", "storage_id", "manufacturer", "model", "name", "firmware"}, labels,
		),
		StorageControllerHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_controller", "health"),
			"Health status for storage controllers",
			[]string{"id", "storage_id", "status"}, labels,
		),
		StorageControllerSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_controller", "speed_mbps"),
			"Speed of storage controllers in Mbps",
			[]string{"id", "storage_id"}, labels,
		),
		StorageVolumeInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "info"),
			"Information about virtual volumes",
			[]string{"id", "storage_id", "name", "volumetype", "raidtype"}, labels,
		),
		StorageVolumeHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "health"),
			"Health status for virtual volumes",
			[]string{"id", "storage_id", "status"}, labels,
		),
		StorageVolumeMediaSpan: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "media_span_count"),
			"Number of media spanned by virtual volumes",
			[]string{"id", "storage_id"}, labels,
		),
		StorageVolumeCapacity: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "storage_volume", "capacity_bytes"),
			"Capacity of virtual volumes in bytes",
			[]string{"id", "storage_id"}, labels,
		),
		MemoryModuleInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "info"),
			"Information about memory modules",
			[]string{"id", "ecc", "manufacturer", "type", "name", "serial", "rank"}, labels,
		),
		MemoryModuleHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "health"),
			"Health status for memory modules",
			[]string{"id", "status"}, labels,
		),
		MemoryModuleCapacity: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "capacity_bytes"),
			"Capacity of memory modules in bytes",
			[]string{"id"}, labels,
		),
		MemoryModuleSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "memory_module", "speed_mhz"),
			"Speed of memory modules in Mhz",
			[]string{"id"}, labels,
		),
		NetworkAdapterInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_adapter", "info"),
			"Information about network adapters",
			[]string{"id", "manufacturer", "model", "serial"}, labels,
		),
		NetworkAdapterHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_adapter", "health"),
			"Health status for network adapters",
			[]string{"id", "status"}, labels,
		),
		NetworkPortHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_port", "health"),
			"Health status for network ports",
			[]string{"id", "adapter_id", "status"}, labels,
		),
		NetworkPortMaxSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_port", "max_speed_mbps"),
			"Max link speed of network ports in Mbps",
			[]string{"id", "adapter_id"}, labels,
		),
		NetworkPortCurrentSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_port", "current_speed_mbps"),
			"Current link speed of network ports in Mbps",
			[]string{"id", "adapter_id"}, labels,
		),
		NetworkPortLinkUp: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "network_port", "link_up"),
			"Link status of network ports (up or down)",
			[]string{"id", "adapter_id", "status"}, labels,
		),
		CpuInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "info"),
			"Information about the CPU",
			[]string{"id", "socket", "manufacturer", "model", "arch"}, labels,
		),
		CpuHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "health"),
			"Health status of the CPU",
			[]string{"id", "status"}, labels,
		),
		CpuVoltage: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "voltage"),
			"Current voltage of the CPU",
			[]string{"id"}, labels,
		),
		CpuMaxSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "max_speed_mhz"),
			"Maximum speed of the CPU in Mhz",
			[]string{"id"}, labels,
		),
		CpuCurrentSpeed: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "current_speed_mhz"),
			"Current speed of the CPU in Mhz",
			[]string{"id"}, labels,
		),
		CpuTotalCores: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "total_cores"),
			"Total number of CPU cores",
			[]string{"id"}, labels,
		),
		CpuTotalThreads: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "cpu", "total_threads"),
			"Total number of CPU threads",
			[]string{"id"}, labels,
		),
		ManagerInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "manager", "info"),
			"Information about the manager",
			[]string{"id", "type", "model", "firmware"}, labels,
		),
		ManagerHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "manager", "health"),
			"Health status of the manager",
			[]string{"id", "status"}, labels,
		),
		DellBatteryRollupHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "dell", "battery_rollup_health"),
			"Health rollup status for the batteries",
			[]string{"status"}, labels,
		),
		DellEstimatedSystemAirflowCFM: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "dell", "estimated_system_airflow_cfm"),
			"Estimated system airflow in cubic feet per minute",
			nil, labels,
		),
		DellControllerBatteryHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "dell", "controller_battery_health"),
			"Health status of storage controller battery",
			[]string{"id", "storage_id", "name", "status"}, labels,
		),
		PduInfo: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "pdu", "info"),
			"Information about the PDU",
			[]string{"id", "firmware", "manufacturer", "model", "name", "serial", "type"}, labels,
		),
		PduHealth: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "pdu", "health"),
			"Health status of the PDU",
			[]string{"id", "status"}, labels,
		),
		PduPowerWatts: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "pdu", "power_watts"),
			"Power reading in watts",
			[]string{"id"}, labels,
		),
		PduPowerApparentVA: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "pdu", "power_apparent_va"),
			"Apparent power reading in VA units",
			[]string{"id"}, labels,
		),
		PduPowerFactor: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "pdu", "power_factor"),
			"Power factor (efficiency)",
			[]string{"id"}, labels,
		),
		PduEnergyKWh: prometheus.NewDesc(
			prometheus.BuildFQName(prefix, "pdu", "energy_kwh"),
			"Energy consumption in kWh",
			[]string{"id"}, labels,
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
	ch <- collector.NetworkAdapterInfo
	ch <- collector.NetworkAdapterHealth
	ch <- collector.NetworkPortHealth
	ch <- collector.NetworkPortMaxSpeed
	ch <- collector.NetworkPortCurrentSpeed
	ch <- collector.NetworkPortLinkUp
	ch <- collector.CpuInfo
	ch <- collector.CpuHealth
	ch <- collector.CpuVoltage
	ch <- collector.CpuMaxSpeed
	ch <- collector.CpuCurrentSpeed
	ch <- collector.CpuTotalCores
	ch <- collector.CpuTotalThreads
	ch <- collector.ManagerInfo
	ch <- collector.ManagerHealth
	ch <- collector.DellBatteryRollupHealth
	ch <- collector.DellEstimatedSystemAirflowCFM
	ch <- collector.DellControllerBatteryHealth
	ch <- collector.PduInfo
	ch <- collector.PduHealth
	ch <- collector.PduPowerWatts
	ch <- collector.PduPowerApparentVA
	ch <- collector.PduPowerFactor
	ch <- collector.PduEnergyKWh
}

func (collector *Collector) CollectServer(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup
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

	if collect.Manager {
		wg.Add(1)
		go func() {
			ok := collector.client.RefreshManager(collector, ch)
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

	wg.Wait()
}

func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	collector.client.redfish.RefreshSession()

	if len(collector.client.path.RackPDUs) > 0 {
		ok := collector.client.RefreshPDUs(collector, ch)
		if !ok {
			collector.errors.Add(1)
		}
	} else {
		collector.CollectServer(ch)
	}

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

func GetCollector(target, auth string) (*Collector, error) {
	authConfig := config.GetAuthConfig(target, auth)
	if authConfig == nil {
		return nil, fmt.Errorf("could not find login credentials")
	}

	mu.Lock()
	collector, ok := collectors[target]
	if !ok {
		collector = NewCollector(prometheus.Labels(authConfig.Labels))
		collectors[target] = collector
	}
	mu.Unlock()

	// Do not act concurrently on the same host
	collector.collected.L.Lock()
	defer collector.collected.L.Unlock()

	// Try to instantiate a new Redfish host
	if collector.client == nil {
		c := NewClient(target, authConfig)
		if c == nil {
			return nil, fmt.Errorf("failed to instantiate new client")
		} else {
			collector.client = c
		}
	}

	return collector, nil
}
