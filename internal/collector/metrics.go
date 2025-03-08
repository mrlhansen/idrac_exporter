package collector

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func health2value(health string) int {
	switch health {
	case "":
		return -1
	case "OK", "GoodInUse":
		return 0
	case "Warning":
		return 1
	case "Critical":
		return 2
	}
	return 10
}

func linkstatus2value(status string) int {
	switch status {
	case "Up", "LinkUp":
		return 1
	}
	return 0
}

func (mc *Collector) NewSystemPowerOn(ch chan<- prometheus.Metric, m *SystemResponse) {
	var value float64
	if m.PowerState == "On" {
		value = 1
	}
	ch <- prometheus.MustNewConstMetric(
		mc.SystemPowerOn,
		prometheus.GaugeValue,
		value,
	)
}

func (mc *Collector) NewSystemHealth(ch chan<- prometheus.Metric, m *SystemResponse) {
	value := health2value(m.Status.Health)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.SystemHealth,
		prometheus.GaugeValue,
		float64(value),
		m.Status.Health,
	)
}

func (mc *Collector) NewSystemIndicatorLED(ch chan<- prometheus.Metric, m *SystemResponse) {
	var value float64
	if m.IndicatorLED != "Off" {
		value = 1
	}
	ch <- prometheus.MustNewConstMetric(
		mc.SystemIndicatorLED,
		prometheus.GaugeValue,
		value,
		m.IndicatorLED,
	)
}

func (mc *Collector) NewSystemIndicatorActive(ch chan<- prometheus.Metric, m *SystemResponse) {
	var value float64
	if m.LocationIndicatorActive == nil {
		return
	}
	if *m.LocationIndicatorActive {
		value = 1
	}
	ch <- prometheus.MustNewConstMetric(
		mc.SystemIndicatorActive,
		prometheus.GaugeValue,
		value,
	)
}

func (mc *Collector) NewSystemMemorySize(ch chan<- prometheus.Metric, m *SystemResponse) {
	if m.MemorySummary == nil {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.SystemMemorySize,
		prometheus.GaugeValue,
		m.MemorySummary.TotalSystemMemoryGiB*1073741824,
	)
}

func (mc *Collector) NewSystemCpuCount(ch chan<- prometheus.Metric, m *SystemResponse) {
	if m.ProcessorSummary == nil {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.SystemCpuCount,
		prometheus.GaugeValue,
		float64(m.ProcessorSummary.Count),
		strings.TrimSpace(m.ProcessorSummary.Model),
	)
}

func (mc *Collector) NewSystemBiosInfo(ch chan<- prometheus.Metric, m *SystemResponse) {
	ch <- prometheus.MustNewConstMetric(
		mc.SystemBiosInfo,
		prometheus.UntypedValue,
		1.0,
		m.BiosVersion,
	)
}

func (mc *Collector) NewSystemMachineInfo(ch chan<- prometheus.Metric, m *SystemResponse) {
	ch <- prometheus.MustNewConstMetric(
		mc.SystemMachineInfo,
		prometheus.UntypedValue,
		1.0,
		strings.TrimSpace(m.Manufacturer),
		strings.TrimSpace(m.Model),
		strings.TrimSpace(m.SerialNumber),
		strings.TrimSpace(m.SKU),
		strings.TrimSpace(m.HostName),
	)
}

func (mc *Collector) NewSensorsTemperature(ch chan<- prometheus.Metric, temperature float64, id, name, units string) {
	ch <- prometheus.MustNewConstMetric(
		mc.SensorsTemperature,
		prometheus.GaugeValue,
		temperature,
		id,
		name,
		units,
	)
}

func (mc *Collector) NewSensorsFanHealth(ch chan<- prometheus.Metric, id, name, health string) {
	value := health2value(health)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.SensorsFanHealth,
		prometheus.GaugeValue,
		float64(value),
		id,
		name,
		health,
	)
}

