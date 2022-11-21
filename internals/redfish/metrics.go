package redfish

import (
	"fmt"
	"math"
	"strings"

	"github.com/mrlhansen/idrac_exporter/internals/config"
)

func MetricsClear(host *config.HostConfig) {
	host.Metrics = ""
}

func metricsAppend(host *config.HostConfig, name string, args map[string]string, value float64) {
	name = "idrac_" + name
	length := len(args)

	if length > 0 {
		name += "{"
		for k, v := range args {
			name += k + "=\"" + strings.TrimSpace(v) + "\""
			length--
			if length > 0 {
				name += ","
			}
		}
		name += "}"
	}

	if value < 0 {
		name += " NaN\n"
	} else {
		if value == math.Trunc(value) {
			name += fmt.Sprintf(" %1.0f\n", value)
		} else {
			name += fmt.Sprintf(" %.4g\n", value)
		}
	}

	host.Metrics += name
}

func MetricsGet(host *config.HostConfig) string {
	return host.Metrics
}
