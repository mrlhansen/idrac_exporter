package collector

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/logging"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const redfishRootPath = "/redfish/v1"

type Client struct {
	hostname    string
	basicAuth   string
	httpClient  *http.Client
	systemPath  string
	thermalPath string
	powerPath   string
	storagePath string
	memoryPath  string
}

func newHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Duration(config.Config.Timeout) * time.Second,
	}
}

func NewClient(hostConfig *config.HostConfig) (*Client, error) {
	client := &Client{
		hostname:   hostConfig.Hostname,
		basicAuth:  hostConfig.Token,
		httpClient: newHttpClient(),
	}

	err := client.findAllEndpoints()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (client *Client) findAllEndpoints() error {
	var root V1Response
	var group GroupResponse
	var chassis ChassisResponse
	var system SystemResponse
	var err error

	// Root
	err = client.redfishGet(redfishRootPath, &root)
	if err != nil {
		return err
	}

	// System
	err = client.redfishGet(root.Systems.OdataId, &group)
	if err != nil {
		return err
	}

	client.systemPath = group.Members[0].OdataId

	// Chassis
	err = client.redfishGet(root.Chassis.OdataId, &group)
	if err != nil {
		return err
	}

	// Thermal and Power
	err = client.redfishGet(group.Members[0].OdataId, &chassis)
	if err != nil {
		return err
	}

	err = client.redfishGet(client.systemPath, &system)
	if err != nil {
		return err
	}

	client.storagePath = system.Storage.OdataId
	client.memoryPath = system.Memory.OdataId
	client.thermalPath = chassis.Thermal.OdataId
	client.powerPath = chassis.Power.OdataId

	return nil
}

func (client *Client) RefreshSensors(mc *Collector, ch chan<- prometheus.Metric) error {
	var resp ThermalResponse

	err := client.redfishGet(client.thermalPath, &resp)
	if err != nil {
		return err
	}

	for _, t := range resp.Temperatures {
		if t.Status.State != StateEnabled {
			continue
		}

		if t.ReadingCelsius < 0 {
			continue
		}

		ch <- mc.NewSensorsTemperature(t.ReadingCelsius, t.MemberId, t.Name, "celsius")
	}

	for _, f := range resp.Fans {
		if f.Status.State != StateEnabled {
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

		ch <- mc.NewSensorsFanHealth(f.MemberId, name, f.Status.Health)
		ch <- mc.NewSensorsFanSpeed(f.GetReading(), f.MemberId, name, strings.ToLower(units))
	}

	return nil
}

func (client *Client) RefreshSystem(mc *Collector, ch chan<- prometheus.Metric) error {
	var resp SystemResponse

	err := client.redfishGet(client.systemPath, &resp)
	if err != nil {
		return err
	}

	ch <- mc.NewSystemPowerOn(resp.PowerState)
	ch <- mc.NewSystemHealth(resp.Status.Health)
	ch <- mc.NewSystemIndicatorLED(resp.IndicatorLED)
	ch <- mc.NewSystemMemorySize(resp.MemorySummary.TotalSystemMemoryGiB * 1073741824)
	ch <- mc.NewSystemCpuCount(resp.ProcessorSummary.Count, resp.ProcessorSummary.Model)
	ch <- mc.NewSystemBiosInfo(resp.BiosVersion)
	ch <- mc.NewSystemMachineInfo(resp.Manufacturer, resp.Model, resp.SerialNumber, resp.SKU)

	return nil
}

func (client *Client) RefreshPower(mc *Collector, ch chan<- prometheus.Metric) error {
	var resp PowerResponse

	err := client.redfishGet(client.powerPath, &resp)
	if err != nil {
		return err
	}

	for i, psu := range resp.PowerSupplies {
		if psu.Status.State != StateEnabled {
			continue
		}

		id := strconv.Itoa(i)
		ch <- mc.NewPowerSupplyHealth(psu.Status.Health, id)
		ch <- mc.NewPowerSupplyInputWatts(psu.PowerInputWatts, id)
		ch <- mc.NewPowerSupplyInputVoltage(psu.LineInputVoltage, id)
		ch <- mc.NewPowerSupplyOutputWatts(psu.GetOutputPower(), id)
		ch <- mc.NewPowerSupplyCapacityWatts(psu.PowerCapacityWatts, id)
		ch <- mc.NewPowerSupplyEfficiencyPercent(psu.EfficiencyPercent, id)
	}

	for i, pc := range resp.PowerControl {
		id := strconv.Itoa(i)
		ch <- mc.NewPowerControlConsumedWatts(pc.PowerConsumedWatts, id, pc.Name)
		ch <- mc.NewPowerControlCapacityWatts(pc.PowerCapacityWatts, id, pc.Name)

		if pc.PowerMetrics == nil {
			continue
		}

		pm := pc.PowerMetrics
		ch <- mc.NewPowerControlMinConsumedWatts(pm.MinConsumedWatts, id, pc.Name)
		ch <- mc.NewPowerControlMaxConsumedWatts(pm.MaxConsumedWatts, id, pc.Name)
		ch <- mc.NewPowerControlAvgConsumedWatts(pm.AverageConsumedWatts, id, pc.Name)
		ch <- mc.NewPowerControlInterval(pm.IntervalInMinutes, id, pc.Name)
	}

	return nil
}

func (client *Client) RefreshIdracSel(mc *Collector, ch chan<- prometheus.Metric) error {
	var resp IdracSelResponse

	err := client.redfishGet(redfishRootPath+"/Managers/iDRAC.Embedded.1/Logs/Sel", &resp)
	if err != nil {
		return err
	}

	for _, e := range resp.Members {
		st := string(e.SensorType)
		if st == "" {
			st = "Unknown"
		}
		ch <- mc.NewSelEntry(e.Id, e.Message, st, e.Severity, e.Created)
	}

	return nil
}

func (client *Client) RefreshStorage(mc *Collector, ch chan<- prometheus.Metric) error {
	var group GroupResponse
	var controller StorageController
	var d Drive

	err := client.redfishGet(client.storagePath, &group)
	if err != nil {
		return err
	}

	for _, c := range group.Members {
		err = client.redfishGet(c.OdataId, &controller)
		if err != nil {
			return err
		}

		for _, drive := range controller.Drives {
			err = client.redfishGet(drive.OdataId, &d)
			if err != nil {
				return err
			}

			ch <- mc.NewDriveInfo(d.Id, d.Name, d.Manufacturer, d.Model, d.SerialNumber, d.MediaType, d.Protocol, d.GetSlot())
			ch <- mc.NewDriveHealth(d.Id, d.Status.Health)
			ch <- mc.NewDriveCapacity(d.Id, d.CapacityBytes)
			ch <- mc.NewDriveLifeLeft(d.Id, d.PredictedLifeLeft)
		}
	}

	return nil
}

func (client *Client) RefreshMemory(mc *Collector, ch chan<- prometheus.Metric) error {
	var group GroupResponse
	var m Memory

	err := client.redfishGet(client.memoryPath, &group)
	if err != nil {
		return err
	}

	for _, c := range group.Members {
		err = client.redfishGet(c.OdataId, &m)
		if err != nil {
			return err
		}

		if m.Status.State == StateAbsent {
			continue
		}

		ch <- mc.NewMemoryModuleInfo(m.Id, m.Name, m.Manufacturer, m.MemoryDeviceType, m.SerialNumber, m.ErrorCorrection, m.RankCount)
		ch <- mc.NewMemoryModuleHealth(m.Id, m.Status.Health)
		ch <- mc.NewMemoryModuleCapacity(m.Id, m.CapacityMiB*1048576)
		ch <- mc.NewMemoryModuleSpeed(m.Id, m.OperatingSpeedMhz)
	}

	return nil
}

func (client *Client) redfishGet(path string, res interface{}) error {
	url := "https://" + client.hostname + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Basic "+client.basicAuth)
	req.Header.Add("Accept", "application/json")

	logging.Debugf("Querying url %q", url)

	resp, err := client.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		logging.Debugf("Failed to query url %q: %v", url, err)
		return err
	}

	if resp.StatusCode != 200 {
		logging.Debugf("Query to url %q returned unexpected status code: %d (%s)", url, resp.StatusCode, resp.Status)
		return fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(res)
	if err != nil {
		logging.Debugf("Error decoding response from url %q: %v", url, err)
		return err
	}

	return nil
}
