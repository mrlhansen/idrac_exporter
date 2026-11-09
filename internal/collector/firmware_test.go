package collector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const (
	expandSupported = iota // $expand works
	expandIgnored          // $expand is accepted, but only links are returned
	expandRejected         // $expand returns an error
)

// mockBMC serves a recorded Redfish tree from testdata/firmware
type mockBMC struct {
	docs     map[string]json.RawMessage
	expand   int
	requests atomic.Int64
}

func newMockBMC(t *testing.T, name string, expand int) (*mockBMC, *httptest.Server) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "firmware", name+".json"))
	if err != nil {
		t.Fatal(err)
	}

	bmc := &mockBMC{expand: expand}
	if err := json.Unmarshal(data, &bmc.docs); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(bmc)
	t.Cleanup(srv.Close)
	return bmc, srv
}

func (m *mockBMC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.requests.Add(1)

	doc, ok := m.docs[strings.TrimSuffix(r.URL.Path, "/")]
	if !ok {
		http.NotFound(w, r)
		return
	}

	if r.URL.Query().Get("$expand") == "" {
		w.Write(doc)
		return
	}

	switch m.expand {
	case expandRejected:
		http.Error(w, "unsupported query", http.StatusBadRequest)
		return
	case expandIgnored:
		w.Write(doc)
		return
	}

	// Inline every member of the collection
	var group struct {
		Members []Odata `json:"Members"`
	}
	if err := json.Unmarshal(doc, &group); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	members := []json.RawMessage{}
	for _, v := range group.Members {
		members = append(members, m.docs[v.OdataId])
	}

	resp, _ := json.Marshal(map[string]any{"Members": members})
	w.Write(resp)
}

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()

	config.Config = &config.RootConfig{
		Timeout:       5,
		Concurrency:   4,
		MetricsPrefix: "idrac",
	}

	auth := &config.AuthConfig{
		Username:  "user",
		Password:  "pass",
		Scheme:    "http",
		BasicAuth: true,
	}
	auth.Metrics.Firmware.Enabled = true

	client := NewClient(strings.TrimPrefix(srv.URL, "http://"), auth)
	if client == nil {
		t.Fatal("failed to create client")
	}
	return client
}

type firmwareMetric struct {
	id, name, version, updateable string
}

func collectFirmware(t *testing.T, client *Client) []firmwareMetric {
	t.Helper()

	mc := NewCollector()
	ch := make(chan prometheus.Metric, 1000)
	if !client.RefreshFirmware(mc, ch) {
		t.Fatal("RefreshFirmware failed")
	}
	close(ch)

	result := []firmwareMetric{}
	for m := range ch {
		pb := &dto.Metric{}
		if err := m.Write(pb); err != nil {
			t.Fatal(err)
		}
		labels := map[string]string{}
		for _, l := range pb.GetLabel() {
			labels[l.GetName()] = l.GetValue()
		}
		result = append(result, firmwareMetric{
			id:         labels["id"],
			name:       labels["name"],
			version:    labels["version"],
			updateable: labels["updateable"],
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].id < result[j].id })
	return result
}

func findFirmware(list []firmwareMetric, id string) (firmwareMetric, bool) {
	for _, f := range list {
		if f.id == id {
			return f, true
		}
	}
	return firmwareMetric{}, false
}

