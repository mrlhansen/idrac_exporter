package redfish

import (
	"time"
)

const (
	StateEnabled = "Enabled"
)

// Odata is a common structure to unmarshal Open Data Protocol metadata
type Odata struct {
	OdataContext string `json:"@odata.context"`
	OdataId      string `json:"@odata.id"`
	OdataType    string `json:"@odata.type"`
}

// Status is a common structure used in any entity with a status
type Status struct {
	Health       string `json:"Health"`
	HealthRollup string `json:"HealthRollup"`
	State        string `json:"State"`
}

// V1Response represents structure of the response body from /redfish/v1
type V1Response struct {
	RedfishVersion     string `json:"RedfishVersion"`
	Name               string `json:"Name"`
	Product            string `json:"Product"`
	Vendor             string `json:"Vendor"`
	Description        string `json:"Description"`
	AccountService     Odata  `json:"AccountService"`
	CertificateService Odata  `json:"CertificateService"`
	Chassis            Odata  `json:"Chassis"`
	EventService       Odata  `json:"EventService"`
	Fabrics            Odata  `json:"Fabrics"`
	JobService         Odata  `json:"JobService"`
	JsonSchemas        Odata  `json:"JsonSchemas"`
	Managers           Odata  `json:"Managers"`
	Registries         Odata  `json:"Registries"`
	SessionService     Odata  `json:"SessionService"`
	Systems            Odata  `json:"Systems"`
	Tasks              Odata  `json:"Tasks"`
	TelemetryService   Odata  `json:"TelemetryService"`
	UpdateService      Odata  `json:"UpdateService"`
}

type GroupResponse struct {
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Members     []Odata `json:"Members"`
}

type ChassisResponse struct {
	Name               string `json:"Name"`
	AssetTag           string `json:"AssetTag"`
	SerialNumber       string `json:"SerialNumber"`
	PartNumber         string `json:"PartNumber"`
	Model              string `json:"Model"`
	ChassisType        string `json:"ChassisType"`
	Manufacturer       string `json:"Manufacturer"`
	Description        string `json:"Description"`
	SKU                string `json:"SKU"`
	PowerState         string `json:"PowerState"`
	EnvironmentalClass string `json:"EnvironmentalClass"`
	IndicatorLED       string `json:"IndicatorLED"`
	Assembly           Odata  `json:"Assembly"`
	Location           *struct {
		Info       string `json:"Info"`
		InfoFormat string `json:"InfoFormat"`
		Placement  struct {
			Rack string `json:"Rack"`
			Row  string `json:"Row"`
		} `json:"Placement"`
		PostalAddress struct {
			Building string `json:"Building"`
			Room     string `json:"Room"`
		} `json:"PostalAddress"`
	} `json:"Location"`
	Memory           Odata  `json:"Memory"`
	NetworkAdapters  Odata  `json:"NetworkAdapters"`
	PCIeSlots        Odata  `json:"PCIeSlots"`
	Power            Odata  `json:"Power"`
	Sensors          Odata  `json:"Sensors"`
	Status           Status `json:"Status"`
	Thermal          Odata  `json:"Thermal"`
	PhysicalSecurity *struct {
		IntrusionSensor       string `json:"IntrusionSensor"`
		IntrusionSensorNumber int    `json:"IntrusionSensorNumber"`
		IntrusionSensorReArm  string `json:"IntrusionSensorReArm"`
	} `json:"PhysicalSecurity"`
}

type ThermalResponse struct {
	Name         string        `json:"Name"`
	Description  string        `json:"Description"`
	Fans         []Fan         `json:"Fans"`
	Temperatures []Temperature `json:"Temperatures"`
	Redundancy   []struct {
		Name              string        `json:"Name"`
		MaxNumSupported   int           `json:"MaxNumSupported"`
		MinNumNeeded      int           `json:"MinNumNeeded"`
		Mode              string        `json:"Mode"`
		RedundancyEnabled bool          `json:"RedundancyEnabled"`
		RedundancySet     []interface{} `json:"RedundancySet"`
		Status            Status        `json:"Status"`
	} `json:"Redundancy"`
}

type Fan struct {
	Name                      string        `json:"Name"`
	FanName                   string        `json:"FanName"`
	Assembly                  Odata         `json:"Assembly"`
	HotPluggable              bool          `json:"HotPluggable"`
	LowerThresholdCritical    int           `json:"LowerThresholdCritical"`
	LowerThresholdFatal       interface{}   `json:"LowerThresholdFatal"`
	LowerThresholdNonCritical int           `json:"LowerThresholdNonCritical"`
	MaxReadingRange           interface{}   `json:"MaxReadingRange"`
	MinReadingRange           interface{}   `json:"MinReadingRange"`
	PhysicalContext           string        `json:"PhysicalContext"`
	Reading                   float64       `json:"Reading"`
	CurrentReading            float64       `json:"CurrentReading"`
	Units                     string        `json:"Units"`
	ReadingUnits              string        `json:"ReadingUnits"`
	Redundancy                []interface{} `json:"Redundancy"`
	SensorNumber              int           `json:"SensorNumber"`
	Status                    Status        `json:"Status"`
	UpperThresholdCritical    interface{}   `json:"UpperThresholdCritical"`
	UpperThresholdFatal       interface{}   `json:"UpperThresholdFatal"`
	UpperThresholdNonCritical interface{}   `json:"UpperThresholdNonCritical"`
}

