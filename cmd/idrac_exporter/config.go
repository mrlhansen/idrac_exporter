package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/mrlhansen/idrac_exporter/internal/collector"
	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/log"
)

func ReloadConfig(filename string) {
	cfg := config.NewConfig()
	old := config.Config

	log.Info("Configuration reload was triggered")

	if len(filename) > 0 {
		err := cfg.FromFile(filename)
		if err != nil {
			log.Error("Failed to %s", err)
			return
		}
	}

	cfg.FromEnvironment()
	err := cfg.Validate()
	if err != nil {
		log.Error("Invalid configuration: %v", err)
		return
	}

	old.Collect = cfg.Collect
	old.Event = cfg.Event

	old.Mutex.Lock()
	defer old.Mutex.Unlock()

	for k, v := range cfg.Hosts {
		h, ok := old.Hosts[k]
		if ok {
			if h.Username != v.Username || h.Password != v.Password || h.Scheme != v.Scheme {
				old.Hosts[k] = v
				collector.Reset(k)
			}
		} else {
			old.Hosts[k] = v
		}
	}

	log.Info("Configuration reload was successfull")
}

func WatchConfig(filename string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("Failed to start file watcher: %v", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(filename)
	if err != nil {
		log.Error("Failed to watch configuration file: %v", err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			reload := false
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				reload = true
			} else if event.Has(fsnotify.Remove) {
				watcher.Remove(event.Name)
				err := watcher.Add(filename)
				if err != nil {
					return
				}
				reload = true
			}
			if reload {
				ReloadConfig(filename)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error("File watcher error: %v", err)
		}
	}
}

func LoadConfig(filename string) {
	cfg := config.NewConfig()

	if len(filename) > 0 {
		err := cfg.FromFile(filename)
		if err != nil {
			log.Fatal("Failed to %s", err)
		}
		log.Info("Loaded configuration file: %s", filename)
	}

	cfg.FromEnvironment()
	err := cfg.Validate()
	if err != nil {
		log.Fatal("Invalid configuration: %v", err)
	}

	config.SetConfig(cfg)

	if len(filename) > 0 {
		go WatchConfig(filename)
	}
}
