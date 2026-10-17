package collector

import (
	"iter"
	"net/url"
	"strings"
	"time"

	"github.com/mrlhansen/idrac_exporter/internal/config"
)

type EventLog struct {
	config   *config.EventConfig
	response *EventLogResponse
}

type Event struct {
	Id        string
	Message   string
	Severity  string
	Timestamp time.Time
}

func (client *Client) QueryEventLog(path string, config *config.EventConfig) (*EventLog, bool) {
	if !config.Enabled || path == "" {
		return nil, true
	}

	if client.features.Filter {
		filter := url.Values{}

		switch config.SeverityLevel {
		case 1:
			filter.Set("$filter", "Severity eq 'Critical' or Severity eq 'Warning'")
		case 2:
			filter.Set("$filter", "Severity eq 'Critical'")
		}

		if s := filter.Encode(); len(s) > 0 {
			path = path + "?" + s
		}
	}

	resp := &EventLogResponse{}
	ok := client.redfish.Get(path, resp)
	if !ok {
		return nil, false
	}

	return &EventLog{
		config:   config,
		response: resp,
	}, true
}

func (e *EventLog) Items() iter.Seq[Event] {
	return func(yield func(Event) bool) {

		// Some machines will report filtering as a supported feature, but in reality the
		// returned data is not always properly filtered. For this reason we always filter
		// the data when looping through the entries.

		for _, v := range e.response.Members {
			t, err := time.Parse(time.RFC3339, v.Created)
			if err != nil {
				continue
			}

			d := time.Since(t)
			if d.Seconds() > e.config.MaxAgeSeconds {
				continue
			}

			severity := health2value(v.Severity)
			if severity < e.config.SeverityLevel {
				continue
			}

			v := Event{
				Id:        v.Id,
				Message:   strings.TrimSpace(v.Message),
				Severity:  v.Severity,
				Timestamp: t,
			}

			if !yield(v) {
				return
			}
		}
	}
}
