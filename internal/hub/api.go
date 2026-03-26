package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/agent/detect"
	"github.com/tudorAbrudan/tracelog/internal/hub/alerts"
	"github.com/tudorAbrudan/tracelog/internal/hub/dockerlogs"
	"github.com/tudorAbrudan/tracelog/internal/hub/notify"
	"github.com/tudorAbrudan/tracelog/internal/hub/store"
	"github.com/tudorAbrudan/tracelog/internal/hub/uptime"
)

// --- Health ---

func (h *Hub) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCount, _ := h.store.UserCount(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"version":    h.cfg.Version,
		"uptime":     time.Now().Unix(),
		"setup_done": userCount > 0,
	})
}

// --- Servers ---

func (h *Hub) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.store.ListServers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list servers: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

func (h *Hub) handleGetServer(w http.ResponseWriter, r *http.Request) {
	server, err := h.store.GetServer(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	writeJSON(w, http.StatusOK, server)
}

func (h *Hub) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Host string `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	server, err := h.store.CreateServer(r.Context(), req.Name, req.Host)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create server: %v", err)
		return
	}
	writeJSON(w, http.StatusCreated, server)
}

func (h *Hub) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteServer(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete server")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Metrics ---

func (h *Hub) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "1h"
	}
	duration, err := parseRange(rangeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid range: %s", rangeStr)
		return
	}
	metrics, err := h.store.QueryMetrics(r.Context(), id, time.Now().Add(-duration), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query metrics: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *Hub) handleGetDockerMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "1h"
	}
	duration, err := parseRange(rangeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid range")
		return
	}
	metrics, err := h.store.GetDockerMetrics(r.Context(), id, time.Now().Add(-duration))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query docker metrics")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *Hub) handleDockerContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	srv, err := h.store.GetServer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	container := r.URL.Query().Get("container")
	tail := 500
	if t := r.URL.Query().Get("tail"); t != "" {
		if n, err := strconv.Atoi(t); err == nil {
			tail = n
		}
	}

	var out string
	if srv.Host == "localhost" {
		out, err = dockerlogs.Fetch(r.Context(), container, tail)
	} else {
		out, err = h.requestDockerLogsFromAgent(r.Context(), id, container, tail)
	}
	if err != nil {
		if errors.Is(err, ErrAgentNotConnected) {
			writeError(w, http.StatusServiceUnavailable, "%v", err)
			return
		}
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"logs": out})
}

// --- Processes ---

func (h *Hub) handleGetProcesses(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	latest := r.URL.Query().Get("latest")
	if latest == "true" {
		procs, err := h.store.GetLatestProcesses(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to query processes")
			return
		}
		writeJSON(w, http.StatusOK, procs)
		return
	}

	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		rangeStr = "1h"
	}
	duration, err := parseRange(rangeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid range")
		return
	}
	procs, err := h.store.GetProcessMetrics(r.Context(), id, time.Now().Add(-duration))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query processes")
		return
	}
	writeJSON(w, http.StatusOK, procs)
}

// --- Logs ---

func (h *Hub) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id required")
		return
	}
	opts := store.LogQueryOpts{
		Source: r.URL.Query().Get("source"),
		Level:  r.URL.Query().Get("level"),
		Search: r.URL.Query().Get("search"),
		Limit:  500,
	}
	if rangeStr := r.URL.Query().Get("range"); rangeStr != "" {
		if d, err := parseRange(rangeStr); err == nil {
			opts.Since = time.Now().Add(-d)
		}
	}
	logs, err := h.store.QueryLogs(r.Context(), serverID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query logs")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// --- Access Logs / HTTP Analytics ---

func (h *Hub) handleAccessLogStats(w http.ResponseWriter, r *http.Request) {
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
	stats, err := h.store.GetAccessLogStats(r.Context(), id, time.Now().Add(-d))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get access stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Hub) handleRecentAccessLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs, err := h.store.GetRecentAccessLogs(r.Context(), id, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get access logs")
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// --- Log Sources ---

func (h *Hub) handleListLogSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.store.ListLogSources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list log sources")
		return
	}
	writeJSON(w, http.StatusOK, sources)
}

func (h *Hub) handleCreateLogSource(w http.ResponseWriter, r *http.Request) {
	var ls store.LogSourceRecord
	if err := json.NewDecoder(r.Body).Decode(&ls); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if ls.Name == "" || ls.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if err := h.store.CreateLogSource(r.Context(), &ls); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create log source: %v", err)
		return
	}
	writeJSON(w, http.StatusCreated, ls)
}

func (h *Hub) handleDeleteLogSource(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteLogSource(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete log source")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Settings ---

func (h *Hub) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetAllSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *Hub) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	allowedKeys := map[string]bool{"retention_days": true, "collection_interval": true}
	for k, v := range settings {
		if !allowedKeys[k] {
			continue
		}
		if err := h.store.SetSetting(r.Context(), k, v); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save setting %s", k)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Uptime Checks ---

func (h *Hub) handleListUptimeChecks(w http.ResponseWriter, r *http.Request) {
	checks, err := h.store.ListUptimeChecks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list uptime checks")
		return
	}
	writeJSON(w, http.StatusOK, checks)
}

func (h *Hub) handleCreateUptimeCheck(w http.ResponseWriter, r *http.Request) {
	var c uptime.Check
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if c.Name == "" || c.URL == "" {
		writeError(w, http.StatusBadRequest, "name and url are required")
		return
	}
	if err := h.store.CreateUptimeCheck(r.Context(), &c); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create uptime check: %v", err)
		return
	}
	h.uptime.AddCheck(&c)
	writeJSON(w, http.StatusCreated, c)
}

func (h *Hub) handleDeleteUptimeCheck(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.uptime.RemoveCheck(id)
	if err := h.store.DeleteUptimeCheck(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete uptime check")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Hub) handleGetUptimeResults(w http.ResponseWriter, r *http.Request) {
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
	results, err := h.store.GetUptimeResults(r.Context(), id, time.Now().Add(-d))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get uptime results")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// --- Alert Rules ---

func (h *Hub) handleListAlertRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.store.ListAlertRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list alert rules")
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *Hub) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule alerts.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if rule.Metric == "" || rule.Operator == "" {
		writeError(w, http.StatusBadRequest, "metric and operator are required")
		return
	}
	if err := h.store.CreateAlertRule(r.Context(), &rule); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create alert rule: %v", err)
		return
	}
	h.alerts.AddRule(&rule)
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Hub) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.alerts.RemoveRule(id)
	if err := h.store.DeleteAlertRule(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete alert rule")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Notification Channels ---

func (h *Hub) handleListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.store.ListNotificationChannels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list notification channels")
		return
	}
	writeJSON(w, http.StatusOK, channels)
}

func (h *Hub) handleCreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	var ch notify.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if ch.Name == "" || ch.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if err := h.store.CreateNotificationChannel(r.Context(), &ch); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create notification channel: %v", err)
		return
	}
	h.notify.AddChannel(&ch)
	writeJSON(w, http.StatusCreated, ch)
}

func (h *Hub) handleDeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.notify.RemoveChannel(id)
	if err := h.store.DeleteNotificationChannel(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete notification channel")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Hub) handleTestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := h.notify.Send(r.Context(), id, "TraceLog Test", "This is a test notification from TraceLog.")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Test failed: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// --- Detection ---

func (h *Hub) handleDetect(w http.ResponseWriter, r *http.Request) {
	d := detect.Run()
	writeJSON(w, http.StatusOK, d)
}

// --- Dashboard ---

func (h *Hub) handleDashboard(w http.ResponseWriter, r *http.Request) {
	spaHandler().ServeHTTP(w, r)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseRange(s string) (time.Duration, error) {
	switch s {
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}
