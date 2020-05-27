package main

import (
	"math"
	"fmt"
)

func metricsClear(host *HostConfig) {
	host.Metrics = ""
}

func metricsAppend(host *HostConfig, name string, args map[string]string, value float64) {
	name = "idrac_" + name
	length := len(args)

	if length > 0 {
		name += "{"
		for k, v := range args {
			name += k + "=\"" + v + "\""
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

func metricsGet(host *HostConfig) string {
	return host.Metrics
}
