package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"

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
	LoadConfig(configFile)

	if debug {
		config.Debug = true
		verbose = true
	}

	if verbose {
		log.SetLevel(log.LevelDebug)
	}

	http.HandleFunc("/discover", discoverHandler)
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/", rootHandler)

	port := fmt.Sprintf("%d", config.Config.Port)
	host := strings.Trim(config.Config.Address, "[]")
	bind := net.JoinHostPort(host, port)
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
