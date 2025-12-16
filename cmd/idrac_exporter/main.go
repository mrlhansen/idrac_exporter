package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/log"
	"github.com/mrlhansen/idrac_exporter/internal/version"
)

func main() {
	var (
		verbose     bool
		debug       bool
		file        string
		watch       bool
		versionOnly bool
		err         error
	)

	flag.BoolVar(&verbose, "verbose", false, "Enable more verbose logging")
	flag.BoolVar(&debug, "debug", false, "Dump JSON response from Redfish requests (only for debugging purpose)")
	flag.StringVar(&file, "config", "/etc/prometheus/idrac.yml", "Path to the configuration file")
	flag.BoolVar(&watch, "config-watch", false, "Watch the configuration file for changes and enable automatic reloading")
	flag.BoolVar(&versionOnly, "version", false, "Show the version and exit gracefully")
	flag.Parse()

	if versionOnly {
		fmt.Println(fmt.Sprintf("version: %s", version.Version))
		fmt.Println(fmt.Sprintf("revision: %s", version.Revision))
		fmt.Println(fmt.Sprintf("goversion: %s", runtime.Version()))
		fmt.Println(fmt.Sprintf("os: %s", runtime.GOOS))
		fmt.Println(fmt.Sprintf("architecture: %s", runtime.GOARCH))
		os.Exit(0)
	}

	log.Info("Build information: version=%s revision=%s", version.Version, version.Revision)
	LoadConfig(file, watch)

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
