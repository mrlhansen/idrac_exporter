package config

import (
	"os"
	"strings"

	"github.com/mrlhansen/idrac_exporter/internal/log"
	"github.com/xhit/go-str2duration/v2"
	"gopkg.in/yaml.v3"
)

var Debug bool = false
var Config RootConfig = RootConfig{
	Hosts: make(map[string]*HostConfig),
}

func (c *RootConfig) GetHostCfg(target string) *HostConfig {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hostCfg, ok := c.Hosts[target]
	if !ok {
		def, ok := c.Hosts["default"]
		if !ok {
			log.Error("Could not find login information for host: %s", target)
			return nil
		}
		hostCfg = &HostConfig{
			Hostname: target,
			Username: def.Username,
			Password: def.Password,
		}
		c.Hosts[target] = hostCfg
	}

	return hostCfg
}

func readConfigFile(filename string) {
	yamlFile, err := os.Open(filename)
	if err != nil {
		log.Fatal("Failed to open configuration file: %s: %s", filename, err)
	}

	err = yaml.NewDecoder(yamlFile).Decode(&Config)
	yamlFile.Close()
	if err != nil {
		log.Fatal("Invalid configuration file: %s: %s", filename, err.Error())
	}

	log.Info("Configuration file: %s", filename)
}

func ReadConfig(filename string) {
	if len(filename) > 0 {
		readConfigFile(filename)
	}

	readConfigEnv()

	// main section
	if Config.Address == "" {
		Config.Address = "0.0.0.0"
	}

	if Config.Port == 0 {
		Config.Port = 9348
	}

	if Config.Timeout == 0 {
		Config.Timeout = 10
	}

	if Config.MetricsPrefix == "" {
		Config.MetricsPrefix = "idrac"
	}

	if Config.HttpsProxy != "" {
		os.Setenv("HTTPS_PROXY", Config.HttpsProxy)
	}

	// hosts section
	if len(Config.Hosts) == 0 {
		log.Fatal("Invalid configuration: empty section: hosts")
	}

	for k, v := range Config.Hosts {
		if v == nil {
			log.Fatal("Invalid configuration: missing username and password for host: %s", k)
		}
		if v.Username == "" {
			log.Fatal("Invalid configuration: missing username for host: %s", k)
		}
		if v.Password == "" {
			log.Fatal("Invalid configuration: missing password for host: %s", k)
		}
		v.Hostname = k
	}

	// events section
	switch strings.ToLower(Config.Event.Severity) {
	case "ok":
		Config.Event.SeverityLevel = 0
	case "warning", "":
		Config.Event.SeverityLevel = 1
	case "critical":
		Config.Event.SeverityLevel = 2
	default:
		log.Fatal("Invalid configuration: invalid value: %s", Config.Event.Severity)
	}

	if Config.Event.MaxAge == "" {
		Config.Event.MaxAge = "7d"
	}

	t, err := str2duration.ParseDuration(Config.Event.MaxAge)
	if err != nil {
		log.Fatal("Invalid configuration: unable to parse duration: %v", err)
	}
	Config.Event.MaxAgeSeconds = t.Seconds()
}
