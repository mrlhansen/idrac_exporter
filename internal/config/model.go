package config

import "sync"

type HostConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Hostname string
}

type CollectConfig struct {
	System  bool `yaml:"system"`
	Sensors bool `yaml:"sensors"`
	Events  bool `yaml:"events"`
	Power   bool `yaml:"power"`
	Storage bool `yaml:"storage"`
	Memory  bool `yaml:"memory"`
	Network bool `yaml:"network"`
}

type EventConfig struct {
	Severity      string `yaml:"severity"`
	MaxAge        string `yaml:"maxage"`
	SeverityLevel int
	MaxAgeSeconds float64
}

type RootConfig struct {
	mutex         sync.Mutex
	Address       string                 `yaml:"address"`
	Port          uint                   `yaml:"port"`
	MetricsPrefix string                 `yaml:"metrics_prefix"`
	Collect       CollectConfig          `yaml:"metrics"`
	Event         EventConfig            `yaml:"events"`
	Timeout       uint                   `yaml:"timeout"`
	Retries       uint                   `yaml:"retries"`
	Hosts         map[string]*HostConfig `yaml:"hosts"`
}
