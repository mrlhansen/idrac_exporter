package collector

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

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
	ADVANTECH
	HUAWEI
)

type Client struct {
	redfish *Redfish
	metrics *config.MetricsConfig
	vendor  int
	version int
	path    struct {
		System           string
		Thermal          string
		ThermalSubsystem string
		Power            string
		PowerSubsystem   string
		Sensors          string
		Storage          string
		SimpleStorage    string
		Memory           string
		Network          string
		Event            string
		Processors       string
		Manager          string
		ManagerEvent     string
		Extra            []string
		RackPDUs         []string
	}
	features struct {
		ExpandLevels bool
		Filter       bool
	}
}

func NewClient(host string, config *config.AuthConfig) *Client {
	client := &Client{
		redfish: NewRedfish(host, config),
		metrics: &config.Metrics,
	}

	client.redfish.CreateSession()
	ok := client.findAllEndpoints()
	if !ok {
		client.redfish.DeleteSession()
		return nil
	}

	return client
}

func findPrimaryChassis(g *GroupResponse) string {
	switch len(g.Members) {
	case 0:
		return ""
	case 1:
		return g.Members[0].OdataId
	}

	preferred := []string{
		"System.Embedded.1",
		"Self",
		"1",
	}

	for _, m := range g.Members {
		s := path.Base(m.OdataId)
		if slices.Contains(preferred, s) {
			return m.OdataId
		}
	}

	return g.Members[0].OdataId
}

func (client *Client) findAllEndpoints() bool {
	var root V1Response
	var group GroupResponse
	var chassis ChassisResponse
	var system SystemResponse
	var path string
	var ok bool

	// Root
	ok = client.redfish.Get(redfishRootPath, &root)
	if !ok {
		return false
	}

	// PDUs
	if root.PowerDistribution != nil {
		path = root.PowerDistribution.OdataId
	}

	if root.PowerEquipment != nil {
		resp := PowerEquipment{}
		ok = client.redfish.Get(root.PowerEquipment.OdataId, &resp)
		if !ok {
			return false
		}
		path = resp.RackPDUs.OdataId
	}

	if len(path) > 0 {
		ok = client.redfish.Get(path, &group)
		if !ok {
			return false
		}
		client.path.RackPDUs = group.Members.GetLinks()
		return true
	}

	// System
	ok = client.redfish.Get(root.Systems.OdataId, &group)
	if !ok {
		return false
	}

	client.path.System = group.Members[0].OdataId

	ok = client.redfish.Get(client.path.System, &system)
	if !ok {
		return false
	}

	// Chassis
	ok = client.redfish.Get(root.Chassis.OdataId, &group)
	if !ok {
		return false
	}

	path = findPrimaryChassis(&group)

	ok = client.redfish.Get(path, &chassis)
	if !ok {
		return false
	}

	// Paths
	client.path.Storage = system.Storage.OdataId
	client.path.SimpleStorage = system.SimpleStorage.OdataId
	client.path.Memory = system.Memory.OdataId
	client.path.Network = chassis.NetworkAdapters.OdataId
	client.path.Thermal = chassis.Thermal.OdataId
	client.path.ThermalSubsystem = chassis.ThermalSubsystem.OdataId
	client.path.Power = chassis.Power.OdataId
	client.path.PowerSubsystem = chassis.PowerSubsystem.OdataId
	client.path.Processors = system.Processors.OdataId
	client.path.Sensors = chassis.Sensors.OdataId

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
	} else if strings.Contains(m, "ieit") {
		client.vendor = INSPUR
	} else if strings.Contains(m, "aivres") {
		client.vendor = INSPUR
	} else if strings.Contains(m, "h3c") {
		client.vendor = H3C
	} else if strings.Contains(m, "inventec") {
		client.vendor = INVENTEC
	} else if strings.Contains(m, "fujitsu") {
		client.vendor = FUJITSU
	} else if strings.Contains(m, "fsas") {
		client.vendor = FUJITSU
	} else if strings.Contains(m, "supermicro") {
		client.vendor = SUPERMICRO
	} else if strings.Contains(m, "advantech") {
		client.vendor = ADVANTECH
	} else if strings.Contains(m, "huawei") {
		client.vendor = HUAWEI
	} else if strings.Contains(m, "kunlun") {
		client.vendor = HUAWEI
	} else if system.Oem.Dell != nil {
		// Issue #175
		client.vendor = DELL
	}

	// OEM systems (e.g. WEKApod, Cloudian HSA) report the integrator as
	// Manufacturer while the service root still names the BMC vendor.
	if client.vendor == UNKNOWN {
		v := strings.ToLower(root.Vendor)
		if strings.Contains(v, "supermicro") {
			client.vendor = SUPERMICRO
		}
	}

	// Manager
	if client.metrics.Manager.Enabled {
		ok = client.redfish.Get(root.Managers.OdataId, &group)
		if ok && len(group.Members) > 0 {
			client.path.Manager = group.Members[0].OdataId
		}
	}

	// Manager event log
	switch client.vendor {
	case DELL:
		client.path.ManagerEvent = "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Lclog/Entries"
	case HPE:
		client.path.ManagerEvent = "/redfish/v1/Managers/1/LogServices/IEL/Entries"
	}

	// System event log
	if client.metrics.Events.Enabled {
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
		case ADVANTECH:
			client.path.Event = "/redfish/v1/Systems/0/LogServices/Log/Entries"
		}
	}

	// Extra
	if client.metrics.Extra.Enabled {
		if client.vendor == DELL {
			if client.redfish.Exists(DellSystemPath) {
				client.path.Extra = append(client.path.Extra, DellSystemPath)
			}
			if client.redfish.Exists(DellLicensePath) {
				client.path.Extra = append(client.path.Extra, DellLicensePath)
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
			client.path.Network = "/redfish/v1/Systems/1/NetworkAdapters/"
			client.path.Processors = "/redfish/v1/Systems/1/Processors/"
			client.path.Event = ""
			client.version = 4
		}
	}

	// Protocol features supported
	client.features.ExpandLevels = root.ProtocolFeaturesSupported.ExpandQuery.Levels
	client.features.Filter = root.ProtocolFeaturesSupported.FilterQuery

	return true
}

