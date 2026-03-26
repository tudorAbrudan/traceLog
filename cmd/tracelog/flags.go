package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func parseServeFlags(cfg *models.Config) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "Port to listen on")
	fs.StringVar(&cfg.BindAddress, "bind", cfg.BindAddress, "Bind address")
	fs.StringVar(&cfg.DataDir, "data", cfg.DataDir, "Data directory")
	fs.StringVar(&cfg.MetricsToken, "metrics-token", cfg.MetricsToken, "If set, require Authorization: Bearer <token> or ?token= for /metrics (env TRACELOG_METRICS_TOKEN)")
	fs.StringVar(&cfg.URLPathPrefix, "url-prefix", cfg.URLPathPrefix, "Public URL path prefix behind a reverse proxy, e.g. /tracelog (env TRACELOG_URL_PREFIX)")
	fs.Usage = func() {
		fmt.Println("Usage: tracelog serve [options]")
		fmt.Println("\nStart hub + local agent in combined mode.")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
}

func parseHubFlags(cfg *models.Config) {
	fs := flag.NewFlagSet("hub", flag.ExitOnError)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "Port to listen on")
	fs.StringVar(&cfg.BindAddress, "bind", cfg.BindAddress, "Bind address")
	fs.StringVar(&cfg.DataDir, "data", cfg.DataDir, "Data directory")
	fs.StringVar(&cfg.MetricsToken, "metrics-token", cfg.MetricsToken, "If set, require Authorization: Bearer <token> or ?token= for /metrics")
	fs.StringVar(&cfg.URLPathPrefix, "url-prefix", cfg.URLPathPrefix, "Public URL path prefix behind a reverse proxy, e.g. /tracelog (env TRACELOG_URL_PREFIX)")
	fs.Usage = func() {
		fmt.Println("Usage: tracelog hub [options]")
		fmt.Println("\nStart hub only (for multi-server setup).")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
}

func parseAgentFlags(cfg *models.Config) {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	fs.StringVar(&cfg.HubURL, "hub", "", "Hub URL (required)")
	fs.StringVar(&cfg.APIKey, "key", "", "API key (required)")
	fs.Usage = func() {
		fmt.Println("Usage: tracelog agent --hub <url> --key <api-key>")
		fmt.Println("\nStart agent that reports to a remote hub.")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	if cfg.HubURL == "" || cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: --hub and --key are required for agent mode")
		fmt.Fprintln(os.Stderr, "")
		fs.Usage()
		os.Exit(1)
	}
}