func TestFirmwareInventory(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		present []firmwareMetric
		absent  []string
	}{
		{
			// 29 entries for 21 components: 21 installed, 6 current and 2 previous
			name:  "dell_r750",
			count: 21,
			present: []firmwareMetric{
				{"iDRAC.Embedded.1-1", "Integrated Dell Remote Access Controller", "6.00.30.00", "true"},
				{"BIOS.Setup.1-1", "BIOS", "1.8.2", "true"},
				{"InfiniBand.Slot.7-1", "Mellanox ConnectX-6 Dual Port 10 GbE QSFP Network Adapter", "20.31.10.14", "true"},
				{"PSU.Slot.1", "Power Supply.Slot.1", "00.2A.2B", "true"},
			},
			absent: []string{
				"Previous-25227-4.40.29.00__iDRAC.Embedded.1-1",
				"Installed-25227-6.00.30.00__iDRAC.Embedded.1-1",
				"Current-159-1.8.2__BIOS.Setup.1-1",
			},
		},
		{
			// iLO 5 does not report the state for most entries
			name:  "hpe_ilo5",
			count: 48,
			present: []firmwareMetric{
				{"1", "iLO 5", "2.72 Sep 04 2022", "true"},
				{"2", "System ROM", "A42 v2.60 (08/11/2022)", "true"},
				{"14", "TPM Firmware", "73.64", "false"},
			},
		},
		{
			// Pending images are disabled, the backup BMC image is kept
			name:  "lenovo_xcc",
			count: 41,
			present: []firmwareMetric{
				{"BMC-Primary", "Firmware:BMC", "42S-2.83", "true"},
				{"BMC-Backup", "Firmware:BMC", "42S-2.83", "true"},
				{"UEFI", "Firmware:UEFI", "22J-1.51", "true"},
				{"GPU1", "GPU firmware", "95.02.39.00.01", "false"},
			},
			absent: []string{"BMC-Primary-Pending", "UEFI-Pending"},
		},
		{
			// Empty backup flash is reported with the version "Not Present"
			name:  "supermicro_hgx",
			count: 104,
			present: []firmwareMetric{
				{"BMC", "BMC", "11.01.01", "true"},
				{"Golden_BMC", "BMC Golden", "01.00.26", "true"},
				{"HGX_FW_GPU_SXM_1", "Software Inventory", "96.00.61.00.01", "true"},
				{"HGX_FW_NVSwitch_0", "Software Inventory", "", "true"},
			},
			absent: []string{"Backup_BMC", "Backup_BIOS"},
		},
		{
			// GPUs that are not present are reported as absent
			name:  "nvidia_gb200",
			count: 26,
			present: []firmwareMetric{
				{"FW_BMC_0", "Software Inventory", "GB200Nvl-24.12-8", "true"},
				{"HGX_FW_CPU_0", "Software Inventory", "02.03.13", "false"},
				{"HGX_InfoROM_GPU_0", "Software Inventory", "G548.0201.00.03", "true"},
			},
			absent: []string{"HGX_FW_GPU_0", "HGX_FW_GPU_3"},
		},
	}

	for _, tt := range tests {
		for mode, label := range []string{"expand", "expand_ignored", "expand_rejected"} {
			t.Run(tt.name+"/"+label, func(t *testing.T) {
				_, srv := newMockBMC(t, tt.name, mode)
				client := newTestClient(t, srv)
				if client.path.Firmware != "/redfish/v1/UpdateService/FirmwareInventory" {
					t.Fatalf("unexpected firmware inventory path %q", client.path.Firmware)
				}

				list := collectFirmware(t, client)
				if len(list) != tt.count {
					t.Errorf("got %d firmware entries, expected %d", len(list), tt.count)
				}

				seen := map[string]bool{}
				for _, f := range list {
					if seen[f.id] {
						t.Errorf("duplicate id %q", f.id)
					}
					seen[f.id] = true
					if f.version == "" {
						t.Errorf("empty version for %q", f.id)
					}
				}

				for _, want := range tt.present {
					got, ok := findFirmware(list, want.id)
					if !ok {
						t.Errorf("missing firmware %q", want.id)
						continue
					}
					if want.version == "" {
						want.version = got.version
					}
					if got != want {
						t.Errorf("got %+v, expected %+v", got, want)
					}
				}

				for _, id := range tt.absent {
					if _, ok := findFirmware(list, id); ok {
						t.Errorf("firmware %q should have been filtered", id)
					}
				}
			})
		}
	}
}

func TestFirmwareInventoryRequests(t *testing.T) {
	// The Supermicro HGX inventory has 106 entries
	tests := []struct {
		mode     int
		requests int64
	}{
		{expandSupported, 1},
		{expandIgnored, 107},
		{expandRejected, 108},
	}

	for _, tt := range tests {
		bmc, srv := newMockBMC(t, "supermicro_hgx", tt.mode)
		client := newTestClient(t, srv)

		bmc.requests.Store(0)
		collectFirmware(t, client)
		if n := bmc.requests.Load(); n != tt.requests {
			t.Errorf("mode %d: first scrape used %d requests, expected %d", tt.mode, n, tt.requests)
		}

		// After a rejected $expand the exporter falls back to plain requests
		bmc.requests.Store(0)
		collectFirmware(t, client)
		expected := tt.requests
		if tt.mode == expandRejected {
			expected = 107
		}
		if n := bmc.requests.Load(); n != expected {
			t.Errorf("mode %d: second scrape used %d requests, expected %d", tt.mode, n, expected)
		}
	}
}

func TestFirmwareInventoryDisabled(t *testing.T) {
	_, srv := newMockBMC(t, "dell_r750", expandSupported)

	config.Config = &config.RootConfig{Timeout: 5, Concurrency: 4, MetricsPrefix: "idrac"}
	auth := &config.AuthConfig{Username: "user", Password: "pass", Scheme: "http", BasicAuth: true}

	client := NewClient(strings.TrimPrefix(srv.URL, "http://"), auth)
	if client == nil {
		t.Fatal("failed to create client")
	}
	if client.path.Firmware != "" {
		t.Errorf("firmware inventory should not be discovered when the group is disabled")
	}
}

func TestParseDellInventoryId(t *testing.T) {
	tests := []struct {
		id, state, fqdd string
		ok              bool
	}{
		{"Installed-25227-6.00.30.00__iDRAC.Embedded.1-1", "Installed", "iDRAC.Embedded.1-1", true},
		{"Previous-159-1.2.4__BIOS.Setup.1-1", "Previous", "BIOS.Setup.1-1", true},
		{"Installed-0-2.1.3__Disk.Bay.0:Enclosure.Internal.0-1", "Installed", "Disk.Bay.0:Enclosure.Internal.0-1", true},
		{"Available-25227-7.00.00.00__iDRAC.Embedded.1-1", "Available", "iDRAC.Embedded.1-1", true},
		{"BMC", "", "", false},
		{"Installed__", "", "", false},
	}

	for _, tt := range tests {
		state, fqdd, ok := parseDellInventoryId(tt.id)
		if state != tt.state || fqdd != tt.fqdd || ok != tt.ok {
			t.Errorf("parseDellInventoryId(%q) = %q, %q, %v", tt.id, state, fqdd, ok)
		}
	}
}
