package config

import (
	"os"

	"github.com/mrlhansen/idrac_exporter/internal/logging"
	"gopkg.in/yaml.v2"
)

var Config RootConfig = RootConfig{
	Hosts: make(map[string]*HostConfig),
}

func (c *RootConfig) GetHostCfg(target string) *HostConfig {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hostCfg, ok := c.Hosts[target]
	if !ok {
		hostCfg = &HostConfig{
			Hostname: target,
			Username: c.Hosts["default"].Username,
			Password: c.Hosts["default"].Password,
		}
		c.Hosts[target] = hostCfg
	}

	return hostCfg
}

func readConfigFile(filename string) {
	yamlFile, err := os.Open(filename)
	if err != nil {
		logging.Fatalf("failed to open configuration file: %s: %s", filename, err)
	}

	err = yaml.NewDecoder(yamlFile).Decode(&Config)
	yamlFile.Close()
	if err != nil {
		logging.Fatalf("invalid configuration file: %s: %s", filename, err.Error())
	}
}

func ReadConfig(filename string) {
	if len(filename) > 0 {
		readConfigFile(filename)
	}

	readConfigEnv()

	if Config.Address == "" {
		Config.Address = "0.0.0.0"
	}

	if Config.Port == 0 {
		Config.Port = 9348
	}

	if Config.Timeout == 0 {
		Config.Timeout = 10
	}

	if Config.Retries == 0 {
		Config.Retries = 1
	}

	if Config.MetricsPrefix == "" {
		Config.MetricsPrefix = "idrac"
	}

	if len(Config.Hosts) == 0 {
		logging.Fatalf("invalid configuration: empty section: hosts")
	}

	for k, v := range Config.Hosts {
		if v.Username == "" {
			logging.Fatalf("invalid configuration: missing username for host: %s", k)
		}
		if v.Password == "" {
			logging.Fatalf("invalid configuration: missing password for host: %s", k)
		}
		v.Hostname = k
	}
}
