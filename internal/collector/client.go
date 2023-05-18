package collector

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	// "strconv"
	// "strings"
	"time"
	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/logging"
	"github.com/prometheus/client_golang/prometheus"
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

	err := client.findAllEndpoints();
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
	err = client.redfishGet(redfishRootPath, &root);
	if err != nil {
		return err
	}

	// System
	err = client.redfishGet(root.Systems.OdataId, &group);
	if err != nil {
		return err
	}

	client.systemPath = group.Members[0].OdataId

	// Chassis
	err = client.redfishGet(root.Chassis.OdataId, &group);
	if err != nil {
		return err
	}

	// Thermal and Power
	err = client.redfishGet(group.Members[0].OdataId, &chassis);
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

func (client *Client) RefreshSensors(c *myCollector, ch chan<- prometheus.Metric) error {
	var resp ThermalResponse

	err := client.redfishGet(client.thermalPath, &resp);
	if err != nil {
		return err
	}

	for _, t := range resp.Temperatures {
		if t.Status.State != StateEnabled {
			continue
		}
		// store.SetTemperature(t.ReadingCelsius, t.Name, "celsius")
		ch <- prometheus.MustNewConstMetric(
			c.SensorsTemperature,
			prometheus.GaugeValue,
			t.ReadingCelsius,
			t.Name,
			"celsius",
		)
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

		// store.SetFanSpeed(f.GetReading(), name, strings.ToLower(units))
	}

	return nil
}

// func (client *Client) RefreshSystem(store metricsStore) error {
// 	var resp SystemResponse
//
// 	err := client.redfishGet(client.systemPath, &resp);
// 	if err != nil {
// 		return err
// 	}
//
// 	store.SetPowerOn(resp.PowerState)
// 	store.SetHealth(resp.Status.Health)
// 	store.SetLedOn(resp.IndicatorLED)
// 	store.SetMemorySize(resp.MemorySummary.TotalSystemMemoryGiB * 1073741824)
// 	store.SetCpuCount(resp.ProcessorSummary.Count, resp.ProcessorSummary.Model)
// 	store.SetBiosInfo(resp.BiosVersion)
// 	store.SetMachineInfo(resp.Manufacturer, resp.Model, resp.SerialNumber, resp.SKU)
//
// 	return nil
// }
//
// func (client *Client) RefreshPower(store metricsStore) error {
// 	var resp PowerResponse
//
// 	err := client.redfishGet(client.powerPath, &resp);
// 	if err != nil {
// 		return err
// 	}
//
// 	for i, psu := range resp.PowerSupplies {
// 		if psu.Status.State != StateEnabled {
// 			continue
// 		}
//
// 		id := strconv.Itoa(i)
// 		store.SetPowerSupplyInputWatts(psu.PowerInputWatts, id)
// 		store.SetPowerSupplyInputVoltage(psu.LineInputVoltage, id)
// 		store.SetPowerSupplyOutputWatts(psu.GetOutputPower(), id)
// 		store.SetPowerSupplyCapacityWatts(psu.PowerCapacityWatts, id)
// 		store.SetPowerSupplyEfficiencyPercent(psu.EfficiencyPercent, id)
// 	}
//
// 	for i, pc := range resp.PowerControl {
// 		id := strconv.Itoa(i)
// 		store.SetPowerControlConsumedWatts(pc.PowerConsumedWatts, id, pc.Name)
// 		store.SetPowerControlCapacityWatts(pc.PowerCapacityWatts, id, pc.Name)
//
// 		if pc.PowerMetrics == nil {
// 			continue
// 		}
//
// 		pm := pc.PowerMetrics
// 		store.SetPowerControlMinConsumedWatts(pm.MinConsumedWatts, id, pc.Name)
// 		store.SetPowerControlMaxConsumedWatts(pm.MaxConsumedWatts, id, pc.Name)
// 		store.SetPowerControlAvgConsumedWatts(pm.AverageConsumedWatts, id, pc.Name)
// 		store.SetPowerControlInterval(pm.IntervalInMinutes, id, pc.Name)
// 	}
//
// 	return nil
// }
//
// func (client *Client) RefreshIdracSel(store metricsStore) error {
// 	var resp IdracSelResponse
//
// 	err := client.redfishGet(redfishRootPath + "/Managers/iDRAC.Embedded.1/Logs/Sel", &resp);
// 	if err != nil {
// 		return err
// 	}
//
// 	for _, e := range resp.Members {
// 		st := string(e.SensorType)
// 		if st == "" {
// 			st = "Unknown"
// 		}
// 		store.SetSelEntry(e.Id, e.Message, st, e.Severity, e.Created)
// 	}
//
// 	return nil
// }
//
// func (client *Client) RefreshStorage(store metricsStore) error {
// 	var group GroupResponse
// 	var controller StorageController
// 	var d Drive
//
// 	err := client.redfishGet(client.storagePath, &group)
// 	if err != nil {
// 		return err
// 	}
//
// 	for _, c := range group.Members {
// 		err = client.redfishGet(c.OdataId, &controller)
// 		if err != nil {
// 			return err
// 		}
//
// 		for _, drive := range controller.Drives {
// 			err = client.redfishGet(drive.OdataId, &d)
// 			if err != nil {
// 				return err
// 			}
// 			store.SetDriveInfo(d.Id, d.Name, d.Manufacturer, d.Model, d.SerialNumber, d.MediaType, d.Protocol, d.GetSlot())
// 			store.SetDriveHealth(d.Id, d.Status.Health)
// 			store.SetDriveCapacity(d.Id, d.CapacityBytes)
// 		}
// 	}
//
// 	return nil
// }
//
// func (client *Client) RefreshMemory(store metricsStore) error {
// 	var group GroupResponse
// 	var m Memory
//
// 	err := client.redfishGet(client.memoryPath, &group)
// 	if err != nil {
// 		return err
// 	}
//
// 	for _, c := range group.Members {
// 		err = client.redfishGet(c.OdataId, &m)
// 		if err != nil {
// 			return err
// 		}
//
// 		if m.Status.State == "Absent" {
// 			continue
// 		}
//
// 		store.SetMemoryInfo(m.Id, m.Name, m.Manufacturer, m.MemoryDeviceType, m.SerialNumber, m.ErrorCorrection, m.RankCount)
// 		store.SetMemoryHealth(m.Id, m.Status.Health)
// 		store.SetMemoryCapacity(m.Id, m.CapacityMiB * 1048576)
// 		store.SetMemorySpeed(m.Id, m.OperatingSpeedMhz)
// 	}
//
// 	return nil
// }

func (client *Client) redfishGet(path string, res interface{}) error {
	url := "https://" + client.hostname + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Basic " + client.basicAuth)
	req.Header.Add("Accept", "application/json")

	logging.Debugf("Querying url %q", url)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		logging.Debugf("Failed to query url %q: %v", url, err)
		return err
	}

	if resp.StatusCode != 200 {
		logging.Debugf("Query to url %q returned unexpected status code: %d (%s)", url, resp.StatusCode, resp.Status)
		return fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(res);
	if err != nil {
		logging.Debugf("Error decoding response from url %q: %v", url, err)
		return err
	}

	return nil
}
