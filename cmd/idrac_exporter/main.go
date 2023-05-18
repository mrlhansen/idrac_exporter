package main

import (
	"flag"
	"fmt"
	"net/http"
	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/logging"
	// "github.com/mrlhansen/idrac_exporter/internal/promexporter"
	"github.com/mrlhansen/idrac_exporter/internal/collector"
)

func main() {
	var verbose bool
	var configFile string

	flag.BoolVar(&verbose, "verbose", false, "Set verbose logging")
	flag.StringVar(&configFile, "config", "/etc/prometheus/idrac.yml", "Path to idrac exporter configuration file")
	flag.Parse()

	config.ReadConfigFile(configFile)

	if verbose {
		logging.SetVerbose(true)
	}

	http.HandleFunc("/metrics", collector.MetricsHandler)
	http.HandleFunc("/health", collector.HealthHandler)
	bind := fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port)

	logging.Infof("Server listening on %s", bind)

	err := http.ListenAndServe(bind, nil);
	if err != nil {
		logging.Fatal(err)
	}
}
