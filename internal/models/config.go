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
	Version     string `yaml:"-"`
	// MetricsToken, if set, protects GET /metrics (Bearer token or ?token=). Also TRACELOG_METRICS_TOKEN.
	MetricsToken string `yaml:"metrics_token,omitempty"`
	// URLPathPrefix is the public path when behind a reverse proxy (e.g. /tracelog). Cookies use this Path.
	// Flag --url-prefix, env TRACELOG_URL_PREFIX, yaml url_path_prefix.
	URLPathPrefix string `yaml:"url_path_prefix,omitempty"`

	Collect CollectConfig `yaml:"collect"`
}

type CollectConfig struct {
	IntervalSeconds int         `yaml:"interval_seconds"`
	System          bool        `yaml:"system"`
	Docker          bool        `yaml:"docker"`
	Processes       bool        `yaml:"processes"`
	LogSources      []LogSource `yaml:"logs"`
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
			Processes:       true,
		},
	}
}
