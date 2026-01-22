package config

import (
	"os"
	"strconv"
	"strings"
)

func getEnvString(env string, val *string) {
	value := os.Getenv(env)
	if len(value) == 0 {
		return
	}

	*val = value
}

func getEnvBool(env string, val *bool) {
	value := os.Getenv(env)
	if len(value) == 0 {
		return
	}

	switch strings.ToLower(value) {
	case "0", "false":
		*val = false
	default:
		*val = true
	}
}

func getEnvUint(env string, val *uint) {
	s := os.Getenv(env)
	if len(s) == 0 {
		return
	}

	value, err := strconv.ParseUint(s, 10, 0)
	if err == nil {
		*val = uint(value)
	}
}

func (c *RootConfig) FromEnvironment() {
	var (
		use_basic_auth bool
		username       string
		password       string
		scheme         string
		port           uint
	)

	getEnvString("CONFIG_ADDRESS", &c.Address)
	getEnvString("CONFIG_METRICS_PREFIX", &c.MetricsPrefix)
	getEnvString("CONFIG_DEFAULT_USERNAME", &username)
	getEnvString("CONFIG_DEFAULT_PASSWORD", &password)
	getEnvString("CONFIG_DEFAULT_SCHEME", &scheme)
	getEnvString("CONFIG_EVENTS_SEVERITY", &c.Event.Severity)
	getEnvString("CONFIG_EVENTS_MAXAGE", &c.Event.MaxAge)
	getEnvString("CONFIG_TLS_CERT_FILE", &c.TLS.CertFile)
	getEnvString("CONFIG_TLS_KEY_FILE", &c.TLS.KeyFile)

	getEnvUint("CONFIG_PORT", &c.Port)
	getEnvUint("CONFIG_TIMEOUT", &c.Timeout)
	getEnvUint("CONFIG_DEFAULT_PORT", &port)

	getEnvBool("CONFIG_DEFAULT_USE_BASIC_AUTH", &use_basic_auth)
	getEnvBool("CONFIG_TLS_ENABLED", &c.TLS.Enabled)
	getEnvBool("CONFIG_METRICS_ALL", &c.Collect.All)
	getEnvBool("CONFIG_METRICS_SYSTEM", &c.Collect.System)
	getEnvBool("CONFIG_METRICS_SENSORS", &c.Collect.Sensors)
	getEnvBool("CONFIG_METRICS_EVENTS", &c.Collect.Events)
	getEnvBool("CONFIG_METRICS_POWER", &c.Collect.Power)
	getEnvBool("CONFIG_METRICS_STORAGE", &c.Collect.Storage)
	getEnvBool("CONFIG_METRICS_MEMORY", &c.Collect.Memory)
	getEnvBool("CONFIG_METRICS_NETWORK", &c.Collect.Network)
	getEnvBool("CONFIG_METRICS_PROCESSORS", &c.Collect.Processors)
	getEnvBool("CONFIG_METRICS_EXTRA", &c.Collect.Extra)

	def, ok := c.Hosts["default"]
	if !ok {
		def = &AuthConfig{}
	}

	if len(username) > 0 {
		def.Username = username
		ok = true
	}

	if len(password) > 0 {
		def.Password = password
		ok = true
	}

	if len(scheme) > 0 {
		def.Scheme = scheme
		ok = true
	}

	if port > 0 {
		def.Port = port
		ok = true
	}

	if use_basic_auth {
		def.BasicAuth = true
		ok = true
	}

	if ok {
		c.Hosts["default"] = def
	}
}
