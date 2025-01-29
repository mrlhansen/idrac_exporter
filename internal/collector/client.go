package collector

import (
	"strconv"
	"strings"
	"time"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	UNKNOWN = iota
	DELL
	HPE
	LENOVO
	INSPUR
	H3C
	INVENTEC
)

type Client struct {
	redfish     *Redfish
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

func NewClient(hostConfig *config.HostConfig) *Client {
	client := &Client{
		redfish: NewRedfish(
			hostConfig.Hostname,
			hostConfig.Username,
			hostConfig.Password,
		),
	}

	client.redfish.CreateSession()
	ok := client.findAllEndpoints()
	if !ok {
		client.redfish.DeleteSession()
		return nil
	}

	return client
}

func (client *Client) findAllEndpoints() bool {
	var root V1Response
	var group GroupResponse
	var chassis ChassisResponse
	var system SystemResponse
	var ok bool

	// Root
	ok = client.redfish.Get(redfishRootPath, &root)
	if !ok {
		return false
	}

	// System
	ok = client.redfish.Get(root.Systems.OdataId, &group)
	if !ok {
		return false
	}

	client.systemPath = group.Members[0].OdataId

	// Chassis
	ok = client.redfish.Get(root.Chassis.OdataId, &group)
	if !ok {
		return false
	}

	// Thermal and Power
	ok = client.redfish.Get(group.Members[0].OdataId, &chassis)
	if !ok {
		return false
	}

	ok = client.redfish.Get(client.systemPath, &system)
	if !ok {
		return false
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
	} else if strings.Contains(m, "inventec") {
		client.vendor = INVENTEC
	}

	// Path for event log
	if config.Config.Collect.Events {
		switch client.vendor {
		case DELL:
			client.eventPath = "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Sel/Entries"
		case LENOVO:
			{
				if client.redfish.Exists("/redfish/v1/Systems/1/LogServices/PlatformLog/Entries") {
					client.eventPath = "/redfish/v1/Systems/1/LogServices/PlatformLog/Entries"
				} else if client.redfish.Exists("/redfish/v1/Systems/1/LogServices/StandardLog/Entries") {
					client.eventPath = "/redfish/v1/Systems/1/LogServices/StandardLog/Entries"
				}
			}
		case HPE:
			client.eventPath = "/redfish/v1/Systems/1/LogServices/IML/Entries"
		}
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
			client.eventPath = ""
			client.version = 4
		}
	}

	return true
}

