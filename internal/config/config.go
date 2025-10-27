package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/mrlhansen/idrac_exporter/internal/log"
	"github.com/xhit/go-str2duration/v2"
	"gopkg.in/yaml.v3"
)

var Debug bool = false
var Config *RootConfig = nil

func GetHostConfig(target string) *HostConfig {
	Config.Mutex.Lock()
	defer Config.Mutex.Unlock()

	host, ok := Config.Hosts[target]
	if !ok {
		def, ok := Config.Hosts["default"]
		if !ok {
			log.Error("Could not find login information for host: %s", target)
			return nil
		}
		host = &HostConfig{
			Hostname: target,
			Scheme:   def.Scheme,
			Port:     def.Port,
			Username: def.Username,
			Password: def.Password,
		}
		Config.Hosts[target] = host
	}

	return host
}

func NewConfig() *RootConfig {
	return &RootConfig{
		Hosts: make(map[string]*HostConfig),
	}
}

func SetConfig(c *RootConfig) {
	Config = c
	if c.HttpsProxy != "" {
		os.Setenv("HTTPS_PROXY", c.HttpsProxy)
	}
}

func (c *RootConfig) FromFile(filename string) error {
	yamlFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open configuration file: %v", err)
	}

	err = yaml.NewDecoder(yamlFile).Decode(c)
	yamlFile.Close()
	if err != nil {
		return fmt.Errorf("parse configuration file: %v", err)
	}

	return nil
}

func (c *RootConfig) Validate() error {
	// main section
	if c.Address == "" {
		c.Address = "0.0.0.0"
	}

	if c.Port == 0 {
		c.Port = 9348
	}

	if c.Timeout == 0 {
		c.Timeout = 10
	}

	if c.MetricsPrefix == "" {
		c.MetricsPrefix = "idrac"
	}

	// hosts section
	if len(c.Hosts) == 0 {
		return fmt.Errorf("empty section: hosts")
	}

	for k, v := range c.Hosts {
		if v == nil {
			return fmt.Errorf("missing username and password for host: %s", k)
		}
		if v.Username == "" {
			return fmt.Errorf("missing username for host: %s", k)
		}
		if v.Password == "" {
			return fmt.Errorf("missing password for host: %s", k)
		}

		switch v.Scheme {
		case "":
			v.Scheme = "https"
		case "http", "https":
		default:
			return fmt.Errorf("invalid scheme for host: %s", k)
		}

		v.Hostname = k
	}

	// events section
	switch strings.ToLower(c.Event.Severity) {
	case "ok":
		c.Event.SeverityLevel = 0
	case "warning", "":
		c.Event.SeverityLevel = 1
	case "critical":
		c.Event.SeverityLevel = 2
	default:
		return fmt.Errorf("invalid value: %s", c.Event.Severity)
	}

	if c.Event.MaxAge == "" {
		c.Event.MaxAge = "7d"
	}

	t, err := str2duration.ParseDuration(c.Event.MaxAge)
	if err != nil {
		return fmt.Errorf("unable to parse duration: %v", err)
	}
	c.Event.MaxAgeSeconds = t.Seconds()

	// metrics
	if c.Collect.All {
		c.Collect.System = true
		c.Collect.Sensors = true
		c.Collect.Events = true
		c.Collect.Power = true
		c.Collect.Storage = true
		c.Collect.Memory = true
		c.Collect.Network = true
		c.Collect.Processors = true
		c.Collect.Extra = true
	}

	return nil
}
