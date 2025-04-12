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
	FUJITSU
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
	} else if strings.Contains(m, "fujitsu") {
		client.vendor = FUJITSU
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
		case FUJITSU:
			client.eventPath = "/redfish/v1/Managers/iRMC/LogServices/SystemEventLog/Entries"
		}
	}

	// Issue #50
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
		mc.NewSensorsTemperature(ch, t.ReadingCelsius, id, t.Name, "celsius")
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
		mc.NewSensorsFanHealth(ch, id, name, f.Status.Health)
		mc.NewSensorsFanSpeed(ch, f.GetReading(), id, name, strings.ToLower(units))
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
	if client.vendor == HPE && resp.IndicatorLED == "" {
		resp.IndicatorLED = resp.Oem.Hpe.IndicatorLED
	}

	mc.NewSystemPowerOn(ch, &resp)
	mc.NewSystemHealth(ch, &resp)
	mc.NewSystemIndicatorLED(ch, &resp)
	mc.NewSystemIndicatorActive(ch, &resp)
	mc.NewSystemMemorySize(ch, &resp)
	mc.NewSystemCpuCount(ch, &resp)
	mc.NewSystemBiosInfo(ch, &resp)
	mc.NewSystemMachineInfo(ch, &resp)

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

		mc.NewNetworkInterfaceHealth(ch, &ni)

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

			// Issue #92
			if client.vendor == DELL {
				if ni.Id == port.Id {
					s := strings.Split(c, "/")
					port.Id = s[len(s)-1]
				}
			}

			mc.NewNetworkPortHealth(ch, ni.Id, &port)
			mc.NewNetworkPortSpeed(ch, ni.Id, &port)
			mc.NewNetworkPortLinkUp(ch, ni.Id, &port)
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

	// Issue #121
	if (client.vendor == FUJITSU) && (resp.Oem.TsFujitsu != nil) {
		for n, p := range resp.PowerSupplies {
			if len(p.Name) == 0 {
				continue
			}
			for _, v := range resp.Oem.TsFujitsu.ChassisPowerSensors {
				if (v.EntityID == "Power Supply") && strings.HasPrefix(v.Designation, p.Name) {
					resp.PowerSupplies[n].PowerInputWatts = v.CurrentPowerConsumptionW
				}
			}
		}
		if cp := resp.Oem.TsFujitsu.ChassisPowerConsumption; cp != nil {
			if len(resp.PowerControl) > 0 {
				pc := &resp.PowerControl[0]
				if cp.CurrentPowerConsumptionW > 0 {
					pc.PowerConsumedWatts = cp.CurrentPowerConsumptionW
				}
				if cp.CurrentMaximumPowerW > 0 {
					pc.PowerCapacityWatts = cp.CurrentMaximumPowerW
				}
				if pc.PowerMetrics == nil {
					pc.PowerMetrics = &PowerMetrics{
						AvgConsumedWatts: cp.AveragePowerW,
						MaxConsumedWatts: cp.PeakPowerW,
						MinConsumedWatts: cp.MinimumPowerW,
					}
				}
			}
		}
	}

	for i, psu := range resp.PowerSupplies {
		// Status is missing, but information is there
		if client.vendor == INVENTEC {
			psu.Status.State = StateEnabled
		}

		// Issue #116
		if (client.vendor == HPE) && (client.version == 4) {
			if psu.FirmwareVersion == "0.00" {
				continue
			}
		}

		if psu.Status.State != StateEnabled {
			continue
		}

		id := strconv.Itoa(i)
		mc.NewPowerSupplyHealth(ch, psu.Status.Health, id)
		mc.NewPowerSupplyInputWatts(ch, psu.PowerInputWatts, id)
		mc.NewPowerSupplyInputVoltage(ch, psu.LineInputVoltage, id)
		mc.NewPowerSupplyOutputWatts(ch, psu.GetOutputPower(), id)
		mc.NewPowerSupplyCapacityWatts(ch, psu.PowerCapacityWatts, id)
		mc.NewPowerSupplyEfficiencyPercent(ch, psu.EfficiencyPercent, id)
	}

	for i, pc := range resp.PowerControl {
		id := strconv.Itoa(i)
		mc.NewPowerControlConsumedWatts(ch, pc.PowerConsumedWatts, id, pc.Name)
		mc.NewPowerControlCapacityWatts(ch, pc.PowerCapacityWatts, id, pc.Name)

		if pc.PowerMetrics == nil {
			continue
		}

		pm := pc.PowerMetrics
		mc.NewPowerControlMinConsumedWatts(ch, pm.MinConsumedWatts, id, pc.Name)
		mc.NewPowerControlMaxConsumedWatts(ch, pm.MaxConsumedWatts, id, pc.Name)
		mc.NewPowerControlAvgConsumedWatts(ch, pm.AvgConsumedWatts, id, pc.Name)
		mc.NewPowerControlInterval(ch, pm.IntervalInMinutes, id, pc.Name)
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

		mc.NewEventLogEntry(ch, e.Id, e.Message, e.Severity, t)
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

			mc.NewDriveInfo(ch, ctlr.Id, &drive)
			mc.NewDriveHealth(ch, ctlr.Id, &drive)
			mc.NewDriveCapacity(ch, ctlr.Id, &drive)
			mc.NewDriveLifeLeft(ch, ctlr.Id, &drive)
			mc.NewDriveIndicatorActive(ch, ctlr.Id, &drive)
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

		mc.NewMemoryModuleInfo(ch, &m)
		mc.NewMemoryModuleHealth(ch, &m)
		mc.NewMemoryModuleCapacity(ch, &m)
		mc.NewMemoryModuleSpeed(ch, &m)
	}

	return true
}

func (client *Client) RefreshDell(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.vendor != DELL {
		return true
	}

	resp := DellSystem{}
	ok := client.redfish.Get("/redfish/v1/Systems/System.Embedded.1/Oem/Dell/DellSystem/System.Embedded.1", &resp)
	if !ok {
		return false
	}

	mc.NewDellBatteryRollupHealth(ch, &resp)
	mc.NewDellEstimatedSystemAirflowCFM(ch, &resp)

	return true
}
