package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// logIngestRules limits which severities are stored per log source (see log_sources.ingest_levels).
type logIngestRules struct {
	byServer         map[string]map[string][]string // serverID -> source name -> levels
	wildcardBySource map[string][]string            // source name -> levels when log_sources.server_id is empty
}

func (h *Hub) reloadLogIngestRules(ctx context.Context) {
	sources, err := h.store.ListLogSources(ctx)
	if err != nil {
		slog.Warn("reload log ingest rules", "error", err)
		return
	}
	next := logIngestRules{
		byServer:         make(map[string]map[string][]string),
		wildcardBySource: make(map[string][]string),
	}
	for _, s := range sources {
		raw := strings.TrimSpace(s.IngestLevels)
		if raw == "" {
			continue
		}
		var levels []string
		if err := json.Unmarshal([]byte(raw), &levels); err != nil || len(levels) == 0 {
			continue
		}
		norm := normalizeIngestLevels(levels)
		if len(norm) == 0 {
			continue
		}
		if s.ServerID == "" {
			next.wildcardBySource[s.Name] = norm
		} else {
			if next.byServer[s.ServerID] == nil {
				next.byServer[s.ServerID] = make(map[string][]string)
			}
			next.byServer[s.ServerID][s.Name] = norm
		}
	}
	h.logIngestMu.Lock()
	h.logIngestRules = next
	h.logIngestMu.Unlock()
}

func normalizeIngestLevels(levels []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, l := range levels {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" || seen[l] {
			continue
		}
		if !allowedIngestLevel[l] {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}

var allowedIngestLevel = map[string]bool{
	"critical": true, "error": true, "warn": true, "info": true, "debug": true, "deprecated": true,
}

// ValidateLogSourceIngestList normalizes severity names for log_sources.ingest_levels. Empty input means “no filter”.
func ValidateLogSourceIngestList(levels []string) ([]string, error) {
	if len(levels) == 0 {
		return nil, nil
	}
	n := normalizeIngestLevels(levels)
	if len(n) == 0 {
		return nil, fmt.Errorf("ingest_levels: use one or more of critical, error, warn, info, debug, deprecated")
	}
	return n, nil
}

func (h *Hub) logIngestAllowed(serverID, source, level string) bool {
	h.logIngestMu.RLock()
	rules := h.logIngestRules
	h.logIngestMu.RUnlock()

	lv := strings.ToLower(strings.TrimSpace(level))
	if lv == "" {
		lv = "info"
	}
	var allowed []string
	if m := rules.byServer[serverID]; m != nil {
		allowed = m[source]
	}
	if len(allowed) == 0 {
		allowed = rules.wildcardBySource[source]
	}
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == lv {
			return true
		}
	}
	return false
}
