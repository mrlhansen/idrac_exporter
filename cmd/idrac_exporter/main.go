package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/mrlhansen/idrac_exporter/internals/config"
	"github.com/mrlhansen/idrac_exporter/internals/logging"
	"github.com/mrlhansen/idrac_exporter/internals/promexporter"
)

var logger = logging.NewLogger().Sugar()

func main() {
	var verbose bool
	var configFile string
	var metricsPrefix string

	flag.BoolVar(&verbose, "verbose", false, "Set verbose logging")
	flag.StringVar(&configFile, "config", "/etc/prometheus/idrac.yml", "Path to idrac exporter configuration file")
	flag.StringVar(&metricsPrefix, "prefix", "idrac_", "Prefix to prepend to metrics names")
	flag.Parse()

	config.ReadConfigFile(configFile)

	if verbose {
		logging.SetVerboseLevel()
	}

	http.HandleFunc("/metrics", promexporter.MetricsHandler(metricsPrefix))
	bind := fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port)

	logger.Infof("Server listening on %s", bind)

	if err := http.ListenAndServe(bind, nil); err != nil {
		logger.Fatal(err)
	}
}
