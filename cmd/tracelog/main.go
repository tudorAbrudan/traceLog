package main

import (
	"fmt"
	"os"

	"github.com/tudorAbrudan/tracelog/internal/agent"
	"github.com/tudorAbrudan/tracelog/internal/hub"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		cmdServe()
	case "hub":
		cmdHub()
	case "agent":
		cmdAgent()
	case "user":
		cmdUser()
	case "status":
		cmdStatus()
	case "backup":
		cmdBackup()
	case "restore":
		cmdRestore()
	case "install":
		cmdInstall()
	case "uninstall":
		cmdUninstall()
	case "upgrade":
		cmdUpgrade()
	case "version":
		fmt.Printf("tracelog %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`TraceLog %s - Server Monitoring Platform

Usage: tracelog <command> [options]

Commands:
  serve       Start hub + local agent (single-server mode)
  hub         Start hub only (multi-server mode)
  agent       Start agent only (connects to remote hub)

  user        Manage users (create, reset-password, list)
  status      Show service status, DB size, connections

  backup      Backup database
  restore     Restore database from backup

  install     Interactive installer
  uninstall   Remove TraceLog from this system
  upgrade     Self-update to latest version

  version     Print version
  help        Show this help

Run 'tracelog <command> --help' for details on a specific command.
`, version)
}

func cmdServe() {
	cfg := models.DefaultConfig()
	cfg.Mode = "serve"
	parseServeFlags(cfg)

	h, err := hub.New(cfg)
	if err != nil {
		fatal("Failed to start hub: %v", err)
	}

	a, err := agent.New(cfg, agent.WithLocalHub(h))
	if err != nil {
		fatal("Failed to start agent: %v", err)
	}

	fmt.Printf("TraceLog %s running in serve mode on %s:%d\n", version, cfg.BindAddress, cfg.Port)
	if err := runBoth(h, a); err != nil {
		fatal("%v", err)
	}
}

func cmdHub() {
	cfg := models.DefaultConfig()
	cfg.Mode = "hub"
	parseHubFlags(cfg)

	h, err := hub.New(cfg)
	if err != nil {
		fatal("Failed to start hub: %v", err)
	}

	fmt.Printf("TraceLog %s running as hub on %s:%d\n", version, cfg.BindAddress, cfg.Port)
	if err := h.Run(); err != nil {
		fatal("%v", err)
	}
}

func cmdAgent() {
	cfg := models.DefaultConfig()
	cfg.Mode = "agent"
	parseAgentFlags(cfg)

	a, err := agent.New(cfg, agent.WithRemoteHub(cfg.HubURL, cfg.APIKey))
	if err != nil {
		fatal("Failed to start agent: %v", err)
	}

	fmt.Printf("TraceLog %s running as agent, reporting to %s\n", version, cfg.HubURL)
	if err := a.Run(); err != nil {
		fatal("%v", err)
	}
}

func cmdUser() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: tracelog user <create|reset-password|list> [username]")
		os.Exit(1)
	}
	// TODO: implement user management
	fmt.Println("User management not yet implemented")
}

func cmdStatus()    { fmt.Println("Status not yet implemented") }
func cmdBackup()    { fmt.Println("Backup not yet implemented") }
func cmdRestore()   { fmt.Println("Restore not yet implemented") }
func cmdInstall()   { fmt.Println("Install wizard not yet implemented") }
func cmdUninstall() { fmt.Println("Uninstall not yet implemented") }
func cmdUpgrade()   { fmt.Println("Upgrade not yet implemented") }

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