func (f *Fan) GetName() string {
	if f.FanName != "" {
		return f.FanName
	}
	return f.Name
}

func (f *Fan) GetReading() float64 {
	if f.Reading > 0 {
		return f.Reading
	}
	return f.CurrentReading
}

func (f *Fan) GetUnits() string {
	if f.ReadingUnits != "" {
		return f.ReadingUnits
	}
	return f.Units
}

type Temperature struct {
	Name                      string  `json:"Name"`
	SensorNumber              int     `json:"SensorNumber"`
	ReadingCelsius            float64 `json:"ReadingCelsius"`
	MaxReadingRangeTemp       float64 `json:"MaxReadingRangeTemp"`
	MinReadingRangeTemp       float64 `json:"MinReadingRangeTemp"`
	PhysicalContext           string  `json:"PhysicalContext"`
	LowerThresholdCritical    float64 `json:"LowerThresholdCritical"`
	LowerThresholdFatal       float64 `json:"LowerThresholdFatal"`
	LowerThresholdNonCritical float64 `json:"LowerThresholdNonCritical"`
	UpperThresholdCritical    float64 `json:"UpperThresholdCritical"`
	UpperThresholdFatal       float64 `json:"UpperThresholdFatal"`
	UpperThresholdNonCritical float64 `json:"UpperThresholdNonCritical"`
	Status                    Status  `json:"Status"`
}

type SystemResponse struct {
	IndicatorLED string `json:"IndicatorLED"`
	Manufacturer string `json:"Manufacturer"`
	AssetTag     string `json:"AssetTag"`
	PartNumber   string `json:"PartNumber"`
	Description  string `json:"Description"`
	HostName     string `json:"HostName"`
	PowerState   string `json:"PowerState"`
	Bios         Odata  `json:"Bios"`
	BiosVersion  string `json:"BiosVersion"`
	Boot         *struct {
		BootOptions                                    Odata       `json:"BootOptions"`
		Certificates                                   Odata       `json:"Certificates"`
		BootOrder                                      []string    `json:"BootOrder"`
		BootSourceOverrideEnabled                      string      `json:"BootSourceOverrideEnabled"`
		BootSourceOverrideMode                         string      `json:"BootSourceOverrideMode"`
		BootSourceOverrideTarget                       string      `json:"BootSourceOverrideTarget"`
		UefiTargetBootSourceOverride                   interface{} `json:"UefiTargetBootSourceOverride"`
		BootSourceOverrideTargetRedfishAllowableValues []string    `json:"BootSourceOverrideTarget@Redfish.AllowableValues"`
	} `json:"Boot"`
	EthernetInterfaces Odata `json:"EthernetInterfaces"`
	HostWatchdogTimer  *struct {
		FunctionEnabled bool   `json:"FunctionEnabled"`
		Status          Status `json:"Status"`
		TimeoutAction   string `json:"TimeoutAction"`
	} `json:"HostWatchdogTimer"`
	HostingRoles  []interface{} `json:"HostingRoles"`
	Memory        Odata         `json:"Memory"`
	MemorySummary *struct {
		MemoryMirroring      string  `json:"MemoryMirroring"`
		Status               Status  `json:"Status"`
		TotalSystemMemoryGiB float64 `json:"TotalSystemMemoryGiB"`
	} `json:"MemorySummary"`
	Model             string  `json:"Model"`
	Name              string  `json:"Name"`
	NetworkInterfaces Odata   `json:"NetworkInterfaces"`
	PCIeDevices       []Odata `json:"PCIeDevices"`
	PCIeFunctions     []Odata `json:"PCIeFunctions"`
	ProcessorSummary  *struct {
		Count                 int    `json:"Count"`
		LogicalProcessorCount int    `json:"LogicalProcessorCount"`
		Model                 string `json:"Model"`
		Status                Status `json:"Status"`
	} `json:"ProcessorSummary"`
	Processors     Odata  `json:"Processors"`
	SKU            string `json:"SKU"`
	SecureBoot     Odata  `json:"SecureBoot"`
	SerialNumber   string `json:"SerialNumber"`
	SimpleStorage  Odata  `json:"SimpleStorage"`
	Status         Status `json:"Status"`
	Storage        Odata  `json:"Storage"`
	SystemType     string `json:"SystemType"`
	TrustedModules []struct {
		FirmwareVersion string `json:"FirmwareVersion"`
		InterfaceType   string `json:"InterfaceType"`
		Status          Status `json:"Status"`
	} `json:"TrustedModules"`
}

