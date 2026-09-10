package collector

import (
	"net/url"
	"testing"
	"time"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

func TestNewEventLogEntryUsesIDRACTimestampAndLogType(t *testing.T) {
	desc := prometheus.NewDesc("idrac_events_log_entry", "Event log entry", []string{"id", "message", "severity", "log_type"}, nil)
	collector := &Collector{EventLogEntry: desc}
	created := time.Date(2026, time.April, 28, 9, 55, 48, 0, time.UTC)

	for _, logType := range []string{"system", "lifecycle"} {
		t.Run(logType, func(t *testing.T) {
			metrics := make(chan prometheus.Metric, 1)
			collector.NewEventLogEntry(metrics, "42", " test message ", "Warning", logType, created)

			metric := &dto.Metric{}
			if err := (<-metrics).Write(metric); err != nil {
				t.Fatalf("unable to encode metric: %v", err)
			}
			if got := metric.GetGauge().GetValue(); got != float64(created.Unix()) {
				t.Fatalf("unexpected metric value: got %v, want %v", got, created.Unix())
			}

			labels := make(map[string]string, len(metric.Label))
			for _, label := range metric.Label {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["log_type"] != logType {
				t.Fatalf("unexpected log_type label: got %q, want %q", labels["log_type"], logType)
			}
			if labels["message"] != "test message" {
				t.Fatalf("unexpected message label: got %q, want %q", labels["message"], "test message")
			}
		})
	}
}