func (client *Client) RefreshSensors(mc *Collector, ch chan<- prometheus.Metric) bool {
	resp := ThermalResponse{}
	ok := client.redfish.Get(client.thermalPath, &resp)
	if !ok {
		return false
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

	return true
}

func (client *Client) RefreshSystem(mc *Collector, ch chan<- prometheus.Metric) bool {
	resp := SystemResponse{}
	ok := client.redfish.Get(client.systemPath, &resp)
	if !ok {
		return false
	}

	// Need on iLO 6
	if resp.IndicatorLED == "" && client.vendor == HPE {
		resp.IndicatorLED = resp.Oem.Hpe.IndicatorLED
	}

	ch <- mc.NewSystemPowerOn(resp.PowerState)
	ch <- mc.NewSystemHealth(resp.Status.Health)
	ch <- mc.NewSystemIndicatorLED(resp.IndicatorLED)

	if resp.LocationIndicatorActive != nil {
		ch <- mc.NewSystemIndicatorActive(*resp.LocationIndicatorActive)
	}

	if resp.MemorySummary != nil {
		ch <- mc.NewSystemMemorySize(resp.MemorySummary.TotalSystemMemoryGiB * 1073741824)
	}

	if resp.ProcessorSummary != nil {
		ch <- mc.NewSystemCpuCount(resp.ProcessorSummary.Count, resp.ProcessorSummary.Model)
	}

	ch <- mc.NewSystemBiosInfo(resp.BiosVersion)
	ch <- mc.NewSystemMachineInfo(resp.Manufacturer, resp.Model, resp.SerialNumber, resp.SKU)

	return true
}

func (client *Client) RefreshNetwork(mc *Collector, ch chan<- prometheus.Metric) bool {
	group := GroupResponse{}
	ok := client.redfish.Get(client.networkPath, &group)
	if !ok {
		return false
	}

	for _, c := range group.Members.GetLinks() {
		ni := NetworkInterface{}
		ok = client.redfish.Get(c, &ni)
		if !ok {
			return false
		}

		if ni.Status.State != StateEnabled {
			continue
		}

		ch <- mc.NewNetworkInterfaceHealth(ni.Id, ni.Status.Health)

		ports := GroupResponse{}
		ok = client.redfish.Get(ni.GetPorts(), &ports)
		if !ok {
			return false
		}

		for _, c := range ports.Members.GetLinks() {
			port := NetworkPort{}
			ok = client.redfish.Get(c, &port)
			if !ok {
				return false
			}

			// Fix for issue #92
			if client.vendor == DELL {
				if ni.Id == port.Id {
					s := strings.Split(c, "/")
					port.Id = s[len(s)-1]
				}
			}

			ch <- mc.NewNetworkPortHealth(port.Id, ni.Id, port.Status.Health)
			ch <- mc.NewNetworkPortSpeed(port.Id, ni.Id, port.GetSpeed())
			ch <- mc.NewNetworkPortLinkUp(port.Id, ni.Id, port.LinkStatus)
		}
	}

	return true
}

func (client *Client) RefreshPower(mc *Collector, ch chan<- prometheus.Metric) bool {
	resp := PowerResponse{}
	ok := client.redfish.Get(client.powerPath, &resp)
	if !ok {
		return false
	}

	for i, psu := range resp.PowerSupplies {
		// Status is missing, but information is there
		if client.vendor == INVENTEC {
			psu.Status.State = StateEnabled
		}

		// iLO 4 (for issue #116)
		if (client.vendor == HPE) && (client.version == 4) {
			if psu.FirmwareVersion == "0.00" {
				continue
			}
		}

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

	return true
}

func (client *Client) RefreshEventLog(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.eventPath == "" {
		return true
	}

	resp := EventLogResponse{}
	ok := client.redfish.Get(client.eventPath, &resp)
	if !ok {
		return false
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

	return true
}

func (client *Client) RefreshStorage(mc *Collector, ch chan<- prometheus.Metric) bool {
	group := GroupResponse{}
	ok := client.redfish.Get(client.storagePath, &group)
	if !ok {
		return false
	}

	for _, c := range group.Members.GetLinks() {
		ctlr := StorageController{}
		ok = client.redfish.Get(c, &ctlr)
		if !ok {
			return false
		}

		// iLO 4
		if (client.vendor == HPE) && (client.version == 4) {
			grp := GroupResponse{}
			ok = client.redfish.Get(c+"DiskDrives/", &grp)
			if !ok {
				return false
			}
			ctlr.Drives = grp.Members
		}

		for _, c := range ctlr.Drives.GetLinks() {
			drive := Drive{}
			ok = client.redfish.Get(c, &drive)
			if !ok {
				return false
			}

			if drive.Status.State == StateAbsent {
				continue
			}

			// iLO 4
			if (client.vendor == HPE) && (client.version == 4) {
				drive.CapacityBytes = 1024 * 1024 * drive.CapacityMiB
				drive.Protocol = drive.InterfaceType
				drive.PredictedLifeLeft = 100.0 - drive.SSDEnduranceUtilizationPercentage
			}

			ch <- mc.NewDriveInfo(drive.Id, ctlr.Id, drive.Name, drive.Manufacturer, drive.Model, drive.SerialNumber, drive.MediaType, drive.Protocol, drive.GetSlot())
			ch <- mc.NewDriveHealth(drive.Id, ctlr.Id, drive.Status.Health)
			ch <- mc.NewDriveCapacity(drive.Id, ctlr.Id, drive.CapacityBytes)
			ch <- mc.NewDriveLifeLeft(drive.Id, ctlr.Id, drive.PredictedLifeLeft)
		}
	}

	return true
}

func (client *Client) RefreshMemory(mc *Collector, ch chan<- prometheus.Metric) bool {
	group := GroupResponse{}
	ok := client.redfish.Get(client.memoryPath, &group)
	if !ok {
		return false
	}

	for _, c := range group.Members.GetLinks() {
		m := Memory{}
		ok = client.redfish.Get(c, &m)
		if !ok {
			return false
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
		ch <- mc.NewMemoryModuleCapacity(m.Id, 1048576*m.CapacityMiB)
		ch <- mc.NewMemoryModuleSpeed(m.Id, m.OperatingSpeedMhz)
	}

	return true
}
