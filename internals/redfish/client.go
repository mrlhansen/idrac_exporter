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
	"github.com/mrlhansen/idrac_exporter/internals/logging"
)

const redfishRootPath = "/redfish/v1"

var logger = logging.NewLogger().Sugar()

type Client struct {
	hostname  string
	basicAuth string

	httpClient *http.Client

	systemPath  string
	thermalPath string
	powerPath   string
}

func NewClient(hostConfig *config.HostConfig) (*Client, error) {
	h := &Client{
		hostname:   hostConfig.Hostname,
		basicAuth:  hostConfig.Token,
		httpClient: newHttpClient(),
	}

	if err := h.findAllEndpoints(); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *Client) findAllEndpoints() error {
	var resp V1Response
	if err := h.redfishGet(redfishRootPath, &resp); err != nil {
		return err
	}

	var sysResp GroupResponse
	if err := h.redfishGet(resp.Systems.OdataId, &sysResp); err != nil {
		return err
	}

	h.systemPath = sysResp.Members[0].OdataId

	// Chassis
	var chSysResp GroupResponse
	if err := h.redfishGet(resp.Chassis.OdataId, &chSysResp); err != nil {
		return err
	}

	// Thermal and Power
	var chResponse ChassisResponse
	if err := h.redfishGet(chSysResp.Members[0].OdataId, &chResponse); err != nil {
		return err
	}

	h.thermalPath = chResponse.Thermal.OdataId
	h.powerPath = chResponse.Power.OdataId

	return nil
}

func (h *Client) RefreshSensors(store sensorsMetricsStore) error {
	var resp ThermalResponse
	if err := h.redfishGet(h.thermalPath, &resp); err != nil {
		return err
	}

	for _, t := range resp.Temperatures {
		if t.Status.State != StateEnabled {
			continue
		}
		store.SetTemperature(t.ReadingCelsius, t.Name, "celsius")
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

		store.SetFanSpeed(f.GetReading(), name, strings.ToLower(units))
	}

	return nil
}

func (h *Client) RefreshSystem(store systemMetricsStore) error {
	var sysResp SystemResponse
	if err := h.redfishGet(h.systemPath, &sysResp); err != nil {
		return err
	}

	store.SetPowerOn(sysResp.PowerState == "On")
	store.SetHealthOk(sysResp.Status.Health == "OK", sysResp.Status.Health)
	store.SetLedOn(sysResp.IndicatorLED != "Off", sysResp.IndicatorLED)

	value := sysResp.MemorySummary.TotalSystemMemoryGiB // depending on the bios version, this is reported in either GB or GiB
	if value == math.Trunc(value) {
		value = value * 1024
	} else {
		value = math.Floor(value * 1099511627776.0 / 1e9)
	}
	store.SetMemorySize(value)

	store.SetCpuCount(sysResp.ProcessorSummary.Count, sysResp.ProcessorSummary.Model)
	store.SetBiosVersion(sysResp.BiosVersion)
	store.SetMachineInfo(sysResp.Manufacturer, sysResp.Model, sysResp.SerialNumber, sysResp.Sku)

	return nil
}

func (h *Client) RefreshPower(store powerMetricsStore) error {
	var resp PowerResponse
	if err := h.redfishGet(h.powerPath, &resp); err != nil {
		return err
	}

	var psuId string
	for i, psu := range resp.PowerSupplies {
		if psu.Status.State != StateEnabled {
			continue
		}

		psuId = strconv.Itoa(i)

		store.SetPowerSupplyInputWatts(psu.PowerInputWatts, psuId)
		store.SetPowerSupplyInputVoltage(psu.LineInputVoltage, psuId)
		store.SetPowerSupplyOutputWatts(psu.GetOutputPower(), psuId)
		store.SetPowerSupplyCapacityWatts(psu.PowerCapacityWatts, psuId)
		store.SetPowerSupplyEfficiencyPercent(psu.EfficiencyPercent, psuId)
	}

	var pcId string
	for i, pc := range resp.PowerControl {
		pcId = strconv.Itoa(i)

		store.SetPowerControlConsumedWatts(pc.PowerConsumedWatts, pcId, pc.Name)
		store.SetPowerControlCapacityWatts(pc.PowerCapacityWatts, pcId, pc.Name)

		if pc.PowerMetrics == nil {
			continue
		}
		pm := pc.PowerMetrics

		store.SetPowerControlMinConsumedWatts(pm.MinConsumedWatts, pcId, pc.Name)
		store.SetPowerControlMaxConsumedWatts(pm.MaxConsumedWatts, pcId, pc.Name)
		store.SetPowerControlAvgConsumedWatts(pm.AverageConsumedWatts, pcId, pc.Name)
		store.SetPowerControlInterval(pm.IntervalInMin, pcId, pc.Name)
	}

	return nil
}

func (h *Client) RefreshIdracSel(store selStore) error {
	var resp IdracSelResponse
	if err := h.redfishGet(redfishRootPath+"/Managers/iDRAC.Embedded.1/Logs/Sel", &resp); err != nil {
		return err
	}

	for _, e := range resp.Members {
		store.AddSelEntry(e.Id, e.Message, e.SensorType, e.Severity, e.Created)
	}

	return nil
}

func (h *Client) redfishGet(path string, res interface{}) error {
	url := "https://" + h.hostname + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Basic "+h.basicAuth)
	req.Header.Add("Accept", "application/json")

	logger.Debugf("Querying url %q", url)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		logger.Debugf("Failed to query url %q: %v", url, err)
		return err
	}

	if resp.StatusCode != 200 {
		logger.Debugf("Query to url %q returned unexpected status code: %d (%s)", url, resp.StatusCode, resp.Status)
		return fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	if err = json.NewDecoder(resp.Body).Decode(res); err != nil {
		logger.Debugf("Error decoding response from url %q: %v", url, err)
		return err
	}

	return nil
}

func newHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Duration(config.Config.Timeout) * time.Second,
	}
}

// Store interfaces

type systemMetricsStore interface {
	SetPowerOn(on bool)
	SetHealthOk(ok bool, status string)
	SetLedOn(on bool, state string)
	SetMemorySize(memory float64)
	SetCpuCount(numCpus int, model string)
	SetBiosVersion(version string)
	SetMachineInfo(manufacturer, model, serial, sku string)
}

type sensorsMetricsStore interface {
	SetTemperature(temperature float64, name, units string)
	SetFanSpeed(speed float64, name, units string)
}

type powerMetricsStore interface {
	SetPowerSupplyInputWatts(value float64, psuId string)
	SetPowerSupplyInputVoltage(value float64, psuId string)
	SetPowerSupplyOutputWatts(value float64, psuId string)
	SetPowerSupplyCapacityWatts(value float64, psuId string)
	SetPowerSupplyEfficiencyPercent(value float64, psuId string)

	SetPowerControlConsumedWatts(value float64, pcId, pcName string)
	SetPowerControlMinConsumedWatts(value float64, pcId, pcName string)
	SetPowerControlMaxConsumedWatts(value float64, pcId, pcName string)
	SetPowerControlAvgConsumedWatts(value float64, pcId, pcName string)
	SetPowerControlCapacityWatts(value float64, pcId, pcName string)
	SetPowerControlInterval(interval int, pcId, pcName string)
}

type selStore interface {
	AddSelEntry(id string, message string, component string, severity string, created time.Time)
}
