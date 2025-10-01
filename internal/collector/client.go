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
	SUPERMICRO
)

type Client struct {
	redfish *Redfish
	vendor  int
	version int
	path    struct {
		System           string
		Thermal          string
		ThermalSubsystem string
		Power            string
		PowerSubsystem   string
		Storage          string
		Memory           string
		Network          string
		Event            string
		Processors       string
		DellOEM          string
	}
}

func NewClient(h *config.HostConfig) *Client {
	client := &Client{
		redfish: NewRedfish(
			h.Scheme,
			h.Hostname,
			h.Username,
			h.Password,
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

	client.path.System = group.Members[0].OdataId

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

	ok = client.redfish.Get(client.path.System, &system)
	if !ok {
		return false
	}

	client.path.Storage = system.Storage.OdataId
	client.path.Memory = system.Memory.OdataId
	client.path.Network = chassis.NetworkAdapters.OdataId
	client.path.Thermal = chassis.Thermal.OdataId
	client.path.ThermalSubsystem = chassis.ThermalSubsystem.OdataId
	client.path.Power = chassis.Power.OdataId
	client.path.PowerSubsystem = chassis.PowerSubsystem.OdataId
	client.path.Processors = system.Processors.OdataId

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
	} else if strings.Contains(m, "supermicro") {
		client.vendor = SUPERMICRO
	}

	// Path for event log
	if config.Config.Collect.Events {
		switch client.vendor {
		case DELL:
			{
				pathA := "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Sel/Entries"
				pathB := "/redfish/v1/Managers/iDRAC.Embedded.1/Logs/Sel"
				if client.redfish.Exists(pathA) {
					client.path.Event = pathA
				} else if client.redfish.Exists(pathB) {
					client.path.Event = pathB
				}
			}
		case LENOVO:
			{
				pathA := "/redfish/v1/Systems/1/LogServices/PlatformLog/Entries"
				pathB := "/redfish/v1/Systems/1/LogServices/StandardLog/Entries"
				if client.redfish.Exists(pathA) {
					client.path.Event = pathA
				} else if client.redfish.Exists(pathB) {
					client.path.Event = pathB
				}
			}
		case HPE:
			client.path.Event = "/redfish/v1/Systems/1/LogServices/IML/Entries"
		case FUJITSU:
			client.path.Event = "/redfish/v1/Managers/iRMC/LogServices/SystemEventLog/Entries"
		case SUPERMICRO:
			client.path.Event = "/redfish/v1/Systems/1/LogServices/Log1/Entries"
		}
	}

	// Dell OEM
	if config.Config.Collect.Extra {
		if client.vendor == DELL {
			if client.redfish.Exists(DellSystemPath) {
				client.path.DellOEM = DellSystemPath
			}
		}
	}

	// Issue #50
	if client.vendor == INSPUR {
		client.path.Storage = strings.ReplaceAll(client.path.Storage, "Storages", "Storage")
	}

	// Fix for iLO 4 machines
	if client.vendor == HPE {
		if strings.Contains(root.Name, "HP RESTful") {
			client.path.Memory = "/redfish/v1/Systems/1/Memory/"
			client.path.Storage = "/redfish/v1/Systems/1/SmartStorage/ArrayControllers/"
			client.path.Event = ""
			client.version = 4
		}
	}

	return true
}

func (client *Client) RefreshSensorsNew(mc *Collector, ch chan<- prometheus.Metric) bool {
	thermal := ThermalSubsystem{}
	ok := client.redfish.Get(client.path.ThermalSubsystem, &thermal)
	if !ok {
		return false
	}

	if thermal.Fans.OdataId != "" {
		group := GroupResponse{}
		ok := client.redfish.Get(thermal.Fans.OdataId, &group)
		if !ok {
			return false
		}

		for _, c := range group.Members.GetLinks() {
			fan := ThermalFan{}
			ok = client.redfish.Get(c, &fan)
			if !ok {
				return false
			}

			units := "percent"
			value := 0.0

			if fan.SpeedPercent.SpeedRPM != nil {
				units = "rpm"
				value = *fan.SpeedPercent.SpeedRPM
			} else if fan.SpeedPercent.Reading != nil {
				value = *fan.SpeedPercent.Reading
			} else {
				continue
			}

			mc.NewSensorsFanHealth(ch, fan.Id, fan.Name, fan.Status.Health)
			mc.NewSensorsFanSpeed(ch, value, fan.Id, fan.Name, strings.ToLower(units))
		}

	}

	if thermal.ThermalMetrics.OdataId != "" {
		temp := ThermalMetrics{}
		ok := client.redfish.Get(thermal.ThermalMetrics.OdataId, &temp)
		if !ok {
			return false
		}

		for n, c := range temp.TemperatureReadingsCelsius {
			if c.Reading != nil {
				name := ""
				if client.vendor == DELL {
					c.DeviceName = ""
				}

				if c.DeviceName != "" {
					name = c.DeviceName
				} else if c.DataSourceUri != "" {
					s := strings.Split(c.DataSourceUri, "/")
					name = s[len(s)-1]
				}

				mc.NewSensorsTemperature(ch, *c.Reading, strconv.Itoa(n), name, "celsius")
			}
		}
	}

	return true
}

func (client *Client) RefreshSensorsOld(mc *Collector, ch chan<- prometheus.Metric) bool {
	resp := ThermalResponse{}
	ok := client.redfish.Get(client.path.Thermal, &resp)
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

func (client *Client) RefreshSensors(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.path.Thermal != "" {
		return client.RefreshSensorsOld(mc, ch)
	}
	if client.path.ThermalSubsystem != "" {
		return client.RefreshSensorsNew(mc, ch)
	}
	return true
}

func (client *Client) RefreshSystem(mc *Collector, ch chan<- prometheus.Metric) bool {
	resp := SystemResponse{}
	ok := client.redfish.Get(client.path.System, &resp)
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

func (client *Client) RefreshProcessors(mc *Collector, ch chan<- prometheus.Metric) bool {
	group := GroupResponse{}
	ok := client.redfish.Get(client.path.Processors, &group)
	if !ok {
		return false
	}

	for _, c := range group.Members.GetLinks() {
		resp := Processor{}
		ok = client.redfish.Get(c, &resp)
		if !ok {
			return false
		}

		if resp.ProcessorType != "CPU" {
			continue
		}

		if resp.Status.State != StateEnabled {
			continue
		}

		mc.NewCpuInfo(ch, &resp)
		mc.NewCpuHealth(ch, &resp)
		mc.NewCpuVoltage(ch, &resp)
		mc.NewCpuMaxSpeed(ch, &resp)
		mc.NewCpuCurrentSpeed(ch, &resp)
		mc.NewCpuTotalCores(ch, &resp)
		mc.NewCpuTotalThreads(ch, &resp)
	}

	return true
}

func (client *Client) RefreshNetwork(mc *Collector, ch chan<- prometheus.Metric) bool {
	group := GroupResponse{}
	ok := client.redfish.Get(client.path.Network, &group)
	if !ok {
		return false
	}

	for _, c := range group.Members.GetLinks() {
		ni := NetworkAdapter{}
		ok = client.redfish.Get(c, &ni)
		if !ok {
			return false
		}

		mc.NewNetworkAdapterInfo(ch, &ni)
		mc.NewNetworkAdapterHealth(ch, &ni)

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
			mc.NewNetworkPortCurrentSpeed(ch, ni.Id, &port)
			mc.NewNetworkPortMaxSpeed(ch, ni.Id, &port)
			mc.NewNetworkPortLinkUp(ch, ni.Id, &port)
		}
	}

	return true
}

func (client *Client) RefreshPowerNew(mc *Collector, ch chan<- prometheus.Metric) bool {
	power := PowerSubsystem{}
	ok := client.redfish.Get(client.path.PowerSubsystem, &power)
	if !ok {
		return false
	}

	// global CapacityWatts?

	if power.PowerSupplies.OdataId == "" {
		return true
	}

	group := GroupResponse{}
	ok = client.redfish.Get(power.PowerSupplies.OdataId, &group)
	if !ok {
		return false
	}

	for _, c := range group.Members.GetLinks() {
		psu := PowerSupply{}
		ok = client.redfish.Get(c, &psu)
		if !ok {
			return false
		}

		mc.NewPowerSupplyHealth(ch, psu.Status.Health, psu.Id)
		mc.NewPowerSupplyCapacityWatts(ch, psu.PowerCapacityWatts, psu.Id)

		if psu.Metrics.OdataId != "" {
			m := PowerSupplyMetrics{}
			ok = client.redfish.Get(psu.Metrics.OdataId, &m)
			if !ok {
				return false
			}

			if m.InputVoltage != nil {
				mc.NewPowerSupplyInputVoltage(ch, m.InputVoltage.Reading, psu.Id)
			}

			if m.InputPowerWatts != nil {
				mc.NewPowerSupplyInputWatts(ch, m.InputPowerWatts.Reading, psu.Id)
			}

			if m.OutputPowerWatts != nil {
				mc.NewPowerSupplyOutputWatts(ch, m.OutputPowerWatts.Reading, psu.Id)
			}
		}
	}

	return true
}

func (client *Client) RefreshPowerOld(mc *Collector, ch chan<- prometheus.Metric) bool {
	resp := PowerResponse{}
	ok := client.redfish.Get(client.path.Power, &resp)
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

func (client *Client) RefreshPower(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.path.Power != "" {
		return client.RefreshPowerOld(mc, ch)
	}
	if client.path.PowerSubsystem != "" {
		return client.RefreshPowerNew(mc, ch)
	}
	return true
}

func (client *Client) RefreshEventLog(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.path.Event == "" {
		return true
	}

	resp := EventLogResponse{}
	ok := client.redfish.Get(client.path.Event, &resp)
	if !ok {
		return false
	}

	// iDRAC 8 (issue #143)
	if client.vendor == DELL {
		if resp.Id == "SEL" && strings.Contains(client.path.Event, "LogServices") {
			client.path.Event = "/redfish/v1/Managers/iDRAC.Embedded.1/Logs/Sel"
			return client.RefreshEventLog(mc, ch)
		}
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
	ok := client.redfish.Get(client.path.Storage, &group)
	if !ok {
		return false
	}

	for _, c := range group.Members.GetLinks() {
		storage := Storage{}
		ok = client.redfish.Get(c, &storage)
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
			storage.Drives = grp.Members
		}

		mc.NewStorageInfo(ch, &storage)
		mc.NewStorageHealth(ch, &storage)
		mc.NewDellControllerBatteryHealth(ch, &storage)

		// Drives
		for _, c := range storage.Drives.GetLinks() {
			drive := StorageDrive{}
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

			mc.NewStorageDriveInfo(ch, storage.Id, &drive)
			mc.NewStorageDriveHealth(ch, storage.Id, &drive)
			mc.NewStorageDriveCapacity(ch, storage.Id, &drive)
			mc.NewStorageDriveLifeLeft(ch, storage.Id, &drive)
			mc.NewStorageDriveIndicatorActive(ch, storage.Id, &drive)
		}

		// iLO 4
		if (client.vendor == HPE) && (client.version == 4) {
			continue
		}

		// Controllers
		if c := storage.Controllers.OdataId; len(c) > 0 {
			grp := GroupResponse{}
			ok = client.redfish.Get(c, &grp)
			if !ok {
				return false
			}

			for _, c := range grp.Members.GetLinks() {
				ctlr := StorageController{}
				ok = client.redfish.Get(c, &ctlr)
				if !ok {
					return false
				}

				mc.NewStorageControllerInfo(ch, storage.Id, &ctlr)
				mc.NewStorageControllerSpeed(ch, storage.Id, &ctlr)
				mc.NewStorageControllerHealth(ch, storage.Id, &ctlr)
			}
		}

		// Volumes
		if c := storage.Volumes.OdataId; len(c) > 0 {
			grp := GroupResponse{}
			ok = client.redfish.Get(c, &grp)
			if !ok {
				return false
			}

			for _, c := range grp.Members.GetLinks() {
				vol := StorageVolume{}
				ok = client.redfish.Get(c, &vol)
				if !ok {
					return false
				}

				mc.NewStorageVolumeInfo(ch, storage.Id, &vol)
				mc.NewStorageVolumeHealth(ch, storage.Id, &vol)
				mc.NewStorageVolumeCapacity(ch, storage.Id, &vol)
				mc.NewStorageVolumeMediaSpan(ch, storage.Id, &vol)
			}
		}
	}

	return true
}

func (client *Client) RefreshMemory(mc *Collector, ch chan<- prometheus.Metric) bool {
	group := GroupResponse{}
	ok := client.redfish.Get(client.path.Memory, &group)
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
	if client.path.DellOEM == "" {
		return true
	}

	resp := DellSystem{}
	ok := client.redfish.Get(client.path.DellOEM, &resp)
	if !ok {
		return false
	}

	mc.NewDellBatteryRollupHealth(ch, &resp)
	mc.NewDellEstimatedSystemAirflowCFM(ch, &resp)

	return true
}
