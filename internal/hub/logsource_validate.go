package hub

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

// Keep in sync with internal/agent/collector/logs.go nginxLogRegex (access log lines).
var nginxAccessLine = regexp.MustCompile(
	`^(\S+)\s+-\s+\S+\s+\[([^\]]+)]\s+"(\S+)\s+(\S+)\s+\S+"\s+(\d+)\s+(\d+)\s+"[^"]*"\s+"([^"]*)"`,
)

// Apache combined or common log: host ident user [time] "METHOD path HTTP/x" status bytes
var apacheAccessLine = regexp.MustCompile(
	`^\S+\s+\S+\s+\S+\s+\[[^\]]+\]\s+"[A-Z]+\s+\S+\s+HTTP\/[0-9.]+\s*"\s+\d{3}\s+(-|\d+)`,
)

var allowedLogFormats = map[string]bool{
	"plain":  true,
	"nginx":  true,
	"apache": true,
}

func normalizeLogSource(ls *store.LogSourceRecord) {
	ls.Name = strings.TrimSpace(ls.Name)
	ls.Path = strings.TrimSpace(ls.Path)
	ls.Type = strings.TrimSpace(ls.Type)
	ls.Format = strings.TrimSpace(ls.Format)
	if ls.Format == "" {
		ls.Format = "plain"
	}
}

// validateLogSourceRecord checks fields before persisting. For type "file", when skipHubPathCheck is false,
// the path must exist on the hub machine. When skipHubPathCheck is true (log source assigned to a remote
// agent server_id), only syntax is validated — the file must exist on the agent host.
func validateLogSourceRecord(ls *store.LogSourceRecord, skipHubPathCheck bool) error {
	if ls.Name == "" {
		return errors.New("name is required")
	}
	if ls.Type == "" {
		return errors.New("type is required")
	}
	if ls.Type != "file" {
		return fmt.Errorf("unsupported log source type %q (only file is supported)", ls.Type)
	}
	if ls.Path == "" {
		return errors.New("path is required for file log sources")
	}
	if !allowedLogFormats[ls.Format] {
		return fmt.Errorf("unsupported format %q (allowed: plain, nginx, apache)", ls.Format)
	}
	if skipHubPathCheck {
		return nil
	}
	info, err := os.Stat(ls.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", ls.Path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("path must be a regular file: %s", ls.Path)
	}
	if err := validateFileMatchesFormat(ls.Path, ls.Format); err != nil {
		return err
	}
	return nil
}

// validateFileMatchesFormat samples non-empty lines and checks they match the parser for nginx/apache.
// Plain accepts any text. Empty or whitespace-only files skip content checks.
func validateFileMatchesFormat(path, format string) error {
	if format == "plain" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	const maxSample = 100
	var lines []string
	for sc.Scan() && len(lines) < maxSample {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read log file: %w", err)
	}
	if len(lines) == 0 {
		return nil
	}

	var match func(string) bool
	switch format {
	case "nginx":
		match = nginxAccessLine.MatchString
	case "apache":
		match = apacheAccessLine.MatchString
	default:
		return nil
	}

	matched := 0
	for _, line := range lines {
		if match(line) {
			matched++
		}
	}
	ratio := float64(matched) / float64(len(lines))
	minRatio := 0.55
	if len(lines) <= 2 {
		minRatio = 1.0
	} else if len(lines) <= 8 {
		minRatio = 0.65
	}
	if ratio < minRatio {
		return fmt.Errorf("file content does not match %s access log format (%d of %d sample lines matched); use plain for generic or error logs", format, matched, len(lines))
	}
	return nil
}
