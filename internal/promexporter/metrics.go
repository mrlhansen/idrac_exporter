package promexporter

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type dict map[string]string

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
	s.appendMetric("memory_size", memory, nil)
}

func (s *MetricsStore) SetCpuCount(numCpus int, model string) {
	labels := dict{
		"model": model,
	}
	s.appendMetric("cpu_count", float64(numCpus), labels)
}

func (s *MetricsStore) SetBiosVersion(version string) {
	value := 1.0
	labels := dict{
		"version": version,
	}
	s.appendMetric("bios_info", value, labels)
}

func (s *MetricsStore) SetMachineInfo(manufacturer, model, serial, sku string) {
	value := 1.0
	labels := make(dict)
	if manufacturer != "" {
		labels["manufacturer"] = manufacturer
	}
	if model != "" {
		labels["model"] = model
	}
	if serial != "" {
		labels["serial"] = serial
	}
	if sku != "" {
		labels["sku"] = sku
	}
	if len(labels) > 0 {
		s.appendMetric("machine_info", value, labels)
	}
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
	s.appendMetric("sensors_tachometer", speed, labels)
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
	_, _ = s.builder.WriteString(s.prefix + name)

	if length := len(labels); length > 0 {
		_, _ = s.builder.WriteRune('{')
		for k, v := range labels {
			_, _ = s.builder.WriteString(fmt.Sprintf("%s=%q", k, strings.TrimSpace(v)))
			length--
			if length > 0 {
				_, _ = s.builder.WriteRune(',')
			}
		}
		_, _ = s.builder.WriteRune('}')
	}

	s.builder.WriteRune(' ')

	if value == math.Trunc(value) {
		_, _ = s.builder.WriteString(strconv.FormatFloat(value, 'f', 0, 64))
	} else {
		_, _ = s.builder.WriteString(strconv.FormatFloat(value, 'g', 4, 64))
	}

	s.builder.WriteRune('\n')
}
