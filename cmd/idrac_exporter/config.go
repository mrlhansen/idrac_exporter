package main

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mrlhansen/idrac_exporter/internal/collector"
	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/log"
)

type Debouncer struct {
	mu      sync.Mutex
	pending bool
	delay   time.Duration
	action  func()
}

func NewDebouncer(delay time.Duration, action func()) *Debouncer {
	return &Debouncer{
		delay:  delay,
		action: action,
	}
}

func (d *Debouncer) Trigger() {
	d.mu.Lock()
	if d.pending {
		d.mu.Unlock()
		return
	}
	d.pending = true
	d.mu.Unlock()

	time.AfterFunc(d.delay, func() {
		d.action()
		d.mu.Lock()
		d.pending = false
		d.mu.Unlock()
	})
}

func ReloadConfig(filename string, expand bool) {
	cfg := config.NewConfig()
	old := config.Config

	log.Info("Configuration reload was triggered")

	if len(filename) > 0 {
		err := cfg.FromFile(filename, expand)
		if err != nil {
			log.Error("Failed to %v", err)
			return
		}
	}

	cfg.FromEnvironment()
	err := cfg.Validate()
	if err != nil {
		log.Error("Invalid configuration: %v", err)
		return
	}
	// We only support configuration reload for the following sections: hosts, groups, metrics
	// Whenever credentials or anything connection related is changed, all related targets are being reset
	old.LCLog = cfg.LCLog

	old.Metrics = cfg.Metrics
	old.Mutex.Lock()
	defer old.Mutex.Unlock()

	for k, v := range cfg.Hosts {
		h, ok := old.Hosts[k]
		if ok {
			if !config.AuthIsEqual(h, v) {
				old.Hosts[k] = v
				collector.ResetAuth(h)
			} else {
				h.Metrics = v.Metrics
			}
		} else {
			old.Hosts[k] = v
		}
	}

	for k, v := range cfg.Groups {
		h, ok := old.Groups[k]
		if ok {
			if !config.AuthIsEqual(h, v) {
				old.Groups[k] = v
				collector.ResetAuth(h)
			} else {
				h.Metrics = v.Metrics
			}
		} else {
			old.Groups[k] = v
		}
	}

	log.Info("Configuration reload was successful")
}

func WatchConfig(filename string, expand bool) {
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

	// We use a debouncer with a delay because there might be multiple notifications
	// when the configuration file is changed. A delay of one second should be enough
	// to complete any operation on the filesystem before doing the reload.

	d := NewDebouncer(time.Second, func() {
		ReloadConfig(filename, expand)
		if len(watcher.WatchList()) == 0 {
			err := watcher.Add(filename)
			if err != nil {
				log.Error("Failed to watch configuration file after reload: %v", err)
				watcher.Close()
			}
		}
	})

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				d.Trigger()
			} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				watcher.Remove(event.Name)
				d.Trigger()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error("File watcher error: %v", err)
		}
	}
}

func LoadConfig(filename string, watch, expand bool) *config.RootConfig {
	cfg := config.NewConfig()

	if len(filename) > 0 {
		err := cfg.FromFile(filename, expand)
		if err != nil {
			log.Fatal("Failed to %v", err)
		}
		log.Info("Loaded configuration file: %s", filename)
	}

	cfg.FromEnvironment()
	err := cfg.Validate()
	if err != nil {
		log.Fatal("Invalid configuration: %v", err)
	}

	config.SetConfig(cfg)

	if watch && len(filename) > 0 {
		go WatchConfig(filename, expand)
	}

	return cfg
}
