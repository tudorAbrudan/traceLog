package collector

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type LogCallback func(entry *models.LogEntry)

type LogCollector struct {
	sources []models.LogSource
	cb      LogCallback
}

func NewLogCollector(sources []models.LogSource, cb LogCallback) *LogCollector {
	return &LogCollector{sources: sources, cb: cb}
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
	f.Seek(0, 2)

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
		}
	case "plain":
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "error") || strings.Contains(lower, "fatal") || strings.Contains(lower, "crit"):
			entry.Level = "error"
		case strings.Contains(lower, "warn"):
			entry.Level = "warn"
		case strings.Contains(lower, "debug"):
			entry.Level = "debug"
		}
	}

	return entry
}
