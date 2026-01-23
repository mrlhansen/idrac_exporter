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

func (c *AuthConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("empty section")
	}

	if c.Username == "" {
		return fmt.Errorf("missing username")
	}

	if c.Password == "" {
		return fmt.Errorf("missing password")
	}

	switch c.Scheme {
	case "":
		c.Scheme = "https"
	case "http", "https":
	default:
		return fmt.Errorf("invalid scheme")
	}

	return nil
}

func GetAuthConfig(target, auth string) *AuthConfig {
	var host *AuthConfig
	var ok bool

	Config.Mutex.Lock()
	defer Config.Mutex.Unlock()

	if len(auth) > 0 {
		host, ok = Config.Auths[auth]
		if !ok {
			log.Error("Could not find login credentials: auth=%s", auth)
			return nil
		}
		return host
	}

	host, ok = Config.Hosts[target]
	if ok {
		return host
	}

	host, ok = Config.Hosts["default"]
	if !ok {
		log.Error("Could not find login credentials: host=%s", target)
		return nil
	}
	return host
}

func NewConfig() *RootConfig {
	return &RootConfig{
		Hosts: make(map[string]*AuthConfig),
		Auths: make(map[string]*AuthConfig),
	}
}

func SetConfig(c *RootConfig) {
	Config = c
	if c.HttpsProxy != "" {
		os.Setenv("HTTPS_PROXY", c.HttpsProxy)
	}
}

func (c *RootConfig) FromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("open configuration file: %v", err)
	}

	temp := os.ExpandEnv(string(data))
	data = []byte(temp)

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("parse configuration file: %v", err)
	}

	return nil
}

func (c *RootConfig) Validate() error {
	// root
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

	// hosts
	for k, v := range c.Hosts {
		err := v.Validate()
		if err != nil {
			return fmt.Errorf("host=%s: %v", k, err)
		}
	}

	// auths
	for k, v := range c.Auths {
		err := v.Validate()
		if err != nil {
			return fmt.Errorf("auth=%s: %v", k, err)
		}
	}

	// events
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
