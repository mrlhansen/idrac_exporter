package config

import (
	"sync"

	"github.com/goccy/go-yaml"
)

type AuthConfig struct {
	Username          string          `yaml:"username"`
	Password          string          `yaml:"password"`
	Scheme            string          `yaml:"scheme"`
	Port              uint            `yaml:"port"`
	Encoded           bool            `yaml:"encoded"`
	BasicAuth         bool            `yaml:"use_basic_auth"`
	AllowLegacyRSAKex bool            `yaml:"allow_legacy_rsa_kex"`
	RawMetrics        yaml.RawMessage `yaml:"metrics"`
	Metrics           MetricsConfig   `yaml:"-"`
}

type EventConfig struct {
	Enabled       bool    `yaml:"enabled"`
	Severity      string  `yaml:"severity"`
	MaxAge        string  `yaml:"maxage"`
	SeverityLevel int     `yaml:"-"`
	MaxAgeSeconds float64 `yaml:"-"`
}

type MetricsConfig struct {
	All struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"all"`
	System struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"system"`
	Sensors struct {
		Enabled bool `yaml:"enabled"`
		Unified bool `yaml:"unified"`
	} `yaml:"sensors"`
	Events EventConfig `yaml:"events"`
	Power  struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"power"`
	Storage struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"storage"`
	Memory struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"memory"`
	Network struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"network"`
	Processors struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"processors"`
	Manager struct {
		Enabled bool        `yaml:"enabled"`
		Events  EventConfig `yaml:"events"`
	} `yaml:"manager"`
	Extra struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"extra"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type RootConfig struct {
	Mutex         sync.Mutex
	Address       string                 `yaml:"address"`
	Port          uint                   `yaml:"port"`
	HttpsProxy    string                 `yaml:"https_proxy"`
	MetricsPrefix string                 `yaml:"metrics_prefix"`
	DefaultTarget string                 `yaml:"default_target"`
	Metrics       MetricsConfig          `yaml:"metrics"`
	TLS           TLSConfig              `yaml:"tls"`
	Timeout       uint                   `yaml:"timeout"`
	Concurrency   uint                   `yaml:"concurrency"`
	Hosts         map[string]*AuthConfig `yaml:"hosts"`
	Groups        map[string]*AuthConfig `yaml:"groups"`
}