func (mc *Collector) NewSensorsFanSpeed(ch chan<- prometheus.Metric, speed float64, id, name, units string) {
	ch <- prometheus.MustNewConstMetric(
		mc.SensorsFanSpeed,
		prometheus.GaugeValue,
		speed,
		id,
		name,
		units,
	)
}

func (mc *Collector) NewPowerSupplyHealth(ch chan<- prometheus.Metric, health, id string) {
	value := health2value(health)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.PowerSupplyHealth,
		prometheus.GaugeValue,
		float64(value),
		id,
		health,
	)
}

func (mc *Collector) NewPowerSupplyInputWatts(ch chan<- prometheus.Metric, value float64, id string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerSupplyInputWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyInputVoltage(ch chan<- prometheus.Metric, value float64, id string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerSupplyInputVoltage,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyOutputWatts(ch chan<- prometheus.Metric, value float64, id string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerSupplyOutputWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyCapacityWatts(ch chan<- prometheus.Metric, value float64, id string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerSupplyCapacityWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyEfficiencyPercent(ch chan<- prometheus.Metric, value float64, id string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerSupplyEfficiencyPercent,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerControlConsumedWatts(ch chan<- prometheus.Metric, value float64, id, name string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerControlConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlCapacityWatts(ch chan<- prometheus.Metric, value float64, id, name string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerControlCapacityWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlMinConsumedWatts(ch chan<- prometheus.Metric, value float64, id, name string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerControlMinConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlMaxConsumedWatts(ch chan<- prometheus.Metric, value float64, id, name string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerControlMaxConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlAvgConsumedWatts(ch chan<- prometheus.Metric, value float64, id, name string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerControlAvgConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlInterval(ch chan<- prometheus.Metric, interval int, id, name string) {
	ch <- prometheus.MustNewConstMetric(
		mc.PowerControlInterval,
		prometheus.GaugeValue,
		float64(interval),
		id,
		name,
	)
}

func (mc *Collector) NewEventLogEntry(ch chan<- prometheus.Metric, id string, message string, severity string, created time.Time) {
	ch <- prometheus.MustNewConstMetric(
		mc.EventLogEntry,
		prometheus.CounterValue,
		float64(created.Unix()),
		id,
		strings.TrimSpace(message),
		severity,
	)
}

func (mc *Collector) NewDriveInfo(ch chan<- prometheus.Metric, parent string, m *Drive) {
	var slot string

	if m.PhysicalLocation != nil {
		if m.PhysicalLocation.PartLocation != nil {
			slot = fmt.Sprint(m.PhysicalLocation.PartLocation.LocationOrdinalValue)
		}
	}

	ch <- prometheus.MustNewConstMetric(
		mc.DriveInfo,
		prometheus.UntypedValue,
		1.0,
		m.Id,
		parent,
		m.Manufacturer,
		m.MediaType,
		m.Model,
		m.Name,
		m.Protocol,
		m.SerialNumber,
		slot,
	)
}

func (mc *Collector) NewDriveHealth(ch chan<- prometheus.Metric, parent string, m *Drive) {
	value := health2value(m.Status.Health)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.DriveHealth,
		prometheus.GaugeValue,
		float64(value),
		m.Id,
		parent,
		m.Status.Health,
	)
}

func (mc *Collector) NewDriveCapacity(ch chan<- prometheus.Metric, parent string, m *Drive) {
	ch <- prometheus.MustNewConstMetric(
		mc.DriveCapacity,
		prometheus.GaugeValue,
		float64(m.CapacityBytes),
		m.Id,
		parent,
	)
}

func (mc *Collector) NewDriveLifeLeft(ch chan<- prometheus.Metric, parent string, m *Drive) {
	ch <- prometheus.MustNewConstMetric(
		mc.DriveLifeLeft,
		prometheus.GaugeValue,
		m.PredictedLifeLeft,
		m.Id,
		parent,
	)
}

func (mc *Collector) NewMemoryModuleInfo(ch chan<- prometheus.Metric, m *Memory) {
	ch <- prometheus.MustNewConstMetric(
		mc.MemoryModuleInfo,
		prometheus.UntypedValue,
		1.0,
		m.Id,
		m.ErrorCorrection,
		m.Manufacturer,
		m.MemoryDeviceType,
		m.Name,
		m.SerialNumber,
		fmt.Sprint(m.RankCount),
	)
}

func (mc *Collector) NewMemoryModuleHealth(ch chan<- prometheus.Metric, m *Memory) {
	value := health2value(m.Status.Health)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.MemoryModuleHealth,
		prometheus.GaugeValue,
		float64(value),
		m.Id,
		m.Status.Health,
	)
}

func (mc *Collector) NewMemoryModuleCapacity(ch chan<- prometheus.Metric, m *Memory) {
	capacity := 1048576 * m.CapacityMiB
	if capacity == 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.MemoryModuleCapacity,
		prometheus.GaugeValue,
		float64(capacity),
		m.Id,
	)
}

func (mc *Collector) NewMemoryModuleSpeed(ch chan<- prometheus.Metric, m *Memory) {
	if m.OperatingSpeedMhz == 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.MemoryModuleSpeed,
		prometheus.GaugeValue,
		float64(m.OperatingSpeedMhz),
		m.Id,
	)
}

func (mc *Collector) NewNetworkInterfaceHealth(ch chan<- prometheus.Metric, m *NetworkInterface) {
	value := health2value(m.Status.Health)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.NetworkInterfaceHealth,
		prometheus.GaugeValue,
		float64(value),
		m.Id,
		m.Status.Health,
	)
}

func (mc *Collector) NewNetworkPortHealth(ch chan<- prometheus.Metric, parent string, m *NetworkPort) {
	value := health2value(m.Status.Health)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.NetworkPortHealth,
		prometheus.GaugeValue,
		float64(value),
		m.Id,
		parent,
		m.Status.Health,
	)
}

func (mc *Collector) NewNetworkPortSpeed(ch chan<- prometheus.Metric, parent string, m *NetworkPort) {
	var speed float64

	if m.CurrentLinkSpeedMbps > 0 {
		speed = m.CurrentLinkSpeedMbps
	} else if m.CurrentSpeedGbps > 0 {
		speed = 1000 * m.CurrentSpeedGbps
	} else if len(m.SupportedLinkCapabilities) > 0 {
		if s := m.SupportedLinkCapabilities[0].LinkSpeedMbps; s > 0 {
			speed = s
		}
	}

	ch <- prometheus.MustNewConstMetric(
		mc.NetworkPortSpeed,
		prometheus.GaugeValue,
		speed,
		m.Id,
		parent,
	)
}

func (mc *Collector) NewNetworkPortLinkUp(ch chan<- prometheus.Metric, parent string, m *NetworkPort) {
	value := linkstatus2value(m.LinkStatus)
	ch <- prometheus.MustNewConstMetric(
		mc.NetworkPortLinkUp,
		prometheus.GaugeValue,
		float64(value),
		m.Id,
		parent,
		m.LinkStatus,
	)
}

func (mc *Collector) NewDellBatteryRollupHealth(ch chan<- prometheus.Metric, m *DellSystem) {
	value := health2value(m.BatteryRollupStatus)
	if value < 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.DellBatteryRollupHealth,
		prometheus.GaugeValue,
		float64(value),
		m.BatteryRollupStatus,
	)
}

func (mc *Collector) NewDellEstimatedSystemAirflowCFM(ch chan<- prometheus.Metric, m *DellSystem) {
	value := m.EstimatedSystemAirflowCFM
	if value == 0 {
		return
	}
	ch <- prometheus.MustNewConstMetric(
		mc.DellEstimatedSystemAirflowCFM,
		prometheus.GaugeValue,
		float64(value),
	)
}
