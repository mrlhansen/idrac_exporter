package config

import (
	"encoding/base64"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

type HostConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Hostname string
	Token    string
}

type RootConfig struct {
	mu sync.Mutex

	Address string                 `yaml:"address"`
	Port    uint                   `yaml:"port"`
	Metrics []string               `yaml:"metrics"`
	Timeout uint                   `yaml:"timeout"`
	Retries uint                   `yaml:"retries"`
	Hosts   map[string]*HostConfig `yaml:"hosts"`
}

func (c *RootConfig) GetHostCfg(target string) *HostConfig {
	c.mu.Lock()
	defer c.mu.Unlock()

	hostCfg, ok := c.Hosts[target]
	if !ok {
		hostCfg = &HostConfig{
			Hostname: target,
			Username: c.Hosts["default"].Username,
			Password: c.Hosts["default"].Password,
			Token:    c.Hosts["default"].Token,
		}
		c.Hosts[target] = hostCfg
	}

	return hostCfg
}

var Config RootConfig
var CollectSystem bool
var CollectSensors bool
var CollectSEL bool
var CollectPower bool

func validateMetrics(name string) bool {
	switch name {
	case "system":
		CollectSystem = true
		return true
	case "sensors":
		CollectSensors = true
		return true
	case "power":
		CollectPower = true
		return true
	case "sel":
		CollectSEL = true
		return true
	}
	return false
}

func parseError(s0, s1 string) {
	log.Fatalf("Error parsing configuration file: %s: %s", s0, s1)
}

func ReadConfigFile(fileName string) {
	yamlFile, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Error opening configuration file %s: %s", fileName, err)
	}

	err = yaml.NewDecoder(yamlFile).Decode(&Config)
	yamlFile.Close()
	if err != nil {
		parseError(fileName, err.Error())
	}

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

	if len(Config.Metrics) == 0 {
		parseError("missing section", "metrics")
	}

	if len(Config.Hosts) == 0 {
		parseError("missing section", "hosts")
	}

	for _, v := range Config.Metrics {
		if !validateMetrics(v) {
			parseError("invalid metrics name", v)
		}
	}

	for k, v := range Config.Hosts {
		if v.Username == "" {
			parseError("missing username for host", k)
		}
		if v.Password == "" {
			parseError("missing password for host", k)
		}

		data := []byte(v.Username + ":" + v.Password)
		v.Token = base64.StdEncoding.EncodeToString(data)
		v.Hostname = k
	}
}
