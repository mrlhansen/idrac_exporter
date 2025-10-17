package config

import (
	"encoding/json"

	"github.com/mrlhansen/idrac_exporter/internal/log"
)

type DiscoverItem struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels,omitempty"`
}

func GetDiscover() string {
	var list []DiscoverItem
	for t := range Config.Hosts {
		if t == "default" {
			continue
		}
		list = append(list, DiscoverItem{
			Targets: []string{t},
		})
	}

	if len(list) == 0 {
		return "[]"
	}

	b, err := json.Marshal(list)
	if err != nil {
		log.Error("Failed to marshal json: %v", err)
		return "[]"
	}

	return string(b)
}
