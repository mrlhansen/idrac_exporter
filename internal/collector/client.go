package collector

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/log"
	"github.com/prometheus/client_golang/prometheus"
)

const redfishRootPath = "/redfish/v1"

const (
	UNKNOWN = iota
	DELL
	HPE
	LENOVO
	INSPUR
	H3C
)

type Client struct {
	hostname    string
	username    string
	password    string
	httpClient  *http.Client
	vendor      int
	version     int
	systemPath  string
	thermalPath string
	powerPath   string
	storagePath string
	memoryPath  string
	networkPath string
	eventPath   string
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
		username:   hostConfig.Username,
		password:   hostConfig.Password,
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
	client.networkPath = system.NetworkInterfaces.OdataId
	client.thermalPath = chassis.Thermal.OdataId
	client.powerPath = chassis.Power.OdataId

	// Vendor
	m := strings.ToLower(system.Manufacturer)
	if strings.Contains(m, "dell") {
		client.vendor = DELL
	} else if strings.Contains(m, "hpe") {
		client.vendor = HPE
	} else if strings.Contains(m, "lenovo") {
		client.vendor = LENOVO
	} else if strings.Contains(m, "inspur") {
		client.vendor = INSPUR
	} else if strings.Contains(m, "h3c") {
		client.vendor = H3C
	}

	// Fix for Inspur bug
	if client.vendor == INSPUR {
		client.storagePath = strings.ReplaceAll(client.storagePath, "Storages", "Storage")
	}

	// Fix for iLO 4 machines
	if client.vendor == HPE {
		if strings.Contains(root.Name, "HP RESTful") {
			client.memoryPath = "/redfish/v1/Systems/1/Memory/"
			client.storagePath = "/redfish/v1/Systems/1/SmartStorage/ArrayControllers/"
			client.version = 4
		}
	}

	// Path for event log
	switch client.vendor {
	case DELL:
		client.eventPath = "/redfish/v1/Managers/iDRAC.Embedded.1/Logs/Sel"
	case LENOVO:
		client.eventPath = "/redfish/v1/Systems/1/LogServices/PlatformLog/Entries"
	case HPE:
		client.eventPath = "/redfish/v1/Systems/1/LogServices/IML/Entries"
	}

	return nil
}

func (client *Client) RefreshSensors(mc *Collector, ch chan<- prometheus.Metric) error {
	resp := ThermalResponse{}
	err := client.redfishGet(client.thermalPath, &resp)
	if err != nil {
		return err
	}

	for n, t := range resp.Temperatures {
		if t.Status.State != StateEnabled {
			continue
		}

		if t.ReadingCelsius < 0 {
			continue
		}

		id := t.GetId(n)
		ch <- mc.NewSensorsTemperature(t.ReadingCelsius, id, t.Name, "celsius")
	}

	for n, f := range resp.Fans {
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

		id := f.GetId(n)
		ch <- mc.NewSensorsFanHealth(id, name, f.Status.Health)
		ch <- mc.NewSensorsFanSpeed(f.GetReading(), id, name, strings.ToLower(units))
	}

	return nil
}

func (client *Client) RefreshSystem(mc *Collector, ch chan<- prometheus.Metric) error {
	resp := SystemResponse{}
	err := client.redfishGet(client.systemPath, &resp)
	if err != nil {
		return err
	}

	ch <- mc.NewSystemPowerOn(resp.PowerState)
	ch <- mc.NewSystemHealth(resp.Status.Health)
	ch <- mc.NewSystemIndicatorLED(resp.IndicatorLED)
	if resp.MemorySummary != nil {
		ch <- mc.NewSystemMemorySize(resp.MemorySummary.TotalSystemMemoryGiB * 1073741824)
	}
	if resp.ProcessorSummary != nil {
		ch <- mc.NewSystemCpuCount(resp.ProcessorSummary.Count, resp.ProcessorSummary.Model)
	}
	ch <- mc.NewSystemBiosInfo(resp.BiosVersion)
	ch <- mc.NewSystemMachineInfo(resp.Manufacturer, resp.Model, resp.SerialNumber, resp.SKU)

	return nil
}

func (client *Client) RefreshNetwork(mc *Collector, ch chan<- prometheus.Metric) error {
	group := GroupResponse{}
	err := client.redfishGet(client.networkPath, &group)
	if err != nil {
		return err
	}

	for _, c := range group.Members.GetLinks() {
		ni := NetworkInterface{}
		err = client.redfishGet(c, &ni)
		if err != nil {
			return err
		}

		if ni.Status.State != StateEnabled {
			continue
		}

		ch <- mc.NewNetworkInterfaceHealth(ni.Id, ni.Status.Health)

		ports := GroupResponse{}
		err = client.redfishGet(ni.GetPorts(), &ports)
		if err != nil {
			return err
		}

		for _, c := range ports.Members.GetLinks() {
			port := NetworkPort{}
			err = client.redfishGet(c, &port)
			if err != nil {
				return err
			}

			ch <- mc.NewNetworkPortHealth(ni.Id, port.Id, port.Status.Health)
			ch <- mc.NewNetworkPortSpeed(ni.Id, port.Id, port.GetSpeed())
			ch <- mc.NewNetworkPortLinkUp(ni.Id, port.Id, port.LinkStatus)
		}
	}

	return nil
}

