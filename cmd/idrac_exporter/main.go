package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/log"
	"github.com/mrlhansen/idrac_exporter/internal/version"
)

var (
	flagVerbose bool
	flagDebug   bool
	flagConfig  string
	flagWatch   bool
	flagVersion bool
	flagExpand  bool
)

func main() {
	var err error

	flag.BoolVar(&flagVerbose, "verbose", false, "Enable more verbose logging")
	flag.BoolVar(&flagDebug, "debug", false, "Dump JSON response from Redfish requests (only for debugging purpose)")
	flag.StringVar(&flagConfig, "config", "/etc/prometheus/idrac.yml", "Path to the configuration file")
	flag.BoolVar(&flagWatch, "config-watch", false, "Watch the configuration file for changes and enable automatic reloading")
	flag.BoolVar(&flagExpand, "config-expand-env", false, "Expand environment variables in the configurtion file")
	flag.BoolVar(&flagVersion, "version", false, "Show version and exit")
	flag.Parse()

	if flagVersion {
		fmt.Printf("version: %s\n", version.Version)
		fmt.Printf("revision: %s\n", version.Revision)
		fmt.Printf("goversion: %s\n", runtime.Version())
		fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	log.Info("Build information: version=%s revision=%s", version.Version, version.Revision)
	cfg := LoadConfig(flagConfig, flagWatch, flagExpand)

	if flagDebug {
		config.Debug = true
		flagVerbose = true
	}

	if flagVerbose {
		log.SetLevel(log.LevelDebug)
	}

	http.HandleFunc("/discover", discoverHandler)
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/reload", reloadHandler)
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/", rootHandler)

	port := fmt.Sprintf("%d", cfg.Port)
	host := strings.Trim(cfg.Address, "[]")
	bind := net.JoinHostPort(host, port)
	log.Info("Server listening on %s (TLS: %v)", bind, cfg.TLS.Enabled)

	if cfg.TLS.Enabled {
		err = http.ListenAndServeTLS(bind, cfg.TLS.CertFile, cfg.TLS.KeyFile, nil)
	} else {
		err = http.ListenAndServe(bind, nil)
	}

	if err != nil {
		log.Fatal("%v", err)
	}
}
