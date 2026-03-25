package detect

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type Detection struct {
	DockerAvailable bool              `json:"docker_available"`
	DockerContainers int             `json:"docker_containers"`
	LogFiles        []models.LogSource `json:"log_files"`
	Processes       []string          `json:"processes"`
	WebServer       string            `json:"web_server"`
}

func Run() *Detection {
	d := &Detection{}
	d.detectDocker()
	d.detectLogFiles()
	d.detectWebServer()
	d.detectProcesses()
	return d
}

func (d *Detection) detectDocker() {
	if _, err := exec.LookPath("docker"); err != nil {
		return
	}
	out, err := exec.Command("docker", "ps", "-q").Output()
	if err != nil {
		return
	}
	d.DockerAvailable = true
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		d.DockerContainers = len(lines)
	}
}

var commonLogPaths = []struct {
	Path   string
	Name   string
	Format string
}{
	{"/var/log/nginx/access.log", "nginx-access", "nginx"},
	{"/var/log/nginx/error.log", "nginx-error", "plain"},
	{"/var/log/apache2/access.log", "apache-access", "apache"},
	{"/var/log/apache2/error.log", "apache-error", "plain"},
	{"/var/log/httpd/access_log", "httpd-access", "apache"},
	{"/var/log/httpd/error_log", "httpd-error", "plain"},
	{"/var/log/syslog", "syslog", "plain"},
	{"/var/log/messages", "messages", "plain"},
	{"/var/log/auth.log", "auth", "plain"},
	{"/var/log/kern.log", "kernel", "plain"},
}

func (d *Detection) detectLogFiles() {
	for _, lp := range commonLogPaths {
		if info, err := os.Stat(lp.Path); err == nil && !info.IsDir() {
			d.LogFiles = append(d.LogFiles, models.LogSource{
				Path:    lp.Path,
				Name:    lp.Name,
				Format:  lp.Format,
				Type:    "file",
				Enabled: true,
			})
		}
	}

	// Glob patterns for additional discovery
	globs := []struct {
		Pattern string
		Name    string
		Format  string
	}{
		{"/var/log/nginx/*.log", "nginx", "nginx"},
		{"/var/log/apache2/*.log", "apache", "apache"},
	}
	for _, g := range globs {
		matches, _ := filepath.Glob(g.Pattern)
		for _, m := range matches {
			if !d.hasLogPath(m) {
				d.LogFiles = append(d.LogFiles, models.LogSource{
					Path:    m,
					Name:    fmt.Sprintf("%s-%s", g.Name, filepath.Base(m)),
					Format:  g.Format,
					Type:    "file",
					Enabled: true,
				})
			}
		}
	}
}

func (d *Detection) hasLogPath(path string) bool {
	for _, l := range d.LogFiles {
		if l.Path == path {
			return true
		}
	}
	return false
}

func (d *Detection) detectWebServer() {
	for _, name := range []string{"nginx", "apache2", "httpd"} {
		if isProcessRunning(name) {
			d.WebServer = name
			return
		}
	}
}

var knownProcesses = []string{
	"nginx", "apache2", "httpd",
	"php-fpm", "php-cgi",
	"node", "pm2",
	"mysqld", "mariadbd", "postgres",
	"redis-server", "memcached",
	"mongod", "elasticsearch",
}

func (d *Detection) detectProcesses() {
	for _, name := range knownProcesses {
		if isProcessRunning(name) {
			d.Processes = append(d.Processes, name)
		}
	}
}

func isProcessRunning(name string) bool {
	out, err := exec.Command("pgrep", "-x", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func (d *Detection) Summary() string {
	var parts []string
	if d.DockerAvailable {
		parts = append(parts, fmt.Sprintf("Docker (%d containers)", d.DockerContainers))
	}
	if d.WebServer != "" {
		parts = append(parts, d.WebServer)
	}
	parts = append(parts, fmt.Sprintf("%d log files", len(d.LogFiles)))
	if len(d.Processes) > 0 {
		parts = append(parts, fmt.Sprintf("%d processes", len(d.Processes)))
	}
	return strings.Join(parts, ", ")
}
