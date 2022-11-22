package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/mrlhansen/idrac_exporter/internals/config"
	"github.com/mrlhansen/idrac_exporter/internals/redfish"
)

func collectMetrics(target string) (string, error) {
	host, ok := config.Config.Hosts[target]
	if !ok {
		config.Config.Hosts[target] = new(config.HostConfig)
		host = config.Config.Hosts[target]
		host.Token = config.Config.Hosts["default"].Token
		host.Hostname = target
		host.Initialized = false
		host.Retries = 0
	}

	var err error
	if !host.Initialized {
		if err = redfish.FindAllEndpoints(host); err != nil {
			log.Printf("Error getting host %s endpoints: %v", host.Hostname, err)
		}
		host.Retries++
		host.Initialized = err == nil || (host.Retries >= config.Config.Retries)
		host.Reachable = err == nil
	}

	if !host.Reachable {
		return "", err
	}

	redfish.MetricsClear(host)

	if config.CollectSystem {
		if err = redfish.System(host); err != nil {
			log.Printf("Error getting host %s system state: %v", host.Hostname, err)
			return "", err
		}
	}

	if config.CollectSensors {
		if err = redfish.Sensors(host); err != nil {
			log.Printf("Error getting host %s sensors state: %v", host.Hostname, err)
			return "", err
		}
	}

	if config.CollectSEL {
		if err = redfish.IdracSel(host); err != nil {
			log.Printf("Error getting host %s iDrac SEL: %v", host.Hostname, err)
			return "", err
		}
	}

	if config.CollectPower {
		if err = redfish.Power(host); err != nil {
			log.Printf("Error getting host %s power state: %v", host.Hostname, err)
			return "", err
		}
	}

	return redfish.MetricsGet(host), nil
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	target, ok := args["target"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metrics, err := collectMetrics(target[0])
	if err != nil {
		log.Printf("Error collecting host %s metrics: %v", target[0], err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, metrics)
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "/etc/prometheus/idrac.yml", "path to idrac exporter configuration file")
	flag.Parse()
	config.ReadConfigFile(configFile)

	http.HandleFunc("/metrics", metricsHandler)
	bind := fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port)
	err := http.ListenAndServe(bind, nil)
	log.Fatal(err)
}
