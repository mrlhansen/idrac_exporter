package promexporter

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type labels map[string]string

var nan = math.NaN()

// MetricsStore can be used to accumulate metrics
type MetricsStore struct {
	prefix string
	b      *strings.Builder
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
	s.appendMetric("health_ok", value, labels{"status": status})
}

func (s *MetricsStore) SetLedOn(on bool, state string) {
	var value float64
	if on {
		value = 1
	}
	s.appendMetric("indicator_led_on", value, labels{"state": state})
}

func (s *MetricsStore) SetMemorySize(memory float64) {
	s.appendMetric("memory_size", memory, nil)
}

func (s *MetricsStore) SetCpuCount(numCpus int, model string) {
	s.appendMetric("cpu_count", float64(numCpus), labels{"model": model})
}

func (s *MetricsStore) SetBiosVersion(version string) {
	s.appendMetric("bios_version", nan, labels{"version": version})
}

func (s *MetricsStore) SetMachineInfo(manufacturer, model, serial, sku string) {
	l := make(labels)
	if manufacturer != "" {
		l["manufacturer"] = manufacturer
	}
	if model != "" {
		l["model"] = model
	}
	if serial != "" {
		l["serial"] = serial
	}
	if sku != "" {
		l["sku"] = sku
	}
	if len(l) > 0 {
		s.appendMetric("machine", nan, l)
	}
}

func (s *MetricsStore) SetTemperature(temperature float64, name, units string) {
	s.appendMetric("sensors_temperature", temperature, labels{"name": name, "units": units})
}

func (s *MetricsStore) SetFanSpeed(speed float64, name, units string) {
	s.appendMetric("sensors_tachometer", speed, labels{"name": name, "units": units})
}

func (s *MetricsStore) SetPowerSupplyInputWatts(value float64, psuId string) {
	s.appendMetric("power_supply_input_watts", value, labels{"id": psuId})
}

func (s *MetricsStore) SetPowerSupplyInputVoltage(value float64, psuId string) {
	s.appendMetric("power_supply_input_voltage", value, labels{"id": psuId})
}

func (s *MetricsStore) SetPowerSupplyOutputWatts(value float64, psuId string) {
	s.appendMetric("power_supply_output_watts", value, labels{"id": psuId})
}

func (s *MetricsStore) SetPowerSupplyCapacityWatts(value float64, psuId string) {
	s.appendMetric("power_supply_capacity_watts", value, labels{"id": psuId})
}

func (s *MetricsStore) SetPowerSupplyEfficiencyPercent(value float64, psuId string) {
	s.appendMetric("power_supply_efficiency_percent", value, labels{"id": psuId})
}

func (s *MetricsStore) SetPowerControlConsumedWatts(value float64, pcId, pcName string) {
	s.appendMetric("power_control_consumed_watts", value, labels{"id": pcId, "name": pcName})
}

func (s *MetricsStore) SetPowerControlMinConsumedWatts(value float64, pcId, pcName string) {
	s.appendMetric("power_control_min_consumed_watts", value, labels{"id": pcId, "name": pcName})
}

func (s *MetricsStore) SetPowerControlMaxConsumedWatts(value float64, pcId, pcName string) {
	s.appendMetric("power_control_max_consumed_watts", value, labels{"id": pcId, "name": pcName})
}

func (s *MetricsStore) SetPowerControlAvgConsumedWatts(value float64, pcId, pcName string) {
	s.appendMetric("power_control_avg_consumed_watts", value, labels{"id": pcId, "name": pcName})
}

func (s *MetricsStore) SetPowerControlCapacityWatts(value float64, pcId, pcName string) {
	s.appendMetric("power_control_capacity_watts", value, labels{"id": pcId, "name": pcName})
}

func (s *MetricsStore) SetPowerControlInterval(interval int, pcId, pcName string) {
	s.appendMetric("power_control_interval_in_minutes", float64(interval), labels{"id": pcId, "name": pcName})
}

func (s *MetricsStore) AddSelEntry(id string, message string, component string, severity string, created time.Time) {
	s.appendMetric("sel_entry", float64(created.Unix()), labels{"id": id, "message": message, "component": component, "severity": severity})
}

// Reset the accumulated string in the MetricsStore buffer
func (s *MetricsStore) Reset() {
	s.b.Reset()
}

// Gather returns the accumulated string in the MetricsStore buffer representing the metrics in OpenMetrics format
func (s *MetricsStore) Gather() string {
	return s.b.String()
}

// appendMetric appends the given metric to the current metrics list
func (s *MetricsStore) appendMetric(name string, value float64, labels labels) {
	_, _ = s.b.WriteString(s.prefix + name)

	if length := len(labels); length > 0 {
		_, _ = s.b.WriteRune('{')
		for k, v := range labels {
			_, _ = s.b.WriteString(fmt.Sprintf("%s=%q", k, strings.TrimSpace(v)))
			length--
			if length > 0 {
				_, _ = s.b.WriteRune(',')
			}
		}
		_, _ = s.b.WriteRune('}')
	}

	s.b.WriteRune(' ')

	if value == math.Trunc(value) {
		_, _ = s.b.WriteString(strconv.FormatFloat(value, 'f', 0, 64))
	} else {
		_, _ = s.b.WriteString(strconv.FormatFloat(value, 'g', 4, 64))
	}

	s.b.WriteRune('\n')
}