func (client *Client) RefreshPower(mc *Collector, ch chan<- prometheus.Metric) error {
	resp := PowerResponse{}
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

func (client *Client) RefreshEventLog(mc *Collector, ch chan<- prometheus.Metric) error {
	if client.eventPath == "" {
		return nil
	}

	resp := EventLogResponse{}
	err := client.redfishGet(client.eventPath, &resp)
	if err != nil {
		return err
	}

	level := config.Config.Event.SeverityLevel
	maxage := config.Config.Event.MaxAgeSeconds

	for _, e := range resp.Members {
		t, err := time.Parse(time.RFC3339, e.Created)
		if err != nil {
			continue
		}

		d := time.Since(t)
		if d.Seconds() > maxage {
			continue
		}

		severity := health2value(e.Severity)
		if severity < level {
			continue
		}

		ch <- mc.NewEventLogEntry(e.Id, e.Message, e.Severity, t)
	}

	return nil
}

func (client *Client) RefreshStorage(mc *Collector, ch chan<- prometheus.Metric) error {
	group := GroupResponse{}
	err := client.redfishGet(client.storagePath, &group)
	if err != nil {
		return err
	}

	for _, c := range group.Members.GetLinks() {
		ctlr := StorageController{}
		err = client.redfishGet(c, &ctlr)
		if err != nil {
			return err
		}

		// iLO 4
		if (client.vendor == HPE) && (client.version == 4) {
			grp := GroupResponse{}
			err = client.redfishGet(c+"DiskDrives/", &grp)
			if err != nil {
				return err
			}
			ctlr.Drives = grp.Members
		}

		for _, c := range ctlr.Drives.GetLinks() {
			drive := Drive{}
			err = client.redfishGet(c, &drive)
			if err != nil {
				return err
			}

			// iLO 4
			if (client.vendor == HPE) && (client.version == 4) {
				drive.CapacityBytes = 1024 * 1024 * drive.CapacityMiB
				drive.Protocol = drive.InterfaceType
				drive.PredictedLifeLeft = 100 - drive.SSDEnduranceUtilizationPercentage
			}

			ch <- mc.NewDriveInfo(drive.Id, drive.Name, drive.Manufacturer, drive.Model, drive.SerialNumber, drive.MediaType, drive.Protocol, drive.GetSlot())
			ch <- mc.NewDriveHealth(drive.Id, drive.Status.Health)
			ch <- mc.NewDriveCapacity(drive.Id, drive.CapacityBytes)
			ch <- mc.NewDriveLifeLeft(drive.Id, drive.PredictedLifeLeft)
		}
	}

	return nil
}

func (client *Client) RefreshMemory(mc *Collector, ch chan<- prometheus.Metric) error {
	group := GroupResponse{}
	err := client.redfishGet(client.memoryPath, &group)
	if err != nil {
		return err
	}

	for _, c := range group.Members.GetLinks() {
		m := Memory{}
		err = client.redfishGet(c, &m)
		if err != nil {
			return err
		}

		if (m.Status.State == StateAbsent) || (m.Id == "") {
			continue
		}

		// iLO 4
		if (client.vendor == HPE) && (client.version == 4) {
			m.Manufacturer = strings.TrimSpace(m.Manufacturer)
			m.RankCount = m.Rank
			m.MemoryDeviceType = m.DIMMType
			m.Status.Health = m.DIMMStatus
			m.CapacityMiB = m.SizeMB
		}

		ch <- mc.NewMemoryModuleInfo(m.Id, m.Name, m.Manufacturer, m.MemoryDeviceType, m.SerialNumber, m.ErrorCorrection, m.RankCount)
		ch <- mc.NewMemoryModuleHealth(m.Id, m.Status.Health)
		ch <- mc.NewMemoryModuleCapacity(m.Id, m.CapacityMiB*1048576)
		ch <- mc.NewMemoryModuleSpeed(m.Id, m.OperatingSpeedMhz)
	}

	return nil
}

func (client *Client) redfishGet(path string, res interface{}) error {
	if !strings.HasPrefix(path, redfishRootPath) {
		return fmt.Errorf("invalid url for redfish request")
	}

	url := "https://" + client.hostname + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.SetBasicAuth(client.username, client.password)

	log.Debug("Querying url %q", url)

	resp, err := client.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Debug("Failed to query url %q: %v", url, err)
		return err
	}

	if resp.StatusCode != 200 {
		log.Debug("Query to url %q returned unexpected status code: %d (%s)", url, resp.StatusCode, resp.Status)
		return fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(res)
	if err != nil {
		log.Debug("Error decoding response from url %q: %v", url, err)
		return err
	}

	return nil
}
