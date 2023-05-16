package promexporter

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type dict map[string]string

func health2value(health string) int {
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

// MetricsStore can be used to accumulate metrics
type MetricsStore struct {
	prefix  string
	builder *strings.Builder
}

func NewMetricsStore(prefix string) *MetricsStore {
	if prefix != "" {
		// Separate the prefix with an underscore
		prefix += "_"
	}
	return &MetricsStore{
		prefix,
		new(strings.Builder),
	}
}

func (s *MetricsStore) SetPowerOn(on bool) {
	var value float64
	if on {
		value = 1
	}
	s.appendMetric("power_on", value, nil)
}

func (s *MetricsStore) SetHealthOk(ok bool, status string) {
	var value float64
	if ok {
		value = 1
	}
	labels := dict{
		"status": status,
	}
	s.appendMetric("health_ok", value, labels)
}

func (s *MetricsStore) SetLedOn(on bool, state string) {
	var value float64
	if on {
		value = 1
	}
	labels := dict{
		"state": state,
	}
	s.appendMetric("indicator_led_on", value, labels)
}

func (s *MetricsStore) SetMemorySize(memory float64) {
	s.appendMetric("memory_size_bytes", memory, nil)
}

func (s *MetricsStore) SetCpuCount(numCpus int, model string) {
	labels := dict{
		"model": model,
	}
	s.appendMetric("cpu_count", float64(numCpus), labels)
}

func (s *MetricsStore) SetBiosInfo(version string) {
	value := 1.0
	labels := dict{
		"version": version,
	}
	s.appendMetric("bios_info", value, labels)
}

func (s *MetricsStore) SetMachineInfo(manufacturer, model, serial, sku string) {
	value := 1.0
	labels := dict{
		"manufacturer": manufacturer,
		"model": model,
		"serial": serial,
		"sku": sku,
	}
	s.appendMetric("machine_info", value, labels)
}

func (s *MetricsStore) SetTemperature(temperature float64, name, units string) {
	labels := dict{
		"name":  name,
		"units": units,
	}
	s.appendMetric("sensors_temperature", temperature, labels)
}

func (s *MetricsStore) SetFanSpeed(speed float64, name, units string) {
	labels := dict{
		"name":  name,
		"units": units,
	}
	s.appendMetric("sensors_fan_speed", speed, labels)
}

func (s *MetricsStore) SetPowerSupplyInputWatts(value float64, id string) {
	labels := dict{
		"psu": id,
	}
	s.appendMetric("power_supply_input_watts", value, labels)
}

func (s *MetricsStore) SetPowerSupplyInputVoltage(value float64, id string) {
	labels := dict{
		"psu": id,
	}
	s.appendMetric("power_supply_input_voltage", value, labels)
}

func (s *MetricsStore) SetPowerSupplyOutputWatts(value float64, id string) {
	labels := dict{
		"psu": id,
	}
	s.appendMetric("power_supply_output_watts", value, labels)
}

func (s *MetricsStore) SetPowerSupplyCapacityWatts(value float64, id string) {
	labels := dict{
		"psu": id,
	}
	s.appendMetric("power_supply_capacity_watts", value, labels)
}

func (s *MetricsStore) SetPowerSupplyEfficiencyPercent(value float64, id string) {
	labels := dict{
		"psu": id,
	}
	s.appendMetric("power_supply_efficiency_percent", value, labels)
}

func (s *MetricsStore) SetPowerControlConsumedWatts(value float64, id, name string) {
	labels := dict{
		"id":   id,
		"name": name,
	}
	s.appendMetric("power_control_consumed_watts", value, labels)
}

func (s *MetricsStore) SetPowerControlMinConsumedWatts(value float64, id, name string) {
	labels := dict{
		"id":   id,
		"name": name,
	}
	s.appendMetric("power_control_min_consumed_watts", value, labels)
}

func (s *MetricsStore) SetPowerControlMaxConsumedWatts(value float64, id, name string) {
	labels := dict{
		"id":   id,
		"name": name,
	}
	s.appendMetric("power_control_max_consumed_watts", value, labels)
}

func (s *MetricsStore) SetPowerControlAvgConsumedWatts(value float64, id, name string) {
	labels := dict{
		"id":   id,
		"name": name,
	}
	s.appendMetric("power_control_avg_consumed_watts", value, labels)
}

func (s *MetricsStore) SetPowerControlCapacityWatts(value float64, id, name string) {
	labels := dict{
		"id":   id,
		"name": name,
	}
	s.appendMetric("power_control_capacity_watts", value, labels)
}

func (s *MetricsStore) SetPowerControlInterval(interval int, id, name string) {
	labels := dict{
		"id":   id,
		"name": name,
	}
	s.appendMetric("power_control_interval_in_minutes", float64(interval), labels)
}

func (s *MetricsStore) AddSelEntry(id string, message string, component string, severity string, created time.Time) {
	labels := dict{
		"id":        id,
		"message":   message,
		"component": component,
		"severity":  severity,
	}
	s.appendMetric("sel_entry", float64(created.Unix()), labels)
}

func (s *MetricsStore) SetDriveInfo(id, name, manufacturer, model, serial, mediatype, protocol string, slot int) {
	var slotstr string

	if slot < 0 {
		slotstr = ""
	} else {
		slotstr = fmt.Sprint(slot)
	}

	labels := dict{
		"id":           id,
		"name":         name,
		"manufacturer": manufacturer,
		"model":        model,
		"serial":       serial,
		"mediatype":    mediatype,
		"protocol":     protocol,
		"slot":         slotstr,
	}
	s.appendMetric("drive_info", 1.0, labels)
}

func (s *MetricsStore) SetDriveHealth(id, health string) {
	value := health2value(health)
	labels := dict{
		"id":    id,
		"status": health,
	}
	s.appendMetric("drive_health", float64(value), labels)
}

func (s *MetricsStore) SetDriveCapacity(id string, capacity int) {
	labels := dict{
		"id": id,
	}
	s.appendMetric("drive_capacity_bytes", float64(capacity), labels)
}

func (s *MetricsStore) SetMemoryInfo(id, name, manufacturer, memtype, serial, ecc string, rank int) {
	labels := dict{
		"id":           id,
		"name":         name,
		"manufacturer": manufacturer,
		"type":         memtype,
		"serial":       serial,
		"ecc":          ecc,
		"rank":         fmt.Sprint(rank),
	}
	s.appendMetric("memory_module_info", 1.0, labels)
}

func (s *MetricsStore) SetMemoryHealth(id, health string) {
	value := health2value(health)
	labels := dict{
		"id":    id,
		"status": health,
	}
	s.appendMetric("memory_module_health", float64(value), labels)
}

func (s *MetricsStore) SetMemoryCapacity(id string, capacity int) {
	labels := dict{
		"id": id,
	}
	s.appendMetric("memory_module_capacity_bytes", float64(capacity), labels)
}

func (s *MetricsStore) SetMemorySpeed(id string, speed int) {
	labels := dict{
		"id": id,
	}
	s.appendMetric("memory_module_speed_mhz", float64(speed), labels)
}


// Reset the accumulated string in the MetricsStore buffer
func (s *MetricsStore) Reset() {
	s.builder.Reset()
}

// Gather returns the accumulated string in the MetricsStore buffer representing the metrics in OpenMetrics format
func (s *MetricsStore) Gather() string {
	return s.builder.String()
}

// appendMetric appends the given metric to the current metrics list
func (s *MetricsStore) appendMetric(name string, value float64, labels dict) {
	s.builder.WriteString(s.prefix + name)

	keys := []string{}
	for k, v := range labels {
		if len(v) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	if length := len(keys); length > 0 {
		s.builder.WriteRune('{')

		for _, k := range keys {
			s.builder.WriteString(fmt.Sprintf("%s=%q", k, strings.TrimSpace(labels[k])))
			length--
			if length > 0 {
				s.builder.WriteRune(',')
			}
		}

		s.builder.WriteRune('}')
	}

	s.builder.WriteRune(' ')

	if value == math.Trunc(value) {
		s.builder.WriteString(strconv.FormatFloat(value, 'f', 0, 64))
	} else {
		s.builder.WriteString(strconv.FormatFloat(value, 'g', 4, 64))
	}

	s.builder.WriteRune('\n')
}
