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
	var env AuthConfig

	getEnvString("CONFIG_ADDRESS", &c.Address)
	getEnvString("CONFIG_METRICS_PREFIX", &c.MetricsPrefix)
	getEnvString("CONFIG_DEFAULT_TARGET", &c.DefaultTarget)
	getEnvString("CONFIG_DEFAULT_USERNAME", &env.Username)
	getEnvString("CONFIG_DEFAULT_PASSWORD", &env.Password)
	getEnvString("CONFIG_DEFAULT_SCHEME", &env.Scheme)
	getEnvString("CONFIG_EVENTS_SEVERITY", &c.Event.Severity)
	getEnvString("CONFIG_EVENTS_MAXAGE", &c.Event.MaxAge)
	getEnvString("CONFIG_TLS_CERT_FILE", &c.TLS.CertFile)
	getEnvString("CONFIG_TLS_KEY_FILE", &c.TLS.KeyFile)

	getEnvUint("CONFIG_PORT", &c.Port)
	getEnvUint("CONFIG_TIMEOUT", &c.Timeout)
	getEnvUint("CONFIG_CONCURRENCY", &c.Concurrency)
	getEnvUint("CONFIG_DEFAULT_PORT", &env.Port)

	getEnvBool("CONFIG_DEFAULT_ENCODED", &env.Encoded)
	getEnvBool("CONFIG_DEFAULT_USE_BASIC_AUTH", &env.BasicAuth)
	getEnvBool("CONFIG_DEFAULT_ALLOW_LEGACY_RSA_KEX", &env.AllowLegacyRSAKex)
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
	getEnvBool("CONFIG_METRICS_MANAGER", &c.Collect.Manager)
	getEnvBool("CONFIG_METRICS_EXTRA", &c.Collect.Extra)

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
