package models

import (
	"os"
	"path/filepath"
)

type Config struct {
	Mode        string `yaml:"mode"`
	Port        int    `yaml:"port"`
	BindAddress string `yaml:"bind_address"`
	DataDir     string `yaml:"data_dir"`
	HubURL      string `yaml:"hub_url"`
	APIKey      string `yaml:"api_key"`

	Collect CollectConfig `yaml:"collect"`
}

type CollectConfig struct {
	IntervalSeconds int          `yaml:"interval_seconds"`
	System          bool         `yaml:"system"`
	Docker          bool         `yaml:"docker"`
	LogSources      []LogSource  `yaml:"logs"`
}

type LogSource struct {
	Path      string `yaml:"path,omitempty"`
	Name      string `yaml:"name"`
	Format    string `yaml:"format,omitempty"`
	Type      string `yaml:"type"`
	Container string `yaml:"container,omitempty"`
	Enabled   bool   `yaml:"enabled"`
}

func DefaultConfig() *Config {
	dataDir := "/var/lib/tracelog"
	if os.Getuid() != 0 {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".tracelog")
	}

	return &Config{
		Mode:        "serve",
		Port:        8090,
		BindAddress: "0.0.0.0",
		DataDir:     dataDir,
		Collect: CollectConfig{
			IntervalSeconds: 10,
			System:          true,
			Docker:          true,
		},
	}
}
