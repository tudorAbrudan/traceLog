package main

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/agent"
	"github.com/tudorAbrudan/tracelog/internal/hub"
	"github.com/tudorAbrudan/tracelog/internal/hub/store"
	"github.com/tudorAbrudan/tracelog/internal/models"
	"github.com/tudorAbrudan/tracelog/internal/upgrade"
)

var version = "dev"

// applyDataDirEnv sets cfg.DataDir from TRACELOG_DATA_DIR (used by systemd and install.sh so CLI matches the service DB).
func applyDataDirEnv(cfg *models.Config) {
	if p := strings.TrimSpace(os.Getenv("TRACELOG_DATA_DIR")); p != "" {
		cfg.DataDir = p
	}
}

// applyURLPrefixEnv sets cfg.URLPathPrefix from TRACELOG_URL_PREFIX (e.g. /tracelog when served under that path).
func applyURLPrefixEnv(cfg *models.Config) {
	if p := strings.TrimSpace(os.Getenv("TRACELOG_URL_PREFIX")); p != "" {
		cfg.URLPathPrefix = p
	}
	cfg.URLPathPrefix = models.NormalizeURLPathPrefix(cfg.URLPathPrefix)
}

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
	applyDataDirEnv(cfg)
	cfg.Mode = "serve"
	cfg.Version = version
	parseServeFlags(cfg)
	applyURLPrefixEnv(cfg)

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
	applyDataDirEnv(cfg)
	cfg.Mode = "hub"
	cfg.Version = version
	parseHubFlags(cfg)
	applyURLPrefixEnv(cfg)

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

	cfg := models.DefaultConfig()
	applyDataDirEnv(cfg)

	s, err := store.New(cfg.DataDir)
	if err != nil {
		fatal("Failed to open database: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	switch os.Args[2] {
	case "create":
		username := "admin"
		if len(os.Args) > 3 {
			username = os.Args[3]
		}
		password, err := store.GeneratePassword()
		if err != nil {
			fatal("Failed to generate password: %v", err)
		}
		user, err := s.CreateUser(ctx, username, password)
		if err != nil {
			fatal("Failed to create user: %v", err)
		}
		fmt.Printf("User created successfully:\n")
		fmt.Printf("  Username: %s\n", user.Username)
		fmt.Printf("  Password: %s\n", password)
		fmt.Printf("\n  Save this password - it is shown only once!\n")

	case "reset-password":
		if len(os.Args) < 4 {
			fatal("Usage: tracelog user reset-password <username>")
		}
		username := os.Args[3]
		password, err := store.GeneratePassword()
		if err != nil {
			fatal("Failed to generate password: %v", err)
		}
		if err := s.UpdatePassword(ctx, username, password); err != nil {
			fatal("Failed to reset password: %v", err)
		}
		fmt.Printf("Password reset for user %q:\n", username)
		fmt.Printf("  New password: %s\n", password)
		fmt.Printf("\n  Save this password - it is shown only once!\n")

	case "list":
		users, err := s.ListUsers(ctx)
		if err != nil {
			fatal("Failed to list users: %v", err)
		}
		if len(users) == 0 {
			fmt.Println("No users found. Create one with: tracelog user create <username>")
			return
		}
		fmt.Printf("%-20s %-20s\n", "USERNAME", "CREATED")
		for _, u := range users {
			fmt.Printf("%-20s %-20s\n", u.Username, u.CreatedAt.Format("2006-01-02 15:04"))
		}

	default:
		fatal("Unknown user command: %s\nUsage: tracelog user <create|reset-password|list>", os.Args[2])
	}
}

func cmdStatus() {
	cfg := models.DefaultConfig()
	applyDataDirEnv(cfg)
	dbPath := cfg.DataDir + "/tracelog.db"

	fmt.Printf("TraceLog %s\n\n", version)

	if info, err := os.Stat(dbPath); err == nil {
		fmt.Printf("  Database: %s (%.1f MB)\n", dbPath, float64(info.Size())/(1024*1024))
	} else {
		fmt.Printf("  Database: %s (not found)\n", dbPath)
	}

	s, err := store.New(cfg.DataDir)
	if err != nil {
		fmt.Printf("  Database: cannot open (%v)\n", err)
		return
	}
	defer s.Close()

	ctx := context.Background()
	userCount, _ := s.UserCount(ctx)
	servers, _ := s.ListServers(ctx)

	fmt.Printf("  Users:    %d\n", userCount)
	fmt.Printf("  Servers:  %d\n", len(servers))
	for _, srv := range servers {
		fmt.Printf("    - %s (%s) [%s]\n", srv.Name, srv.Host, srv.Status)
	}

	retDays, _ := s.GetSetting(ctx, "retention_days")
	fmt.Printf("  Retention: %s days\n", retDays)
	fmt.Println()
}

