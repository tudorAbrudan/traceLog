package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/agent/detect"
	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

func (h *Hub) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": "dev",
		"uptime":  time.Now().Unix(),
	})
}

func (h *Hub) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.store.ListServers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list servers: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

func (h *Hub) handleGetServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	server, err := h.store.GetServer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	writeJSON(w, http.StatusOK, server)
}

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

	from := time.Now().Add(-duration)
	metrics, err := h.store.QueryMetrics(r.Context(), id, from, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query metrics: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
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

// Detection handler
func (h *Hub) handleDetect(w http.ResponseWriter, r *http.Request) {
	d := detect.Run()
	writeJSON(w, http.StatusOK, d)
}

// Dashboard handler - serves embedded Svelte SPA
func (h *Hub) handleDashboard(w http.ResponseWriter, r *http.Request) {
	spaHandler().ServeHTTP(w, r)
}

// Settings handlers

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
	allowedKeys := map[string]bool{
		"retention_days":      true,
		"collection_interval": true,
	}
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

func (h *Hub) handleGetLogSources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (h *Hub) handleCreateLogSource(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Logs handlers

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

	rangeStr := r.URL.Query().Get("range")
	if rangeStr != "" {
		d, err := parseRange(rangeStr)
		if err == nil {
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

// Docker metrics handlers

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

// Helpers

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
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
