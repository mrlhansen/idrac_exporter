package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"time"
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

func linkstatus2value(status string) float64 {
	switch status {
	case "LinkUp":
	case "Up":
		return 1
	}
	return 0
}

func (mc *Collector) NewSystemPowerOn(state string) prometheus.Metric {
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

func (mc *Collector) NewSystemHealth(health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.SystemHealth,
		prometheus.GaugeValue,
		value,
		health,
	)
}

func (mc *Collector) NewSystemIndicatorLED(state string) prometheus.Metric {
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

func (mc *Collector) NewSystemMemorySize(memory float64) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemMemorySize,
		prometheus.GaugeValue,
		memory,
	)
}

func (mc *Collector) NewSystemCpuCount(cpus int, model string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemCpuCount,
		prometheus.GaugeValue,
		float64(cpus),
		strings.TrimSpace(model),
	)
}

func (mc *Collector) NewSystemBiosInfo(version string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemBiosInfo,
		prometheus.UntypedValue,
		1.0,
		version,
	)
}

func (mc *Collector) NewSystemMachineInfo(manufacturer, model, serial, sku string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SystemMachineInfo,
		prometheus.UntypedValue,
		1.0,
		manufacturer,
		model,
		serial,
		sku,
	)
}

func (mc *Collector) NewSensorsTemperature(temperature float64, id, name, units string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SensorsTemperature,
		prometheus.GaugeValue,
		temperature,
		id,
		name,
		units,
	)
}

func (mc *Collector) NewSensorsFanHealth(id, name, health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.SensorsFanHealth,
		prometheus.GaugeValue,
		value,
		id,
		name,
		health,
	)
}
func (mc *Collector) NewSensorsFanSpeed(speed float64, id, name, units string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.SensorsFanSpeed,
		prometheus.GaugeValue,
		speed,
		id,
		name,
		units,
	)
}

func (mc *Collector) NewPowerSupplyHealth(health, id string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyHealth,
		prometheus.GaugeValue,
		value,
		id,
		health,
	)
}
func (mc *Collector) NewPowerSupplyInputWatts(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyInputWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyInputVoltage(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyInputVoltage,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyOutputWatts(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyOutputWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyCapacityWatts(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyCapacityWatts,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerSupplyEfficiencyPercent(value float64, id string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerSupplyEfficiencyPercent,
		prometheus.GaugeValue,
		value,
		id,
	)
}

func (mc *Collector) NewPowerControlConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlCapacityWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlCapacityWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlMinConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlMinConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlMaxConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlMaxConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlAvgConsumedWatts(value float64, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlAvgConsumedWatts,
		prometheus.GaugeValue,
		value,
		id,
		name,
	)
}

func (mc *Collector) NewPowerControlInterval(interval int, id, name string) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.PowerControlInterval,
		prometheus.GaugeValue,
		float64(interval),
		id,
		name,
	)
}

func (mc *Collector) NewSelEntry(id string, message string, component string, severity string, created time.Time) prometheus.Metric {
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

func (mc *Collector) NewDriveInfo(id, name, manufacturer, model, serial, mediatype, protocol string, slot int) prometheus.Metric {
	var slotstr string

	if slot < 0 {
		slotstr = ""
	} else {
		slotstr = fmt.Sprint(slot)
	}

	return prometheus.MustNewConstMetric(
		mc.DriveInfo,
		prometheus.UntypedValue,
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

func (mc *Collector) NewDriveHealth(id, health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.DriveHealth,
		prometheus.GaugeValue,
		value,
		id,
		health,
	)
}

func (mc *Collector) NewDriveCapacity(id string, capacity int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.DriveCapacity,
		prometheus.GaugeValue,
		float64(capacity),
		id,
	)
}

func (mc *Collector) NewDriveLifeLeft(id string, lifeLeft int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.DriveLifeLeft,
		prometheus.GaugeValue,
		float64(lifeLeft),
		id,
	)
}

func (mc *Collector) NewMemoryModuleInfo(id, name, manufacturer, memtype, serial, ecc string, rank int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleInfo,
		prometheus.UntypedValue,
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

func (mc *Collector) NewMemoryModuleHealth(id, health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleHealth,
		prometheus.GaugeValue,
		value,
		id,
		health,
	)
}

func (mc *Collector) NewMemoryModuleCapacity(id string, capacity int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleCapacity,
		prometheus.GaugeValue,
		float64(capacity),
		id,
	)
}

func (mc *Collector) NewMemoryModuleSpeed(id string, speed int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.MemoryModuleSpeed,
		prometheus.GaugeValue,
		float64(speed),
		id,
	)
}

func (mc *Collector) NewNetworkInterfaceHealth(id, health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.NetworkInterfaceHealth,
		prometheus.GaugeValue,
		value,
		id,
		health,
	)
}

func (mc *Collector) NewNetworkPortHealth(iface, id, health string) prometheus.Metric {
	value := health2value(health)
	return prometheus.MustNewConstMetric(
		mc.NetworkPortHealth,
		prometheus.GaugeValue,
		value,
		iface,
		id,
		health,
	)
}

func (mc *Collector) NewNetworkPortSpeed(iface, id string, speed int) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		mc.NetworkPortSpeed,
		prometheus.GaugeValue,
		float64(speed),
		iface,
		id,
	)
}

func (mc *Collector) NewNetworkPortLinkUp(iface, id, status string) prometheus.Metric {
	value := linkstatus2value(status)
	return prometheus.MustNewConstMetric(
		mc.NetworkPortLinkUp,
		prometheus.GaugeValue,
		value,
		iface,
		id,
		status,
	)
}
