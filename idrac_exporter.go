package main

import (
	"fmt"
	"log"
	"flag"
	"net/http"
)

func collectMetrics(target string) (string, bool) {
	host, ok := config.Hosts[target]
	if !ok {
		config.Hosts[target] = new(HostConfig)
		host = config.Hosts[target]
		host.Token = config.Hosts["default"].Token
		host.Hostname = target
		host.Active = false
	}

	if !host.Active {
		_, ok = redfishGet(host, "Chassis")
		host.Active = true
		host.Valid = ok
	}

	if !host.Valid {
		return "", false
	}

	metricsClear(host)

	if collectSystem {
		redfishSystem(host)
	}
	if collectSensors {
		redfishSensors(host)
	}
	if collectSEL {
		redfishSEL(host)
	}

	return metricsGet(host), true
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
    flag.StringVar(&configFile, "config", "/etc/prometheus/idrac.yml", "path to idrac exporter configuration file")
	flag.Parse()
	readConfigFile(configFile)

	http.HandleFunc("/metrics", metricsHandler)
	bind := fmt.Sprintf("%s:%d", config.Address, config.Port)
	err := http.ListenAndServe(bind, nil)
	log.Fatal(err)
}
