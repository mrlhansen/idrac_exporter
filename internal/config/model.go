package config

import "sync"

type HostConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Hostname string
}

type CollectConfig struct {
	System     bool `yaml:"system"`
	Sensors    bool `yaml:"sensors"`
	Events     bool `yaml:"events"`
	Power      bool `yaml:"power"`
	Storage    bool `yaml:"storage"`
	Memory     bool `yaml:"memory"`
	Network    bool `yaml:"network"`
	Processors bool `yaml:"processors"`
	OEM        bool `yaml:"oem"`
}

type EventConfig struct {
	Severity      string `yaml:"severity"`
	MaxAge        string `yaml:"maxage"`
	SeverityLevel int
	MaxAgeSeconds float64
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type RootConfig struct {
	mutex         sync.Mutex
	Address       string                 `yaml:"address"`
	Port          uint                   `yaml:"port"`
	HttpsProxy    string                 `yaml:"https_proxy"`
	MetricsPrefix string                 `yaml:"metrics_prefix"`
	Collect       CollectConfig          `yaml:"metrics"`
	Event         EventConfig            `yaml:"events"`
	TLS           TLSConfig              `yaml:"tls"`
	Timeout       uint                   `yaml:"timeout"`
	Hosts         map[string]*HostConfig `yaml:"hosts"`
}
