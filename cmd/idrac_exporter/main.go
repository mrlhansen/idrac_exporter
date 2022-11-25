package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/mrlhansen/idrac_exporter/internals/config"
	"github.com/mrlhansen/idrac_exporter/internals/promexporter"
)

func main() {
	var configFile string
	var metricsPrefix string

	flag.StringVar(&configFile, "config", "/etc/prometheus/idrac.yml", "Path to idrac exporter configuration file")
	flag.StringVar(&metricsPrefix, "prefix", "idrac_", "Prefix to prepend to metrics names")
	flag.Parse()

	config.ReadConfigFile(configFile)

	http.HandleFunc("/metrics", promexporter.MetricsHandler(metricsPrefix))
	bind := fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port)
	if err := http.ListenAndServe(bind, nil); err != nil {
		log.Fatal(err)
	}
}
