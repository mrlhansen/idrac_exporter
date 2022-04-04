package main

import (
	"os"
	"log"
	"io/ioutil"
	"encoding/base64"
	"gopkg.in/yaml.v2"
)

type HostConfig struct {
	Username string              `yaml:"username"`
	Password string              `yaml:"password"`
	Hostname string
	Token string
	Metrics string
	Active bool
	Valid bool
	SystemCollection string
}

type RootConfig struct {
	Address string               `yaml:"address"`
	Port int                     `yaml:"port"`
	Metrics []string             `yaml:"metrics"`
	Hosts map[string]*HostConfig `yaml:"hosts"`
}

var config RootConfig
var collectSystem bool = false
var collectSensors bool = false
var collectSEL bool = false

func validateMetrics(name string) bool {
	switch name {
		case "system":
			collectSystem = true
			return true
		case "sensors":
			collectSensors = true
			return true
		case "sel":
			collectSEL = true
			return true
	}
	return false
}

func parseError(s0, s1 string) {
	log.Printf("Error parsing configuration file: %s: %s", s0, s1)
	os.Exit(1)
}

func readConfigFile(fileName string) {
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Printf("Error reading configuration file: %s\n", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Printf("Error parsing configuration file: %s\n", err)
		os.Exit(1)
	}

	if len(config.Address) == 0 {
		config.Address = "0.0.0.0"
	}

	if config.Port == 0 {
		config.Port = 9348
	}

	if len(config.Metrics) == 0 {
		parseError("missing section", "metrics")
	}

	if len(config.Hosts) == 0 {
		parseError("missing section", "hosts")
	}

	for _, v := range config.Metrics {
		if !validateMetrics(v) {
			parseError("invalid metrics name", v)
		}
	}

	for k, v := range config.Hosts {
		if len(v.Username) == 0 {
			parseError("missing username for host", k)
		}
		if len(v.Password) == 0 {
			parseError("missing password for host", k)
		}

		data := []byte(v.Username + ":" + v.Password)
		v.Token = base64.StdEncoding.EncodeToString(data)
		v.Hostname = k
		v.Active = false
	}
}