type PowerResponse struct {
	Name          string             `json:"Name"`
	Description   string             `json:"Description"`
	PowerControl  []PowerControlUnit `json:"PowerControl"`
	PowerSupplies []PowerSupplyUnit  `json:"PowerSupplies"`
	Redundancy    []struct {
		Name            string `json:"Name"`
		MinNumNeeded    int    `json:"MinNumNeeded"`
		MaxNumSupported int    `json:"MaxNumSupported"`
		Mode            string `json:"Mode"`
		Status          Status `json:"Status"`
	} `json:"Redundancy"`
	Voltages []struct {
		Name                      string  `json:"Name"`
		ReadingVolts              float64 `json:"ReadingVolts"`
		SensorNumber              int     `json:"SensorNumber"`
		PhysicalContext           string  `json:"PhysicalContext"`
		LowerThresholdCritical    float64 `json:"LowerThresholdCritical"`
		LowerThresholdFatal       float64 `json:"LowerThresholdFatal"`
		LowerThresholdNonCritical float64 `json:"LowerThresholdNonCritical"`
		UpperThresholdCritical    float64 `json:"UpperThresholdCritical"`
		UpperThresholdFatal       float64 `json:"UpperThresholdFatal"`
		UpperThresholdNonCritical float64 `json:"UpperThresholdNonCritical"`
		Status                    Status  `json:"Status"`
	} `json:"Voltages"`
}

type PowerControlUnit struct {
	Name                string  `json:"Name"`
	Id                  string  `json:"Id"`
	PowerAllocatedWatts float64 `json:"PowerAllocatedWatts"`
	PowerAvailableWatts float64 `json:"PowerAvailableWatts"`
	PowerCapacityWatts  float64 `json:"PowerCapacityWatts"`
	PowerConsumedWatts  float64 `json:"PowerConsumedWatts"`
	PowerRequestedWatts float64 `json:"PowerRequestedWatts"`
	PowerLimit          *struct {
		CorrectionInMs int    `json:"CorrectionInMs"`
		LimitException string `json:"LimitException"`
		LimitInWatts   int    `json:"LimitInWatts"`
	} `json:"PowerLimit"`
	PowerMetrics *struct {
		AverageConsumedWatts float64 `json:"AverageConsumedWatts"`
		IntervalInMinutes    int     `json:"IntervalInMin"`
		MaxConsumedWatts     float64 `json:"MaxConsumedWatts"`
		MinConsumedWatts     float64 `json:"MinConsumedWatts"`
	} `json:"PowerMetrics"`
}

type PowerSupplyUnit struct {
	Name            string `json:"Name"`
	Assembly        Odata  `json:"Assembly"`
	FirmwareVersion string `json:"FirmwareVersion"`
	InputRanges     []struct {
		InputType          string  `json:"InputType"`
		MaximumFrequencyHz float64 `json:"MaximumFrequencyHz"`
		MaximumVoltage     float64 `json:"MaximumVoltage"`
		MinimumFrequencyHz float64 `json:"MinimumFrequencyHz"`
		MinimumVoltage     float64 `json:"MinimumVoltage"`
		OutputWattage      float64 `json:"OutputWattage"`
	} `json:"InputRanges"`
	HotPluggable         bool    `json:"HotPluggable"`
	EfficiencyPercent    float64 `json:"EfficiencyPercent"`
	PowerOutputWatts     float64 `json:"PowerOutputWatts"`
	LastPowerOutputWatts float64 `json:"LastPowerOutputWatts"`
	PowerInputWatts      float64 `json:"PowerInputWatts"`
	PowerCapacityWatts   float64 `json:"PowerCapacityWatts"`
	LineInputVoltage     float64 `json:"LineInputVoltage"`
	LineInputVoltageType string  `json:"LineInputVoltageType"`
	Manufacturer         string  `json:"Manufacturer"`
	Model                string  `json:"Model"`
	PartNumber           string  `json:"PartNumber"`
	PowerSupplyType      string  `json:"PowerSupplyType"`
	SerialNumber         string  `json:"SerialNumber"`
	SparePartNumber      string  `json:"SparePartNumber"`
	Status               Status  `json:"Status"`
	Redundancy           []struct {
		Name            string  `json:"Name"`
		MaxNumSupported int     `json:"MaxNumSupported"`
		MinNumNeeded    int     `json:"MinNumNeeded"`
		Mode            string  `json:"Mode"`
		RedundancySet   []Odata `json:"RedundancySet"`
		Status          Status  `json:"Status"`
	} `json:"Redundancy"`
}

func (psu *PowerSupplyUnit) GetOutputPower() float64 {
	if psu.PowerOutputWatts > 0 {
		return psu.PowerOutputWatts
	}
	return psu.LastPowerOutputWatts
}

type IdracSelResponse struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Members     []struct {
		Id           string        `json:"Id"`
		Name         string        `json:"Name"`
		Created      time.Time     `json:"Created"`
		Description  string        `json:"Description"`
		EntryCode    interface{}   `json:"EntryCode"` // An array in iDRAC8 but a string in iDRAC9
		EntryType    string        `json:"EntryType"`
		Message      string        `json:"Message"`
		MessageArgs  []interface{} `json:"MessageArgs"`
		MessageId    string        `json:"MessageId"`
		SensorNumber int           `json:"SensorNumber"`
		SensorType   string        `json:"SensorType"`
		Severity     string        `json:"Severity"`
	} `json:"Members"`
}
