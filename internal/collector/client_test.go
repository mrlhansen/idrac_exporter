package collector

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mrlhansen/idrac_exporter/internal/config"
)

func TestLatestManagerLogId(t *testing.T) {
	const entries = "/redfish/v1/Managers/iDRAC.Embedded.1/LogServices/Lclog/Entries"

	cases := []struct {
		name    string
		vendor  int
		members string
		wantId  uint64
		wantOk  bool
	}{
		{"newest entry", DELL, `[{"Id":"10825"},{"Id":"10824"}]`, 10825, true},
		{"non-numeric id", DELL, `[{"Id":"abc"}]`, 0, false},
		{"empty log", DELL, `[]`, 0, false},
		{"not a Dell", HPE, `[{"Id":"10825"}]`, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != entries || r.URL.Query().Get("$top") != "1" {
					http.NotFound(w, r)
					return
				}
				fmt.Fprintf(w, `{"Members":%s}`, tc.members)
			}))
			defer server.Close()

			config.Config = &config.RootConfig{Timeout: 5, Concurrency: 1}
			host := strings.TrimPrefix(server.URL, "https://")

			client := &Client{
				redfish: NewRedfish(host, &config.AuthConfig{Scheme: "https"}),
				vendor:  tc.vendor,
				version: 9,
			}
			client.path.ManagerEvent = entries

			id, ok := client.LatestManagerLogId()
			if id != tc.wantId || ok != tc.wantOk {
				t.Fatalf("got (%d, %v), want (%d, %v)", id, ok, tc.wantId, tc.wantOk)
			}
		})
	}
}
