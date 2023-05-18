package config

import (
	"encoding/base64"
	"os"
	"sync"
	"github.com/mrlhansen/idrac_exporter/internal/logging"
	"gopkg.in/yaml.v2"
)

type HostConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Hostname string
	Token    string
}

type RootConfig struct {
	mutex         sync.Mutex
	Address       string                 `yaml:"address"`
	Port          uint                   `yaml:"port"`
	MetricsPrefix string                 `yaml:"metrics_prefix"`
	Collect       struct {
		System  bool `yaml:"system"`
		Sensors bool `yaml:"sensors"`
		SEL     bool `yaml:"sel"`
		Power   bool `yaml:"power"`
		Storage bool `yaml:"storage"`
		Memory  bool `yaml:"memory"`
	} `yaml:"metrics"`
	Timeout       uint                   `yaml:"timeout"`
	Retries       uint                   `yaml:"retries"`
	Hosts         map[string]*HostConfig `yaml:"hosts"`
}

func (config *RootConfig) GetHostCfg(target string) *HostConfig {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	hostCfg, ok := config.Hosts[target]
	if !ok {
		hostCfg = &HostConfig{
			Hostname: target,
			Username: config.Hosts["default"].Username,
			Password: config.Hosts["default"].Password,
			Token:    config.Hosts["default"].Token,
		}
		config.Hosts[target] = hostCfg
	}

	return hostCfg
}

var Config RootConfig

func parseError(s0, s1 string) {
	logging.Fatalf("Error parsing configuration file: %s: %s", s0, s1)
}

func ReadConfigFile(fileName string) {
	yamlFile, err := os.Open(fileName)
	if err != nil {
		logging.Fatalf("Error opening configuration file %s: %s", fileName, err)
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

	if len(Config.Hosts) == 0 {
		parseError("missing section", "hosts")
	}

	if Config.MetricsPrefix == "" {
		Config.MetricsPrefix = "idrac"
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
