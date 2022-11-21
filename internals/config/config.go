package config

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type HostConfig struct {
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Hostname        string
	Token           string
	Metrics         string
	Initialized     bool
	Reachable       bool
	Retries         uint32
	SystemEndpoint  string
	ThermalEndpoint string
	PowerEndpoint   string
}

type RootConfig struct {
	Address string                 `yaml:"address"`
	Port    uint32                 `yaml:"port"`
	Metrics []string               `yaml:"metrics"`
	Timeout uint32                 `yaml:"timeout"`
	Retries uint32                 `yaml:"retries"`
	Hosts   map[string]*HostConfig `yaml:"hosts"`
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
	log.Printf("Error parsing configuration file: %s: %s", s0, s1)
	os.Exit(1)
}

func ReadConfigFile(fileName string) {
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Printf("Error reading configuration file: %s\n", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		log.Printf("Error parsing configuration file: %s\n", err)
		os.Exit(1)
	}

	if len(Config.Address) == 0 {
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
		if len(v.Username) == 0 {
			parseError("missing username for host", k)
		}
		if len(v.Password) == 0 {
			parseError("missing password for host", k)
		}

		data := []byte(v.Username + ":" + v.Password)
		v.Token = base64.StdEncoding.EncodeToString(data)
		v.Hostname = k
		v.Initialized = false
		v.Retries = 0
	}
}
