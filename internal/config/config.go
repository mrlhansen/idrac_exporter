package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/mrlhansen/idrac_exporter/internal/log"
	"github.com/xhit/go-str2duration/v2"
)

var Debug bool = false
var Config *RootConfig = nil

func AuthIsEqual(a, b *AuthConfig) bool {
	if a.Username != b.Username {
		return false
	}
	if a.Password != b.Password {
		return false
	}
	if a.Scheme != b.Scheme {
		return false
	}
	if a.Port != b.Port {
		return false
	}
	if a.BasicAuth != b.BasicAuth {
		return false
	}
	if a.AllowLegacyRSAKex != b.AllowLegacyRSAKex {
		return false
	}
	return true
}

func (c *EventConfig) validate() error {
	switch strings.ToLower(c.Severity) {
	case "ok":
		c.SeverityLevel = 0
	case "warning", "":
		c.SeverityLevel = 1
	case "critical":
		c.SeverityLevel = 2
	default:
		return fmt.Errorf("invalid severity: %s", c.Severity)
	}

	if c.MaxAge == "" {
		c.MaxAge = "7d"
	}

	t, err := str2duration.ParseDuration(c.MaxAge)
	if err != nil {
		return fmt.Errorf("unable to parse duration: %v", err)
	}
	c.MaxAgeSeconds = t.Seconds()

	return nil
}

func (c *AuthConfig) validate(root *RootConfig) error {
	if c == nil {
		return fmt.Errorf("empty section")
	}

	if c.Encoded {
		u, err := base64.StdEncoding.DecodeString(c.Username)
		if err != nil {
			return fmt.Errorf("failed to decode username")
		}

		p, err := base64.StdEncoding.DecodeString(c.Password)
		if err != nil {
			return fmt.Errorf("failed to decode password")
		}

		c.Username = string(u)
		c.Password = string(p)
		c.Encoded = false
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

	c.Metrics = root.Metrics
	if c.RawMetrics != nil {
		err := yaml.Unmarshal(c.RawMetrics, &c.Metrics)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetAuthConfig(target, group string) *AuthConfig {
	var host *AuthConfig
	var ok bool

	Config.Mutex.Lock()
	defer Config.Mutex.Unlock()

	if len(group) > 0 {
		host, ok = Config.Groups[group]
		if !ok {
			log.Error("Could not find login credentials: group=%s", group)
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
		Hosts:  make(map[string]*AuthConfig),
		Groups: make(map[string]*AuthConfig),
	}
}

func SetConfig(c *RootConfig) {
	Config = c
	if c.HttpsProxy != "" {
		os.Setenv("HTTPS_PROXY", c.HttpsProxy)
	}
}

func (c *RootConfig) FromFile(filename string, expand bool) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("open configuration file: %v", err)
	}

	if expand {
		temp := os.ExpandEnv(string(data))
		data = []byte(temp)
	}

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

	if c.Concurrency == 0 {
		c.Concurrency = 10
	}

	if c.MetricsPrefix == "" {
		c.MetricsPrefix = "idrac"
	}

	// events
	err := c.Metrics.Events.validate()
	if err != nil {
		return err
	}

	err = c.Metrics.Manager.Events.validate()
	if err != nil {
		return err
	}
	err = c.LCLog.validate()
	if err != nil {
		return err
	}

	// metrics
	if c.Metrics.All.Enabled {
		c.Metrics.System.Enabled = true
		c.Metrics.Sensors.Enabled = true
		c.Metrics.Events.Enabled = true
		c.Metrics.Power.Enabled = true
		c.Metrics.Storage.Enabled = true
		c.Metrics.Memory.Enabled = true
		c.Metrics.Network.Enabled = true
		c.Metrics.Processors.Enabled = true
		c.Metrics.Manager.Enabled = true
		c.Metrics.Extra.Enabled = true
	}

	// hosts
	for k, v := range c.Hosts {
		err := v.validate(c)
		if err != nil {
			return fmt.Errorf("host=%s: %v", k, err)
		}
	}

	// groups
	for k, v := range c.Groups {
		err := v.validate(c)
		if err != nil {
			return fmt.Errorf("group=%s: %v", k, err)
		}
	}
	return nil
}

func validateEventConfig(event *EventConfig) error {
	switch strings.ToLower(event.Severity) {
	case "ok":
		event.SeverityLevel = 0
	case "warning", "":
		event.SeverityLevel = 1
	case "critical":
		event.SeverityLevel = 2
	default:
		return fmt.Errorf("invalid value: %s", event.Severity)
	}

	if event.MaxAge == "" {
		event.MaxAge = "7d"
	}

	duration, err := str2duration.ParseDuration(event.MaxAge)
	if err != nil {
		return fmt.Errorf("unable to parse duration: %v", err)
	}
	event.MaxAgeSeconds = duration.Seconds()

	return nil
}
