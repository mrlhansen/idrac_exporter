package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/mrlhansen/idrac_exporter/internals/config"
	"github.com/mrlhansen/idrac_exporter/internals/redfish"
)

func collectMetrics(target string) (string, bool) {
	host, ok := config.Config.Hosts[target]
	if !ok {
		config.Config.Hosts[target] = new(config.HostConfig)
		host = config.Config.Hosts[target]
		host.Token = config.Config.Hosts["default"].Token
		host.Hostname = target
		host.Initialized = false
		host.Retries = 0
	}

	if !host.Initialized {
		ok = redfish.RedfishFindAllEndpoints(host)
		host.Retries++
		host.Initialized = ok || (host.Retries >= config.Config.Retries)
		host.Reachable = ok
	}

	if !host.Reachable {
		return "", false
	}

	redfish.MetricsClear(host)

	if config.CollectSystem {
		ok = redfish.RedfishSystem(host)
		if !ok {
			return "", false
		}
	}

	if config.CollectSensors {
		ok = redfish.RedfishSensors(host)
		if !ok {
			return "", false
		}
	}

	if config.CollectSEL {
		ok = redfish.RedfishSEL(host)
		if !ok {
			return "", false
		}
	}

	if config.CollectPower {
		ok = redfish.RedfishPower(host)
		if !ok {
			return "", false
		}
	}

	return redfish.MetricsGet(host), true
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	target, ok := args["target"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metrics, ok := collectMetrics(target[0])
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, metrics)
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "Config", "/etc/prometheus/idrac.yml", "path to idrac exporter configuration file")
	flag.Parse()
	config.ReadConfigFile(configFile)

	http.HandleFunc("/metrics", metricsHandler)
	bind := fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port)
	err := http.ListenAndServe(bind, nil)
	log.Fatal(err)
}
