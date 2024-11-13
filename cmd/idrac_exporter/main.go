package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/log"
	"github.com/mrlhansen/idrac_exporter/internal/version"
)

func main() {
	var verbose bool
	var debug bool
	var configFile string
	var err error

	flag.BoolVar(&verbose, "verbose", false, "Enable more verbose logging")
	flag.BoolVar(&debug, "debug", false, "Dump JSON response from Redfish requests (only for debugging purpose)")
	flag.StringVar(&configFile, "config", "/etc/prometheus/idrac.yml", "Path to idrac exporter configuration file")
	flag.Parse()

	log.Info("Build information: version=%s revision=%s", version.Version, version.Revision)
	config.ReadConfig(configFile)

	if debug {
		config.Debug = true
		verbose = true
	}

	if verbose {
		log.SetLevel(log.LevelDebug)
	}

	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/", rootHandler)

	bind := fmt.Sprintf("%s:%d", config.Config.Address, config.Config.Port)
	log.Info("Server listening on %s (TLS: %v)", bind, config.Config.TLS.Enabled)

	if config.Config.TLS.Enabled {
		err = http.ListenAndServeTLS(bind, config.Config.TLS.CertFile, config.Config.TLS.KeyFile, nil)
	} else {
		err = http.ListenAndServe(bind, nil)
	}

	if err != nil {
		log.Fatal("%v", err)
	}
}