func cmdBackup() {
	cfg := models.DefaultConfig()
	applyDataDirEnv(cfg)
	src := cfg.DataDir + "/tracelog.db"

	if _, err := os.Stat(src); os.IsNotExist(err) {
		fatal("Database not found at %s", src)
	}

	s, err := store.New(cfg.DataDir)
	if err != nil {
		fatal("Failed to open database: %v", err)
	}

	ts := time.Now().Format("20060102_150405")
	backupDir := cfg.DataDir + "/backups"
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		fatal("Cannot create backup dir: %v", err)
	}

	dst := backupDir + "/tracelog_" + ts + ".db"
	if err := s.Backup(context.Background(), dst); err != nil {
		s.Close()
		fatal("Backup failed: %v", err)
	}
	s.Close()

	info, _ := os.Stat(dst)
	sizeMB := float64(info.Size()) / (1024 * 1024)
	fmt.Printf("Backup created: %s (%.1f MB)\n", dst, sizeMB)
}

func cmdRestore() {
	if len(os.Args) < 3 {
		fatal("Usage: tracelog restore <backup-file>")
	}
	backupPath := os.Args[2]
	cfg := models.DefaultConfig()
	applyDataDirEnv(cfg)
	dst := cfg.DataDir + "/tracelog.db"

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		fatal("Backup file not found: %s", backupPath)
	}

	srcFile, err := os.Open(backupPath)
	if err != nil {
		fatal("Cannot open backup: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		fatal("Cannot write database: %v", err)
	}
	defer dstFile.Close()

	buf := make([]byte, 1024*1024)
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, werr := dstFile.Write(buf[:n]); werr != nil {
				fatal("Write failed: %v", werr)
			}
		}
		if err != nil {
			break
		}
	}

	fmt.Printf("Database restored from %s\n", backupPath)
}

func cmdInstall() {
	fmt.Printf("TraceLog %s - Interactive Installer\n\n", version)
	fmt.Println("Install (works with or without Go — release tarball or go install fallback):")
	fmt.Println("  curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/install.sh | bash")
	fmt.Println("Uninstall:")
	fmt.Println("  curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo bash")
	fmt.Println()
	fmt.Println("Or start the hub directly:")
	fmt.Println("  tracelog serve")
	fmt.Println("  Then open the browser to complete setup.")
}

func cmdUninstall() {
	if os.Getuid() != 0 {
		fmt.Println("Uninstall requires root. Run:")
		fmt.Println("  sudo tracelog uninstall")
		fmt.Println()
		fmt.Println("Or manually:")
		fmt.Println("  sudo systemctl stop tracelog && sudo systemctl disable tracelog")
		fmt.Println("  curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo bash")
		fmt.Println("  # or: sudo rm /etc/systemd/system/tracelog.service && sudo rm -f /usr/local/bin/tracelog && sudo rm -rf /etc/tracelog /var/lib/tracelog")
		return
	}

	fmt.Printf("TraceLog %s - Uninstaller\n\n", version)

	// Stop service
	fmt.Print("Stopping service... ")
	exec("systemctl", "stop", "tracelog")
	exec("systemctl", "disable", "tracelog")
	fmt.Println("done")

	// Remove systemd unit
	os.Remove("/etc/systemd/system/tracelog.service")
	exec("systemctl", "daemon-reload")
	fmt.Println("Removed systemd service")

	_ = os.RemoveAll("/etc/tracelog")
	fmt.Println("Removed /etc/tracelog (if present)")

	// Ask about data
	fmt.Print("\nDelete all data (/var/lib/tracelog)? [y/N] ")
	var answer string
	if _, err := fmt.Scanln(&answer); err != nil {
		answer = "" // EOF or unreadable input — treat as "no"
	}
	if answer == "y" || answer == "Y" {
		os.RemoveAll("/var/lib/tracelog")
		fmt.Println("Data deleted")
	} else {
		fmt.Println("Data kept at /var/lib/tracelog")
	}

	// Remove binary
	selfPath, _ := os.Executable()
	os.Remove(selfPath)
	fmt.Println("Binary removed")

	fmt.Println("\nTraceLog uninstalled successfully.")
}

func cmdUpgrade() {
	if err := upgrade.Run(version); err != nil {
		fmt.Fprintf(os.Stderr, "Upgrade failed: %v\n", err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Fallback — install script:")
		fmt.Fprintln(os.Stderr, "  curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/install.sh | bash")
		os.Exit(1)
	}
}

func exec(name string, args ...string) {
	cmd := osexec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v: %v\n", name, args, err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
