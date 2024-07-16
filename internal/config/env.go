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

func readConfigEnv() {
	var username string
	var password string

	getEnvString("CONFIG_ADDRESS", &Config.Address)
	getEnvString("CONFIG_METRICS_PREFIX", &Config.MetricsPrefix)
	getEnvString("CONFIG_DEFAULT_USERNAME", &username)
	getEnvString("CONFIG_DEFAULT_PASSWORD", &password)
	getEnvString("CONFIG_EVENTS_SEVERITY", &Config.Event.Severity)
	getEnvString("CONFIG_EVENTS_MAXAGE", &Config.Event.MaxAge)

	getEnvUint("CONFIG_PORT", &Config.Port)
	getEnvUint("CONFIG_TIMEOUT", &Config.Timeout)
	getEnvUint("CONFIG_RETRIES", &Config.Retries)

	getEnvBool("CONFIG_METRICS_SYSTEM", &Config.Collect.System)
	getEnvBool("CONFIG_METRICS_SENSORS", &Config.Collect.Sensors)
	getEnvBool("CONFIG_METRICS_EVENTS", &Config.Collect.Events)
	getEnvBool("CONFIG_METRICS_POWER", &Config.Collect.Power)
	getEnvBool("CONFIG_METRICS_STORAGE", &Config.Collect.Storage)
	getEnvBool("CONFIG_METRICS_MEMORY", &Config.Collect.Memory)
	getEnvBool("CONFIG_METRICS_NETWORK", &Config.Collect.Network)

	def, ok := Config.Hosts["default"]
	if !ok {
		def = &HostConfig{}
	}

	if len(username) > 0 {
		def.Username = username
		ok = true
	}

	if len(password) > 0 {
		def.Password = password
		ok = true
	}

	if ok {
		Config.Hosts["default"] = def
	}
}
