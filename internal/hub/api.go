package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

// Auth handlers (stubs)

func (h *Hub) handleLogin(w http.ResponseWriter, r *http.Request)  { writeJSON(w, http.StatusOK, map[string]string{"status": "TODO"}) }
func (h *Hub) handleLogout(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, map[string]string{"status": "TODO"}) }
func (h *Hub) handleMe(w http.ResponseWriter, r *http.Request)     { writeJSON(w, http.StatusOK, map[string]string{"status": "TODO"}) }

// Settings handlers (stubs)

func (h *Hub) handleGetSettings(w http.ResponseWriter, r *http.Request)    { writeJSON(w, http.StatusOK, map[string]string{"status": "TODO"}) }
func (h *Hub) handleUpdateSettings(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, map[string]string{"status": "TODO"}) }
func (h *Hub) handleGetLogSources(w http.ResponseWriter, r *http.Request)  { writeJSON(w, http.StatusOK, []any{}) }
func (h *Hub) handleCreateLogSource(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, map[string]string{"status": "TODO"}) }

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
