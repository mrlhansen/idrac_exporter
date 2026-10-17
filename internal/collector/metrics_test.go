package collector

import (
	"encoding/json"
	"testing"
)

func TestNormalizeBootMode(t *testing.T) {
	tests := map[string]string{
		"Uefi":    "UEFI",
		"uefi":    "UEFI",
		"BIOS":    "BIOS",
		"legacy":  "BIOS",
		" Other ": "Other",
		"":        "",
	}

	for input, want := range tests {
		if got := normalizeBootMode(input); got != want {
			t.Errorf("normalizeBootMode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLicenseIDAndDescription(t *testing.T) {
	license := &DellLicense{
		EntitlementID:      "entitlement-id",
		LicenseDescription: []string{"iDRAC9 Enterprise License"},
	}

	if got := licenseID(license); got != "entitlement-id" {
		t.Errorf("licenseID() = %q, want entitlement-id", got)
	}
	if got := licenseDescription(license); got != "iDRAC9 Enterprise License" {
		t.Errorf("licenseDescription() = %q, want iDRAC9 Enterprise License", got)
	}
}

func TestLegacyDellResponseModels(t *testing.T) {
	var system SystemResponse
	if err := json.Unmarshal([]byte(`{
		"Bios": {"@odata.id": "/redfish/v1/Systems/System.Embedded.1/Bios"},
		"SimpleStorage": {"@odata.id": "/redfish/v1/Systems/System.Embedded.1/Storage/Controllers"},
		"EthernetInterfaces": {"@odata.id": "/redfish/v1/Systems/System.Embedded.1/EthernetInterfaces"}
	}`), &system); err != nil {
		t.Fatal(err)
	}
	if system.Bios.OdataId == "" || system.SimpleStorage.OdataId == "" || system.EthernetInterfaces.OdataId == "" {
		t.Fatalf("legacy system links were not decoded: %+v", system)
	}

	var manager ManagerResponse
	if err := json.Unmarshal([]byte(`{
		"GraphicalConsole": {"ConnectTypesSupported": ["KVMIP"], "MaxConcurrentSessions": 6, "ServiceEnabled": true},
		"Links": {"Oem": {"Dell": {"DellLicenseCollection": {"@odata.id": "/redfish/v1/Managers/iDRAC.Embedded.1/Oem/Dell/DellLicenses"}}}}
	}`), &manager); err != nil {
		t.Fatal(err)
	}
	if manager.GraphicalConsole == nil || manager.GraphicalConsole.ConnectTypesSupported[0] != "KVMIP" {
		t.Fatalf("graphical console was not decoded: %+v", manager.GraphicalConsole)
	}
	if manager.Links.Oem.Dell.DellLicenseCollection.OdataId == "" {
		t.Fatalf("license collection link was not decoded: %+v", manager.Links.Oem.Dell)
	}
}
