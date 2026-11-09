package collector

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func (client *Client) RefreshFirmware(mc *Collector, ch chan<- prometheus.Metric) bool {
	if client.path.Firmware == "" {
		return true
	}

	inventory, ok := client.getFirmwareInventory()
	if !ok {
		return false
	}

	for _, fw := range filterFirmwareInventory(client.vendor, inventory) {
		mc.NewFirmwareInfo(ch, &fw)
	}

	return true
}

// getFirmwareInventory returns all entries in the firmware inventory. The
// inventory can easily hold 50-100 entries on GPU servers, so when the service
// supports it, the whole collection is fetched in a single request using
// $expand instead of one request per entry.
func (client *Client) getFirmwareInventory() ([]SoftwareInventory, bool) {
	if client.features.ExpandLevels && !client.firmwareNoExpand {
		resp := ExpandedGroup[SoftwareInventory]{}
		if client.redfish.Get(client.path.Firmware+"?$expand=.($levels=1)", &resp) {
			result := make([]SoftwareInventory, 0, len(resp.Members))
			for _, fw := range resp.Members {
				// Members that were not expanded only contain a link
				if fw.Id == "" && fw.OdataId != "" {
					if !client.redfish.Get(fw.OdataId, &fw) {
						return nil, false
					}
				}
				result = append(result, fw)
			}
			return result, true
		}
		// Do not try $expand again for this host
		client.firmwareNoExpand = true
	}

	group := GroupResponse{}
	if !client.redfish.Get(client.path.Firmware, &group) {
		return nil, false
	}

	result := make([]SoftwareInventory, 0, len(group.Members))
	for _, link := range group.Members.GetLinks() {
		fw := SoftwareInventory{}
		if !client.redfish.Get(link, &fw) {
			return nil, false
		}
		result = append(result, fw)
	}

	return result, true
}

// parseDellInventoryId splits a Dell firmware inventory id of the form
// "<State>-<ComponentId>-<Version>__<FQDD>" into its state and FQDD,
// e.g. "Installed-25227-6.00.30.00__iDRAC.Embedded.1-1".
func parseDellInventoryId(id string) (string, string, bool) {
	prefix, fqdd, ok := strings.Cut(id, "__")
	if !ok || fqdd == "" {
		return "", "", false
	}
	state, _, ok := strings.Cut(prefix, "-")
	if !ok {
		return "", "", false
	}
	return state, fqdd, true
}

// filterFirmwareInventory keeps the entries that describe firmware which is
// actually present on the machine, and drops images that are absent, pending
// or kept for rollback.
//
//   - Entries with state "Absent" (e.g. empty GPU slots) or "Disabled" (e.g.
//     pending images on Lenovo XCC) are skipped.
//   - Entries without a version, or with the version "Not Present" (empty
//     backup flash on Supermicro), are skipped.
//   - Dell lists every component several times, with the version encoded in
//     the id: "Installed-", "Current-", "Previous-" (rollback image) and
//     "Available-" (staged in the iDRAC repository). Only the installed or
//     current image is kept, and the id is replaced by the component FQDD, so
//     that the id label does not change when the firmware is updated.
func filterFirmwareInventory(vendor int, inventory []SoftwareInventory) []SoftwareInventory {
	current := map[string]bool{}
	if vendor == DELL {
		for _, fw := range inventory {
			state, fqdd, ok := parseDellInventoryId(fw.Id)
			if ok && state == "Current" {
				current[fqdd] = true
			}
		}
	}

	result := []SoftwareInventory{}
	for _, fw := range inventory {
		if fw.Status.State == StateAbsent || fw.Status.State == StateDisabled {
			continue
		}

		version := strings.TrimSpace(fw.Version.String())
		if version == "" || strings.EqualFold(version, "Not Present") {
			continue
		}
		fw.Version = xstring(version)

		if vendor == DELL {
			state, fqdd, ok := parseDellInventoryId(fw.Id)
			if ok {
				switch state {
				case "Current":
				case "Installed":
					if current[fqdd] {
						continue
					}
				default:
					continue
				}
				fw.Id = fqdd
			}
		}

		result = append(result, fw)
	}

	return result
}
