package redfish

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mrlhansen/idrac_exporter/internals/config"
)

const redfishRootPath = "/redfish/v1"

var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: time.Duration(config.Config.Timeout) * time.Second,
}

type stringmap = map[string]string

func FindAllEndpoints(host *config.HostConfig) error {
	var resp V1Response
	if err := redfishGet(host, redfishRootPath, &resp); err != nil {
		return err
	}

	var sysResp GroupResponse
	if err := redfishGet(host, resp.Systems.OdataId, &sysResp); err != nil {
		return err
	}

	host.SystemEndpoint = sysResp.Members[0].OdataId

	// Chassis
	var chSysResp GroupResponse
	if err := redfishGet(host, resp.Chassis.OdataId, &chSysResp); err != nil {
		return err
	}

	// Thermal and Power
	var chResponse ChassisResponse
	if err := redfishGet(host, chSysResp.Members[0].OdataId, &chResponse); err != nil {
		return err
	}

	host.ThermalEndpoint = chResponse.Thermal.OdataId
	host.PowerEndpoint = chResponse.Power.OdataId

	return nil
}

func Sensors(host *config.HostConfig) error {
	var resp ThermalResponse
	if err := redfishGet(host, host.ThermalEndpoint, &resp); err != nil {
		return err
	}

	for _, t := range resp.Temperatures {
		if t.Status.State != StateEnabled {
			continue
		}

		args := stringmap{
			"name":  t.Name,
			"units": "celsius",
		}

		metricsAppend(host, "sensors_temperature", args, t.ReadingCelsius)
	}

	for _, f := range resp.Fans {
		status := f.Status

		if status.State != StateEnabled {
			continue
		}

		name := f.GetName()
		if name == "" {
			continue
		}

		units := f.GetUnits()
		if units == "" {
			continue
		}

		args := stringmap{
			"name":  name,
			"units": strings.ToLower(units),
		}

		metricsAppend(host, "sensors_tachometer", args, float64(f.GetReading()))
	}

	return nil
}

func System(host *config.HostConfig) error {
	var value float64
	var labels stringmap

	var sysResp SystemResponse
	if err := redfishGet(host, host.SystemEndpoint, &sysResp); err != nil {
		return err
	}

	if sysResp.PowerState == "On" {
		value = 1
	} else {
		value = 0
	}
	metricsAppend(host, "power_on", nil, value)

	if sysResp.Status.Health == "OK" {
		value = 1
	} else {
		value = 0
	}
	metricsAppend(host, "health_ok", stringmap{"status": sysResp.Status.Health}, value)

	if sysResp.IndicatorLED == "Off" {
		value = 0
	} else {
		value = 1
	}
	metricsAppend(host, "indicator_led_on", stringmap{"state": sysResp.IndicatorLED}, value)

	value = sysResp.MemorySummary.TotalSystemMemoryGiB // depending on the bios version, this is reported in either GB or GiB
	if value == math.Trunc(value) {
		value = value * 1024
	} else {
		value = math.Floor(value * 1099511627776.0 / 1e9)
	}
	metricsAppend(host, "memory_size", nil, value)

	if sysResp.ProcessorSummary.Model != "" {
		labels = stringmap{"model": sysResp.ProcessorSummary.Model}
	}
	metricsAppend(host, "cpu_count", labels, float64(sysResp.ProcessorSummary.Count))

	metricsAppend(host, "bios_version", stringmap{"version": sysResp.BiosVersion}, -1)

	labels = make(stringmap)
	if sysResp.Manufacturer != "" {
		labels["manufacturer"] = sysResp.Manufacturer
	}
	if sysResp.Model != "" {
		labels["model"] = sysResp.Model
	}
	if sysResp.SerialNumber != "" {
		labels["serial"] = sysResp.SerialNumber
	}
	if sysResp.Sku != "" {
		labels["sku"] = sysResp.Sku
	}

	metricsAppend(host, "machine", labels, -1)

	return nil
}

func IdracSel(host *config.HostConfig) error {
	var args stringmap

	var resp IdracSelResponse
	if err := redfishGet(host, "/redfish/v1/Managers/iDRAC.Embedded.1/Logs/Sel", &resp); err != nil {
		return err
	}

	for _, e := range resp.Members {
		args = stringmap{
			"id":        e.Id,
			"message":   e.Message,
			"component": e.SensorType, // sometimes reported as null
			"severity":  e.Severity,
		}

		metricsAppend(host, "sel_entry", args, float64(e.Created.Unix()))
	}

	return nil
}

func Power(host *config.HostConfig) error {
	var args stringmap

	var resp PowerResponse
	if err := redfishGet(host, host.PowerEndpoint, &resp); err != nil {
		return err
	}

	for i, psu := range resp.PowerSupplies {
		if psu.Status.State != StateEnabled {
			continue
		}

		args = stringmap{
			"psu": strconv.Itoa(i),
		}

		if psu.PowerOutputWatts > 0 {
			metricsAppend(host, "power_supply_output_watts", args, psu.PowerOutputWatts)
		} else if psu.LastPowerOutputWatts > 0 {
			metricsAppend(host, "power_supply_output_watts", args, psu.LastPowerOutputWatts)
		}
		metricsAppend(host, "power_supply_input_watts", args, psu.PowerInputWatts)
		metricsAppend(host, "power_supply_capacity_watts", args, psu.PowerCapacityWatts)
		metricsAppend(host, "power_supply_input_voltage", args, psu.LineInputVoltage)
		metricsAppend(host, "power_supply_efficiency_percent", args, psu.EfficiencyPercent)
	}

	for i, pc := range resp.PowerControl {
		args = stringmap{
			"id": strconv.Itoa(i),
		}

		if pc.Name != "" {
			args["name"] = pc.Name
		}

		metricsAppend(host, "power_control_consumed_watts", args, pc.PowerConsumedWatts)
		metricsAppend(host, "power_control_capacity_watts", args, pc.PowerCapacityWatts)

		if pc.PowerMetrics == nil {
			continue
		}
		pm := pc.PowerMetrics

		metricsAppend(host, "power_control_min_consumed_watts", args, pm.MinConsumedWatts)
		metricsAppend(host, "power_control_max_consumed_watts", args, pm.MaxConsumedWatts)
		metricsAppend(host, "power_control_avg_consumed_watts", args, pm.AverageConsumedWatts)
		metricsAppend(host, "power_control_interval_in_minutes", args, pm.IntervalInMin)
	}

	return nil
}

func redfishGet(host *config.HostConfig, path string, res any) error {
	url := "https://" + host.Hostname + path
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Basic "+host.Token)
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(res)
	if err != nil {
		return err
	}

	return nil
}
