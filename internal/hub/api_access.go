package hub

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/hub/ipmatch"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (h *Hub) accessStatsExcludeUAPatterns(ctx context.Context) []string {
	raw, err := h.store.GetSetting(ctx, "access_stats_exclude_ua_substrings")
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
	var out []string
	for _, s := range list {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// accessStatsExcludeHubPathPrefix returns cfg.URLPathPrefix normalized (install: --url-prefix / TRACELOG_URL_PREFIX) so analytics omit hub UI, not the monitored app.
func (h *Hub) accessStatsExcludeHubPathPrefix() string {
	return models.NormalizeURLPathPrefix(h.cfg.URLPathPrefix)
}

func (h *Hub) handleAccessLogStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "24h"
	}
	section := r.URL.Query().Get("section")
	d, err := parseRange(rangeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid range")
		return
	}
	topN := 30
	if s := r.URL.Query().Get("top_n"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			topN = n
		}
	}
	since := time.Now().Add(-d)
	uaExclude := h.accessStatsExcludeUAPatterns(r.Context())
	hubPathEx := h.accessStatsExcludeHubPathPrefix()
	stats, err := h.store.GetAccessLogStats(r.Context(), id, since, topN, uaExclude, hubPathEx, section)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get access stats")
		return
	}

	rules := h.loadAccessIPBlacklist(r.Context())
	if (section == "" || section == "overview") && len(rules) > 0 {
		const capIPGroups = 15000
		ipRows, err := h.store.GetAccessLogTopIPCounts(r.Context(), id, since, capIPGroups, uaExclude, hubPathEx)
		if err != nil {
			slog.Warn("access stats: top ip counts for blacklist", "error", err)
		} else {
			for _, row := range ipRows {
				if ipmatch.Match(row.IP, rules) {
					stats.BlacklistedHits += row.Count
				}
			}
			if len(ipRows) >= capIPGroups {
				stats.BlacklistHitsNote = fmt.Sprintf("Blacklist hits summed over the busiest %d distinct IPs (approximation).", capIPGroups)
			}
		}
		seen := make(map[string]bool)
		addBl := func(ip string) {
			if ip == "" || seen[ip] {
				return
			}
			if ipmatch.Match(ip, rules) {
				seen[ip] = true
				stats.BlacklistedInTop = append(stats.BlacklistedInTop, ip)
			}
		}
		for _, r := range stats.TopIPs {
			addBl(r.IP)
		}
		for _, r := range stats.BadRequestsByIP {
			addBl(r.IP)
		}
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *Hub) loadAccessIPBlacklist(ctx context.Context) []string {
	raw, err := h.store.GetSetting(ctx, "access_ip_blacklist")
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Warn("load access_ip_blacklist", "error", err)
		}
		return nil
	}
	return ipmatch.ParseJSONArray(raw)
}

func (h *Hub) handleAccessIPPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		raw, err := h.store.GetSetting(ctx, "access_ip_blacklist")
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, "Failed to read policy")
			return
		}
		ips := ipmatch.ParseJSONArray(raw)
		writeJSON(w, http.StatusOK, map[string][]string{"ips": ips})
	case http.MethodPut:
		var body struct {
			IPs []string `json:"ips"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body (expected {\"ips\":[\"1.2.3.4\",...]})")
			return
		}
		encoded, err := ipmatch.ToJSONArray(body.IPs)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid ip list")
			return
		}
		if err := h.store.SetSetting(ctx, "access_ip_blacklist", encoded); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save policy")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *Hub) handleAccessBadRequests(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "24h"
	}
	d, err := parseRange(rangeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid range")
		return
	}
	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	limit := 150
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}
	logs, err := h.store.QueryAccessBadRequests(r.Context(), id, time.Now().Add(-d), ip, limit, h.accessStatsExcludeHubPathPrefix())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query bad requests")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (h *Hub) handleAccessSlowRequests(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "24h"
	}
	d, err := parseRange(rangeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid range")
		return
	}
	minMs := 500.0
	if s := r.URL.Query().Get("min_ms"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			minMs = v
		}
	}
	limit := 150
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}
	uaExclude := h.accessStatsExcludeUAPatterns(r.Context())
	hubPathEx := h.accessStatsExcludeHubPathPrefix()
	logs, err := h.store.QueryAccessSlowRequests(r.Context(), id, time.Now().Add(-d), minMs, limit, uaExclude, hubPathEx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query slow requests")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (h *Hub) handleRecentAccessLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs, err := h.store.GetRecentAccessLogs(r.Context(), id, 200, h.accessStatsExcludeHubPathPrefix())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get access logs")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (h *Hub) handleAccessTimeline(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "24h"
	}
	timeline, err := h.store.GetAccessTimeline(r.Context(), serverID, rangeParam, h.accessStatsExcludeHubPathPrefix())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load timeline")
		return
	}
	writeJSON(w, http.StatusOK, timeline)
}
