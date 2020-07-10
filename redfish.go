package main

import (
	"log"
	"time"
	"math"
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
	Timeout: time.Second,
}

type jsonmap = map[string]interface{}
type stringmap = map[string]string

func redfishGet(host *HostConfig, path string) (jsonmap, bool) {
	var result jsonmap

	url := "https://" + host.Hostname + "/redfish/v1/" + path
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

func redfishSensors(host *HostConfig) {
	var enabled string
	var name string
	var value float64
	var args stringmap
	var entry jsonmap

	data, ok := redfishGet(host, "Dell/Systems/System.Embedded.1/DellNumericSensorCollection")
	if !ok {
		return
	}

	members := data["Members"].([]interface{})
	for _, v := range members {
		entry = v.(jsonmap)

		if entry["EnabledState"] == "Enabled" {
			enabled = "1"
		} else {
			enabled = "0"
		}

		args = stringmap{
			"name": entry["ElementName"].(string),
			"id": entry["DeviceID"].(string),
			"enabled": enabled,
		}

		if entry["SensorType"] == "Temperature" {
			value = entry["CurrentReading"].(float64)/10.0
			name = "sensors_temperature"
		} else if entry["SensorType"] == "Tachometer" {
			value = entry["CurrentReading"].(float64)
			name = "sensors_tachometer"
		} else {
			continue
		}

		if value < 0 {
			value = 0
		}

		metricsAppend(host, name, args, value)
	}

}

func redfishSystem(host *HostConfig) {
	var text string
	var value float64
	var args stringmap
	var entry jsonmap

	data, ok := redfishGet(host, "Systems/System.Embedded.1")
	if !ok {
		return
	}

	if data["PowerState"] == "On" {
		value = 1
	} else {
		value = 0
	}
	metricsAppend(host, "power_on", nil, value)

	entry = data["Status"].(jsonmap)
	text = entry["Health"].(string)
	args = stringmap{"status": text}
	if text == "OK" {
		value = 1
	} else {
		value = 0
	}
	metricsAppend(host, "health_ok", args, value)

	if data["IndicatorLed"] == "Off" {
		value = 0
	} else {
		value = 1
	}
	metricsAppend(host, "indicator_led_on", nil, value)

	entry = data["MemorySummary"].(jsonmap)
	value = entry["TotalSystemMemoryGiB"].(float64)
	value = math.Floor(value*1099511627776.0/1000000000.0)
	metricsAppend(host, "memory_size", nil, value)

	entry = data["ProcessorSummary"].(jsonmap)
	text = entry["Model"].(string)
	value = entry["Count"].(float64)
	args = stringmap{"model": text}
	metricsAppend(host, "cpu_count", args, value)

	text = data["BiosVersion"].(string)
	args = stringmap{"version": text}
	metricsAppend(host, "bios_version", args, -1)
}

func redfishSEL(host *HostConfig) {
	var args stringmap
	var text string
	var value float64

	data, ok := redfishGet(host, "Managers/iDRAC.Embedded.1/Logs/Sel")
	if !ok {
		return
	}

	members := data["Members"].([]interface{})
	for _, v := range members {
		entry := v.(jsonmap)
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
			return
		}

		value = float64(time.Now().Unix() - tm.Unix())
		metricsAppend(host, "sel_entry", args, value)
	}
}
