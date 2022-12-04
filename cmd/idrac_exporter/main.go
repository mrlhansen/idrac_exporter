package main

import (
	"flag"
	"fmt"
	"net/http"
	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/logging"
	"github.com/mrlhansen/idrac_exporter/internal/promexporter"
)

var logger = logging.NewLogger().Sugar()

func main() {
	var verbose bool
	var configFile string

	flag.BoolVar(&verbose, "verbose", false, "Set verbose logging")
	flag.StringVar(&configFile, "config", "/etc/prometheus/idrac.yml", "Path to idrac exporter configuration file")
	flag.Parse()

	config.ReadConfigFile(configFile)

	if verbose {
		logging.SetVerboseLevel()
	}

	http.HandleFunc("/metrics", promexporter.MetricsHandler)
	bind := fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port)

	logger.Infof("Server listening on %s", bind)

	err := http.ListenAndServe(bind, nil);
	if err != nil {
		logger.Fatal(err)
	}
}
