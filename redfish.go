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
	var args map[string]string

	data, ok := redfishGet(host, "Dell/Systems/System.Embedded.1/DellNumericSensorCollection")
	if !ok {
		return
	}

	members := data["Members"].([]interface{})
	for _, v := range members {
		v := v.(jsonmap)

		if v["EnabledState"] == "Enabled" {
			enabled = "1"
		} else {
			enabled = "0"
		}

		args = map[string]string{
			"name": v["ElementName"].(string),
			"id": v["DeviceID"].(string),
			"enabled": enabled,
		}

		if v["SensorType"] == "Temperature" {
			value = v["CurrentReading"].(float64)/10.0
			name = "sensors_temperature"
		} else if v["SensorType"] == "Tachometer" {
			value = v["CurrentReading"].(float64)
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
	var args map[string]string
	var entry map[string]interface{}

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

	entry = data["Status"].(map[string]interface{})
	text = entry["Health"].(string)
	args = map[string]string{"status": text}
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

	entry = data["MemorySummary"].(map[string]interface{})
	value = entry["TotalSystemMemoryGiB"].(float64)
	value = math.Floor(value*1099511627776.0/1000000000.0)
	metricsAppend(host, "memory_size", nil, value)

	entry = data["ProcessorSummary"].(map[string]interface{})
	text = entry["Model"].(string)
	value = entry["Count"].(float64)
	args = map[string]string{"model": text}
	metricsAppend(host, "cpu_count", args, value)

	text = data["BiosVersion"].(string)
	args = map[string]string{"version": text}
	metricsAppend(host, "bios_version", args, -1)
}

func redfishSEL(host *HostConfig) {
	var args map[string]string
	var text string
	var value float64

	data, ok := redfishGet(host, "Managers/iDRAC.Embedded.1/Logs/Sel")
	if !ok {
		return
	}

	members := data["Members"].([]interface{})
	for _, v := range members {
		entry := v.(jsonmap)

		args = map[string]string{
			"id" : entry["Id"].(string),
			"message" : entry["Message"].(string),
			"component" : entry["SensorType"].(string),
			"severity" : entry["Severity"].(string),
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