func (client *Client) RefreshSensorsUnified(mc *Collector, ch chan<- prometheus.Metric) bool {
	resp := ExpandedGroup[Sensor]{}
	ok := client.redfish.Get(client.path.Sensors+"?$expand=.($levels=1)", &resp)
	if !ok {
		return false
	}

	for _, s := range resp.Members {
		if s.Status.State != StateEnabled {
			continue
		}

		if s.ReadingType == "" {
			continue
		}

		if s.ReadingUnits == "" {
			continue
		}

		mc.NewSensorInfo(ch, &s)
		mc.NewSensorHealth(ch, &s)
		mc.NewSensorValue(ch, &s)
		mc.NewSensorLowerCritical(ch, &s)
		mc.NewSensorUpperCritical(ch, &s)
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

				mc.NewSensorsTemperature(ch, *c.Reading, strconv.Itoa(n), name, c.PhysicalContext, "celsius")
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
		mc.NewSensorsTemperature(ch, t.ReadingCelsius, id, t.Name, t.PhysicalContext, "celsius")
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
	if client.metrics.Sensors.Unified {
		if len(client.path.Sensors) > 0 && client.features.ExpandLevels {
			return client.RefreshSensorsUnified(mc, ch)
		}
	}

	if len(client.path.Thermal) > 0 {
		return client.RefreshSensorsOld(mc, ch)
	}

	if len(client.path.ThermalSubsystem) > 0 {
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

func (client *Client) RefreshManager(mc *Collector, ch chan<- prometheus.Metric) bool {
	mgr := ManagerResponse{}
	ok := client.redfish.Get(client.path.Manager, &mgr)
	if !ok {
		return false
	}

	// Issue #190 and #175
	if client.vendor == DELL {
		s := mgr.Links.Oem.Dell.DellAttributes.GetLinks()
		if slices.Contains(s, DellAttributesPath) {
			attr := DellAttributes{}
			ok := client.redfish.Get(DellAttributesPath, &attr)
			if ok {
				mgr.Model = attr.Attributes.InfoHWModel
				mgr.ManagerType = attr.Attributes.InfoType
			}
		}
	}

	mc.NewManagerInfo(ch, &mgr)
	mc.NewManagerHealth(ch, &mgr)
	mc.NewManagerRemoteConsoleEnabled(ch, &mgr)
	mc.NewManagerRemoteConsoleMaxSessions(ch, &mgr)

	log, ok := client.QueryEventLog(client.path.ManagerEvent, &client.metrics.Manager.Events)
	if log == nil {
		return ok
	}

	for v := range log.Items() {
		mc.NewManagerEventLogEntry(ch, v.Id, v.Message, v.Severity, v.Timestamp)
	}

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

		// iLO 4
		if (client.vendor == HPE) && (client.version == 4) {
			if resp.Status.State == "" && resp.Status.Health != "" {
				resp.Status.State = StateEnabled
			}
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
	if client.path.Network == "" {
		return true
	}

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

		// iLO 4
		if (client.vendor == HPE) && (client.version == 4) {
			if ni.Model == "" {
				ni.Model = ni.Name
			}
			mc.NewNetworkAdapterInfo(ch, &ni)
			for n, p := range ni.PhysicalPorts {
				port := NetworkPort{
					Id:                   strconv.Itoa(n),
					CurrentLinkSpeedMbps: p.SpeedMbps,
				}
				mc.NewNetworkPortCurrentSpeed(ch, ni.Id, &port)
			}
			continue
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
		// Issue #116
		if (client.vendor == HPE) && (client.version == 4) {
			if psu.FirmwareVersion == "0.00" {
				continue
			}
		}

		if psu.Status.State == StateAbsent {
			continue
		}

		id := strconv.Itoa(i)
		mc.NewPowerSupplyHealth(ch, psu.Status.Health, id)

		if client.vendor == HUAWEI && psu.Oem.Huawei != nil && psu.Oem.Huawei.PowerInputWatts > 0 {
			mc.NewPowerSupplyInputWatts(ch, psu.Oem.Huawei.PowerInputWatts, id)
		} else {
			mc.NewPowerSupplyInputWatts(ch, psu.PowerInputWatts, id)
		}
		if client.vendor == HUAWEI && psu.Oem.Huawei != nil && psu.Oem.Huawei.PowerOutputWatts > 0 {
			mc.NewPowerSupplyOutputWatts(ch, psu.Oem.Huawei.PowerOutputWatts, id)
		} else {
			mc.NewPowerSupplyOutputWatts(ch, psu.GetOutputPower(), id)
		}

		mc.NewPowerSupplyInputVoltage(ch, psu.LineInputVoltage, id)
		mc.NewPowerSupplyCapacityWatts(ch, psu.PowerCapacityWatts, id)
		mc.NewPowerSupplyEfficiencyPercent(ch, psu.EfficiencyPercent, id)

	}

	if client.vendor == INSPUR && len(resp.PowerControl) == 0 && resp.Oem.Public != nil {
		mc.NewPowerControlConsumedWatts(ch, resp.Oem.Public.TotalPower, "0", "Chassis Power")
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

	// Voltage sensors belong to the sensors metrics group, but the data lives
	// in the Power response fetched above, so when both groups are enabled they
	// are emitted here at no additional cost.
	if client.metrics.Sensors.Enabled {
		client.emitVoltages(mc, ch, &resp)
	}

	return true
}

// RefreshVoltages emits the voltage sensors from the legacy Power resource. It
// is used when the sensors metrics group is enabled but the power group is not;
// when both groups are enabled the voltages are instead emitted during
// RefreshPowerOld, from the response that is fetched there anyway.
func (client *Client) RefreshVoltages(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.path.Power == "" {
		return true
	}

	resp := PowerResponse{}
	ok := client.redfish.Get(client.path.Power, &resp)
	if !ok {
		return false
	}

	client.emitVoltages(mc, ch, &resp)
	return true
}

// emitVoltages emits the entries of the Voltages array found in the legacy
// Power resource. On many BMCs (e.g. Lenovo XCC) this includes the board rails
// and the CMOS/RTC battery. Entries without a usable reading are skipped.
func (client *Client) emitVoltages(mc *Collector, ch chan<- prometheus.Metric, resp *PowerResponse) {
	for i, v := range resp.Voltages {
		if v.Name == "" {
			continue
		}
		value, ok := asFloat64(v.ReadingVolts)
		if !ok {
			continue
		}
		mc.NewSensorsVoltage(ch, value, strconv.Itoa(i), v.Name, "volts")
	}
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
	log, ok := client.QueryEventLog(client.path.Event, &client.metrics.Events)
	if log == nil {
		return ok
	}

	// iDRAC 8 (issue #143)
	if client.vendor == DELL {
		if log.response.Id == "SEL" && strings.Contains(client.path.Event, "LogServices") {
			client.path.Event = "/redfish/v1/Managers/iDRAC.Embedded.1/Logs/Sel"
			return client.RefreshEventLog(mc, ch)
		}
	}

	for v := range log.Items() {
		mc.NewEventLogEntry(ch, v.Id, v.Message, v.Severity, v.Timestamp)
	}

	return true
}

func (client *Client) RefreshSimpleStorage(mc *Collector, ch chan<- prometheus.Metric) bool {
	group := GroupResponse{}
	if !client.redfish.Get(client.path.SimpleStorage, &group) {
		return false
	}

	for _, link := range group.Members.GetLinks() {
		resp := SimpleStorage{}
		if !client.redfish.Get(link, &resp) {
			return false
		}

		controller := StorageController{
			Id:     resp.Id,
			Name:   resp.Name,
			Status: resp.Status,
		}

		mc.NewStorageControllerInfo(ch, resp.Id, &controller)
		mc.NewStorageControllerHealth(ch, resp.Id, &controller)

		for i, device := range resp.Devices {
			drive := StorageDrive{
				Id:            fmt.Sprintf("%s-%d", resp.Id, i),
				Name:          device.Name,
				Manufacturer:  device.Manufacturer,
				Model:         device.Model,
				Status:        device.Status,
				CapacityBytes: device.CapacityBytes,
			}

			mc.NewStorageDriveInfo(ch, resp.Id, &drive)
			mc.NewStorageDriveHealth(ch, resp.Id, &drive)
			mc.NewStorageDriveCapacity(ch, resp.Id, &drive)
		}
	}

	return true
}

func (client *Client) RefreshStorage(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.path.Storage == "" {
		if client.path.SimpleStorage != "" {
			return client.RefreshSimpleStorage(mc, ch)
		}
		return true
	}

	group := GroupResponse{}
	ok := client.redfish.Get(client.path.Storage, &group)
	if !ok {
		return false
	}
	re := regexp.MustCompile(`Chassis\/([^\/]+)`)

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
				drive.PredictedLifeLeft = new(100.0 - drive.SSDEnduranceUtilizationPercentage)
			}

			// Inspur (issue #162)
			if (client.vendor == INSPUR) && (drive.PredictedLifeLeft == nil) {
				if v, ok := asFloat64(drive.Oem.Public.TimeLeft.String()); ok {
					drive.PredictedLifeLeft = &v
				}
			}

			// Supermicro (issue #164)
			if client.vendor == SUPERMICRO {
				match := re.FindStringSubmatch(drive.OdataId)
				if len(match) == 2 {
					drive.Id = fmt.Sprintf("%s:%s", match[1], drive.Id)
				}
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

				if ctlr.CacheSummary != nil {
					mc.NewStorageControllerCacheSize(ch, storage.Id, &ctlr)
					mc.NewStorageControllerCacheHealth(ch, storage.Id, &ctlr)
				}
			}
		} else {
			for _, ctlr := range storage.StorageControllers {
				if len(ctlr.MemberId) > 0 {
					ctlr.Id = ctlr.MemberId
				}

				mc.NewStorageControllerInfo(ch, storage.Id, &ctlr)
				mc.NewStorageControllerSpeed(ch, storage.Id, &ctlr)
				mc.NewStorageControllerHealth(ch, storage.Id, &ctlr)

				if ctlr.CacheSummary != nil {
					mc.NewStorageControllerCacheSize(ch, storage.Id, &ctlr)
					mc.NewStorageControllerCacheHealth(ch, storage.Id, &ctlr)
				}
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
	if client.path.Memory == "" {
		return true
	}

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
	if client.vendor != DELL {
		return true
	}

	result := true

	if slices.Contains(client.path.Extra, DellSystemPath) {
		resp := DellSystem{}
		ok := client.redfish.Get(DellSystemPath, &resp)
		if ok {
			mc.NewDellBatteryRollupHealth(ch, &resp)
			mc.NewDellEstimatedSystemAirflowCFM(ch, &resp)
		} else {
			result = false
		}
	}

	if slices.Contains(client.path.Extra, DellLicensePath) {
		licenses := ExpandedGroup[DellLicense]{}
		if client.redfish.Get(DellLicensePath, &licenses) {
			for _, v := range licenses.Members {
				mc.NewDellManagerLicenseInfo(ch, &v)
			}
		} else {
			result = false
		}
	}

	return result
}

func (client *Client) RefreshPDUs(mc *Collector, ch chan<- prometheus.Metric) bool {
	result := true

	for _, path := range client.path.RackPDUs {
		pd := PowerDistribution{}
		ok := client.redfish.Get(path, &pd)
		if !ok {
			result = false
			continue
		}

		id := pd.Id.String()
		mc.NewPduInfo(ch, id, &pd)
		mc.NewPduHealth(ch, id, &pd)

		pdm := PowerDistributionMetrics{}
		ok = client.redfish.Get(pd.Metrics.OdataId, &pdm)
		if !ok {
			result = false
			continue
		}

		mc.NewPduPowerWatts(ch, id, &pdm)
		mc.NewPduPowerApparentVA(ch, id, &pdm)
		mc.NewPduPowerFactor(ch, id, &pdm)
		mc.NewPduEnergyKWh(ch, id, &pdm)
	}

	return result
}
