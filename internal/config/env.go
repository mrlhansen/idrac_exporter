package config

import (
	"os"
	"strconv"
	"strings"
)

type envType interface {
	string | bool | uint
}

func getEnv[T envType](env string, val *T) {
	s := os.Getenv(env)
	if len(s) == 0 {
		return
	}

	switch any(*val).(type) {
	case string:
		*val = any(s).(T)
	case bool:
		switch strings.ToLower(s) {
		case "0", "false":
			*val = any(false).(T)
		default:
			*val = any(true).(T)
		}
	case uint:
		value, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return
		}
		*val = any(value).(T)
	}
}

func (c *RootConfig) FromEnvironment() {
	var env AuthConfig
	var m = &c.Metrics

	getEnv("CONFIG_ADDRESS", &c.Address)
	getEnv("CONFIG_PORT", &c.Port)
	getEnv("CONFIG_TIMEOUT", &c.Timeout)
	getEnv("CONFIG_CONCURRENCY", &c.Concurrency)

	getEnv("CONFIG_METRICS_PREFIX", &c.MetricsPrefix)
	getEnv("CONFIG_DEFAULT_TARGET", &c.DefaultTarget)

	getEnv("CONFIG_TLS_ENABLED", &c.TLS.Enabled)
	getEnv("CONFIG_TLS_CERT_FILE", &c.TLS.CertFile)
	getEnv("CONFIG_TLS_KEY_FILE", &c.TLS.KeyFile)

	getEnv("CONFIG_DEFAULT_ENCODED", &env.Encoded)
	getEnv("CONFIG_DEFAULT_USERNAME", &env.Username)
	getEnv("CONFIG_DEFAULT_PASSWORD", &env.Password)
	getEnv("CONFIG_DEFAULT_SCHEME", &env.Scheme)
	getEnv("CONFIG_DEFAULT_PORT", &env.Port)
	getEnv("CONFIG_DEFAULT_USE_BASIC_AUTH", &env.BasicAuth)
	getEnv("CONFIG_DEFAULT_ALLOW_LEGACY_RSA_KEX", &env.AllowLegacyRSAKex)

	getEnv("CONFIG_METRICS_ALL_ENABLED", &m.All.Enabled)
	getEnv("CONFIG_METRICS_SYSTEM_ENABLED", &m.System.Enabled)
	getEnv("CONFIG_METRICS_SENSORS_ENABLED", &m.Sensors.Enabled)
	getEnv("CONFIG_METRICS_SENSORS_UNIFIED", &m.Sensors.Unified)
	getEnv("CONFIG_METRICS_EVENTS_ENABLED", &m.Events.Enabled)
	getEnv("CONFIG_METRICS_EVENTS_SEVERITY", &m.Events.Severity)
	getEnv("CONFIG_METRICS_EVENTS_MAXAGE", &m.Events.MaxAge)
	getEnv("CONFIG_METRICS_POWER_ENABLED", &m.Power.Enabled)
	getEnv("CONFIG_METRICS_STORAGE_ENABLED", &m.Storage.Enabled)
	getEnv("CONFIG_METRICS_MEMORY_ENABLED", &m.Memory.Enabled)
	getEnv("CONFIG_METRICS_NETWORK_ENABLED", &m.Network.Enabled)
	getEnv("CONFIG_METRICS_PROCESSORS_ENABLED", &m.Processors.Enabled)
	getEnv("CONFIG_METRICS_MANAGER_ENABLED", &m.Manager.Enabled)
	getEnv("CONFIG_METRICS_EXTRA_ENABLED", &m.Extra.Enabled)

	def, ok := c.Hosts["default"]
	if !ok {
		def = &AuthConfig{}
	}

	if len(env.Username) > 0 {
		def.Username = env.Username
		ok = true
	}

	if len(env.Password) > 0 {
		def.Password = env.Password
		ok = true
	}

	if len(env.Scheme) > 0 {
		def.Scheme = env.Scheme
		ok = true
	}

	if env.Port > 0 {
		def.Port = env.Port
		ok = true
	}

	if env.Encoded {
		def.Encoded = true
		ok = true
	}

	if env.BasicAuth {
		def.BasicAuth = true
		ok = true
	}

	if env.AllowLegacyRSAKex {
		def.AllowLegacyRSAKex = true
		ok = true
	}

	if ok {
		c.Hosts["default"] = def
	}
}
