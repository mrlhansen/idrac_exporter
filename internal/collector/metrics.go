package collector

import (
	"fmt"
	"time"
	"github.com/prometheus/client_golang/prometheus"
)

func health2value(health string) float64 {
	switch health {
	case "OK":
		return 0
	case "Warning":
		return 1
	case "Critical":
		return 2
	}
	return 10
}

func (mc *metricsCollector) NewSystemPowerOn(state string) prometheus.Metric {
	var value float64
	if state == "On" {
		value = 1
	}
	return prometheus.MustNewConstMetric(
		mc.SystemPowerOn,
		prometheus.GaugeValue,
		value,
	)
}

func (mc *metricsCollector) NewSystemHealth(health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.SystemHealth,
		prometheus.GaugeValue,
		value,
		health,
	)
}

func (mc *metricsCollector) NewSystemIndicatorLED(state string) prometheus.Metric {
	var value float64
	if state != "Off" {
		value = 1
	}
	return prometheus.MustNewConstMetric(
		mc.SystemIndicatorLED,
		prometheus.GaugeValue,
		value,
		state,
	)
}

func (mc *metricsCollector) NewSystemMemorySize(memory float64) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemMemorySize,
		prometheus.GaugeValue,
		memory,
	)
}

func (mc *metricsCollector) NewSystemCpuCount(cpus int, model string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemCpuCount,
		prometheus.GaugeValue,
		float64(cpus),
		model,
	)
}

func (mc *metricsCollector) NewSystemBiosInfo(version string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemBiosInfo,
		prometheus.GaugeValue,
		1.0,
		version,
	)
}

func (mc *metricsCollector) NewSystemMachineInfo(manufacturer, model, serial, sku string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemMachineInfo,
		prometheus.GaugeValue,
		1.0,
		manufacturer,
		model,
		serial,
		sku,
	)
}
// TODO: Should sensor metrics have an ID as well?
func (mc *metricsCollector) NewSensorsTemperature(temperature float64, name, units string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SensorsTemperature,
		prometheus.GaugeValue,
		temperature,
		name,
		units,
	)
}

func (mc *metricsCollector) NewSensorsFanSpeed(speed float64, name, units string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SensorsFanSpeed,
		prometheus.GaugeValue,
		speed,
		name,
		units,
	)
}

func (mc *metricsCollector) NewPowerSupplyInputWatts(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyInputWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *metricsCollector) NewPowerSupplyInputVoltage(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyInputVoltage,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *metricsCollector) NewPowerSupplyOutputWatts(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyOutputWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *metricsCollector) NewPowerSupplyCapacityWatts(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyCapacityWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *metricsCollector) NewPowerSupplyEfficiencyPercent(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyEfficiencyPercent,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *metricsCollector) NewPowerControlConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *metricsCollector) NewPowerControlCapacityWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlCapacityWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *metricsCollector) NewPowerControlMinConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlMinConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *metricsCollector) NewPowerControlMaxConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlMaxConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *metricsCollector) NewPowerControlAvgConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlAvgConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *metricsCollector) NewPowerControlInterval(interval int, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlInterval,
		prometheus.GaugeValue,
		float64(interval),
		id,
		name,
	)
}

func (mc *metricsCollector) NewSelEntry(id string, message string, component string, severity string, created time.Time) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SelEntry,
		prometheus.CounterValue,
		float64(created.Unix()),
		id,
		message,
		component,
		severity,
	)
}

func (mc *metricsCollector) NewDriveInfo(id, name, manufacturer, model, serial, mediatype, protocol string, slot int) prometheus.Metric {
	var slotstr string

	if slot < 0 {
		slotstr = ""
	} else {
		slotstr = fmt.Sprint(slot)
	}

	return prometheus.MustNewConstMetric(
		mc.DriveInfo,
		prometheus.GaugeValue,
		1.0,
		id,
		manufacturer,
		mediatype,
		model,
		name,
		protocol,
		serial,
		slotstr,
	)
}

func (mc *metricsCollector) NewDriveHealth(id, health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.DriveHealth,
		prometheus.GaugeValue,
		value,
		id,
		health,
	)
}

func (mc *metricsCollector) NewDriveCapacity(id string, capacity int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.DriveCapacity,
		prometheus.GaugeValue,
		float64(capacity),
		id,
	)
}

func (mc *metricsCollector) NewMemoryModuleInfo(id, name, manufacturer, memtype, serial, ecc string, rank int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleInfo,
		prometheus.GaugeValue,
		1.0,
		id,
		ecc,
		manufacturer,
		memtype,
		name,
		serial,
		fmt.Sprint(rank),
	)
}

func (mc *metricsCollector) NewMemoryModuleHealth(id, health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleHealth,
		prometheus.GaugeValue,
		value,
		id,
		health,
	)
}

func (mc *metricsCollector) NewMemoryModuleCapacity(id string, capacity int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleCapacity,
		prometheus.GaugeValue,
		float64(capacity),
		id,
	)
}

func (mc *metricsCollector) NewMemoryModuleSpeed(id string, speed int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleSpeed,
		prometheus.GaugeValue,
		float64(speed),
		id,
	)
}
