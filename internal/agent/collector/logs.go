package collector

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type LogCallback func(entry *models.LogEntry)
type AccessLogCallback func(entry *models.AccessLogEntry)

type LogCollector struct {
	sources    []models.LogSource
	cb         LogCallback
	accessCb   AccessLogCallback
}

func NewLogCollector(sources []models.LogSource, cb LogCallback, accessCb AccessLogCallback) *LogCollector {
	return &LogCollector{sources: sources, cb: cb, accessCb: accessCb}
}

func (lc *LogCollector) Start(ctx context.Context) {
	for _, src := range lc.sources {
		if !src.Enabled || src.Type != "file" {
			continue
		}
		go lc.tailFile(ctx, src)
	}
}

func (lc *LogCollector) tailFile(ctx context.Context, src models.LogSource) {
	slog.Info("Tailing log file", "path", src.Path, "name", src.Name)

	f, err := os.Open(src.Path)
	if err != nil {
		slog.Error("Failed to open log file", "path", src.Path, "error", err)
		return
	}
	defer f.Close()

	// Seek to end of file
	if _, err := f.Seek(0, 2); err != nil {
		slog.Error("Seek log file", "path", src.Path, "error", err)
		return
	}

	scanner := bufio.NewScanner(f)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				entry := parseLine(src, line)
				if entry != nil {
					lc.cb(entry)
				}
				if src.Format == "nginx" && lc.accessCb != nil {
					if alog := parseNginxAccessLog(src, line); alog != nil {
						lc.accessCb(alog)
					}
				}
			} else {
				time.Sleep(250 * time.Millisecond)
				// Reset scanner after EOF
				scanner = bufio.NewScanner(f)
			}
		}
	}
}

var nginxLogRegex = regexp.MustCompile(
	`^(\S+)\s+-\s+\S+\s+\[([^\]]+)]\s+"(\S+)\s+(\S+)\s+\S+"\s+(\d+)\s+(\d+)\s+"[^"]*"\s+"([^"]*)"`,
)

func parseNginxAccessLog(src models.LogSource, line string) *models.AccessLogEntry {
	matches := nginxLogRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	statusCode, _ := strconv.Atoi(matches[5])
	bytesSent, _ := strconv.ParseUint(matches[6], 10, 64)
	return &models.AccessLogEntry{
		Ts:         time.Now().UTC(),
		Method:     matches[3],
		Path:       matches[4],
		StatusCode: statusCode,
		IP:         matches[1],
		UserAgent:  matches[7],
		BytesSent:  bytesSent,
	}
}

// applyPlainSeverity sets level from common keywords (plain, apache, docker-style text, etc.).
func applyPlainSeverity(line string, entry *models.LogEntry) {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "panic"):
		entry.Level = "critical"
	case strings.Contains(lower, "fatal") || strings.Contains(lower, "critical") ||
		strings.Contains(lower, "emerg") || strings.Contains(lower, "alert"):
		entry.Level = "critical"
	case strings.Contains(lower, "error"):
		entry.Level = "error"
	case strings.Contains(lower, "warn"):
		entry.Level = "warn"
	case strings.Contains(lower, "debug"):
		entry.Level = "debug"
	}
}

func parseLine(src models.LogSource, line string) *models.LogEntry {
	if strings.TrimSpace(line) == "" {
		return nil
	}

	entry := &models.LogEntry{
		Ts:      time.Now().UTC(),
		Source:  src.Name,
		Message: line,
		Level:   "info",
	}

	switch src.Format {
	case "nginx":
		matches := nginxLogRegex.FindStringSubmatch(line)
		if matches != nil {
			entry.Metadata = fmt.Sprintf(`{"ip":"%s","method":"%s","path":"%s","status":"%s","bytes":"%s","ua":"%s"}`,
				matches[1], matches[3], matches[4], matches[5], matches[6], matches[7])
			if strings.HasPrefix(matches[5], "5") {
				entry.Level = "error"
			} else if strings.HasPrefix(matches[5], "4") {
				entry.Level = "warn"
			}
		} else {
			applyPlainSeverity(line, entry)
		}
	case "plain", "apache":
		applyPlainSeverity(line, entry)
	default:
		applyPlainSeverity(line, entry)
	}

	return entry
}
