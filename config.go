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
		log.Printf("Error parsing configuration file: missing section: metrics")
		os.Exit(1)
	}

	if len(config.Hosts) == 0 {
		log.Printf("Error parsing configuration file: missing section: hosts")
		os.Exit(1)
	}

	for _, v := range config.Metrics {
		if !validateMetrics(v) {
			log.Printf("Error parsing configuration file: invalid metrics name: %s", v)
			os.Exit(1)
		}
	}

	for k, v := range config.Hosts {
		if len(v.Username) == 0 {
			log.Printf("Error parsing configuration file: missing username for host: %s", k)
		}
		if len(v.Password) == 0 {
			log.Printf("Error parsing configuration file: missing password for host: %s", k)
		}

		data := []byte(v.Username + ":" + v.Password)
		v.Token = base64.StdEncoding.EncodeToString(data)
		v.Hostname = k
		v.Active = false
	}
}
