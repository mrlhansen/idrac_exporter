package main

import (
	"log"
	"time"
	"math"
	"strings"
	"strconv"
	"io/ioutil"
	"net/http"
	"crypto/tls"
	"encoding/json"
)

var transport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

var client = &http.Client{
	Transport: transport,
	Timeout: time.Duration(config.Timeout)*time.Second,
}

type dict = map[string]interface{}
type list = []interface{}
type stringmap = map[string]string

func redfishGet(host *HostConfig, path string) (dict, bool) {
	var result dict

	url := "https://" + host.Hostname + path
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Basic " + host.Token)
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		log.Print(err)
		return result, false
	}

	body, err := ioutil.ReadAll(resp.Body)
	s := []byte(body)

	if resp.StatusCode != 200 {
		return result, false
	}

	json.Unmarshal(s, &result)
	return result, true
}

func redfishFindAllEndpoints(host *HostConfig) bool {
	root, ok := redfishGet(host, "/redfish/v1/")
	if !ok {
		return false
	}

	// Systems
	collection := root["Systems"].(dict)
	url := collection["@odata.id"].(string)

	data, ok := redfishGet(host, url)
	if !ok {
		return false
	}

	members := data["Members"].(list)
	entry := members[0].(dict)
	host.SystemEndpoint = entry["@odata.id"].(string)

	// Chassis
	collection = root["Chassis"].(dict)
	url = collection["@odata.id"].(string)

	data, ok = redfishGet(host, url)
	if !ok {
		return false
	}

	members = data["Members"].(list)
	entry = members[0].(dict)
	url = entry["@odata.id"].(string)

	// Thermal and Power
	data, ok = redfishGet(host, url)
	if !ok {
		return false
	}

	collection = data["Thermal"].(dict)
	host.ThermalEndpoint = collection["@odata.id"].(string)

	collection = data["Power"].(dict)
	host.PowerEndpoint = collection["@odata.id"].(string)

	return true
}

func redfishSensors(host *HostConfig) bool {
	var name string
	var value float64
	var args stringmap
	var entry dict
	var status dict
	var units string

	data, ok := redfishGet(host, host.ThermalEndpoint)
	if !ok {
		return false
	}

	temp := data["Temperatures"].(list)
	for _, v := range temp {
		entry = v.(dict)

		status = entry["Status"].(dict)
		if status["State"] != "Enabled" {
			continue
		}

		args = stringmap{
			"name": entry["Name"].(string),
			"units": "celsius",
		}

		value = entry["ReadingCelsius"].(float64)
		if value < 0 {
			continue
		}

		metricsAppend(host, "sensors_temperature", args, value)
	}

	fans := data["Fans"].(list)
	for _, v := range fans {
		entry = v.(dict)
		status = entry["Status"].(dict)

		if status["State"] != "Enabled" {
			continue
		}

		name, ok = entry["Name"].(string)
		if !ok {
			name, ok = entry["FanName"].(string)
			if !ok {
				continue
			}
		}

		units, ok = entry["ReadingUnits"].(string)
		if !ok {
			units, ok = entry["Units"].(string)
			if !ok {
				continue
			}
		}

		value, ok = entry["Reading"].(float64)
		if !ok {
			value, ok = entry["CurrentReading"].(float64)
			if !ok {
				continue
			}
		}

		args = stringmap{
			"name": name,
			"units": strings.ToLower(units),
		}

		metricsAppend(host, "sensors_tachometer", args, value)
	}

	return true
}

func redfishSystem(host *HostConfig) bool {
	var text string
	var value float64
	var args stringmap
	var entry dict

	data, ok := redfishGet(host, host.SystemEndpoint)
	if !ok {
		return false
	}

	if data["PowerState"] == "On" {
		value = 1
	} else {
		value = 0
	}
	metricsAppend(host, "power_on", nil, value)

	entry = data["Status"].(dict)
	text = entry["Health"].(string)
	args = stringmap{"status": text}
	if text == "OK" {
		value = 1
	} else {
		value = 0
	}
	metricsAppend(host, "health_ok", args, value)

	if data["IndicatorLED"] == "Off" {
		value = 0
	} else {
		value = 1
	}
	metricsAppend(host, "indicator_led_on", nil, value)

	entry = data["MemorySummary"].(dict)
	value = entry["TotalSystemMemoryGiB"].(float64) // depending on the bios version, this is reported in either GB or GiB
	if value == math.Trunc(value) {
		value = value * 1024
	} else {
		value = math.Floor(value*1099511627776.0/1000000000.0)
	}
	metricsAppend(host, "memory_size", nil, value)

	entry = data["ProcessorSummary"].(dict)
	text = entry["Model"].(string)
	value = entry["Count"].(float64)
	args = stringmap{"model": text}
	metricsAppend(host, "cpu_count", args, value)

	text = data["BiosVersion"].(string)
	args = stringmap{"version": text}
	metricsAppend(host, "bios_version", args, -1)

	return true
}

func redfishSEL(host *HostConfig) bool {
	var args stringmap
	var text string
	var value float64

	data, ok := redfishGet(host, "/redfish/v1/Managers/iDRAC.Embedded.1/Logs/Sel")
	if !ok {
		return false
	}

	members := data["Members"].(list)
	for _, v := range members {
		entry := v.(dict)
		component, _ := entry["SensorType"].(string) // sometimes reported as null

		args = stringmap{
			"id": entry["Id"].(string),
			"message": entry["Message"].(string),
			"component": component,
			"severity": entry["Severity"].(string),
		}

		text = entry["Created"].(string)
		tm, err := time.Parse(time.RFC3339, text)
		if err != nil {
			log.Print(err)
			return false
		}

		value = float64(tm.Unix())
		metricsAppend(host, "sel_entry", args, value)
	}

	return true
}

func redfishPower(host *HostConfig) bool {
	var entry dict
	var status dict
	var args stringmap
	var value float64

	data, ok := redfishGet(host, host.PowerEndpoint)
	if !ok {
		return false
	}

	psu := data["PowerSupplies"].(list)
	for i, v := range psu {
		entry = v.(dict)

		status = entry["Status"].(dict)
		if status["State"] != "Enabled" {
			continue
		}

		args = stringmap{
			"psu": strconv.Itoa(i),
		}

		value, ok = entry["PowerOutputWatts"].(float64)
		if !ok {
			value, ok = entry["LastPowerOutputWatts"].(float64)
		}
		if ok {
			metricsAppend(host, "power_output_watts", args, value)
		}

		value, ok = entry["PowerInputWatts"].(float64)
		if ok {
			metricsAppend(host, "power_input_watts", args, value)
		}

		value, ok = entry["PowerCapacityWatts"].(float64)
		if ok {
			metricsAppend(host, "power_capacity_watts", args, value)
		}

		value, ok = entry["LineInputVoltage"].(float64)
		if ok {
			metricsAppend(host, "power_input_voltage", args, value)
		}

		value, ok = entry["EfficiencyPercent"].(float64)
		if ok {
			metricsAppend(host, "power_efficiency_percent", args, value)
		}
	}

	return true
}
