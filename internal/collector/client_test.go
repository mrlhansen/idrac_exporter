package collector

import (
	"net/url"
	"testing"

	"github.com/mrlhansen/idrac_exporter/internal/config"
)

func TestEventLogFilterPath(t *testing.T) {
	tests := []struct {
		name     string
		severity int
		want     []string
	}{
		{name: "all severities", severity: 0, want: []string{""}},
		{name: "warning and critical", severity: 1, want: []string{"Severity eq 'Warning'", "Severity eq 'Critical'"}},
		{name: "critical", severity: 2, want: []string{"Severity eq 'Critical'"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths := eventLogFilterPaths("/redfish/v1/lclog/Entries", config.EventConfig{
				SeverityLevel: test.severity,
				MaxAgeSeconds: 24 * 60 * 60,
			})
			if len(paths) != len(test.want) {
				t.Fatalf("unexpected number of filter paths: got %d, want %d", len(paths), len(test.want))
			}
			for index, path := range paths {
				parsed, err := url.Parse(path)
				if err != nil {
					t.Fatalf("unable to parse filter path: %v", err)
				}
				if got := parsed.Query().Get("$filter"); got != test.want[index] {
					t.Fatalf("unexpected filter: got %q, want %q", got, test.want[index])
				}
			}
		})
	}
}
