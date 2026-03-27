package hub

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/hub/dockerlogs"
)

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

func (h *Hub) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing server id")
		return
	}
	var body struct {
		Name  string `json:"name"`
		Host  string `json:"host"`
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	host := strings.TrimSpace(body.Host)
	notes := strings.TrimSpace(body.Notes)
	if len(notes) > 2000 {
		writeError(w, http.StatusBadRequest, "notes too long (max 2000 characters)")
		return
	}
	if _, err := h.store.GetServer(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	if err := h.store.UpdateServer(r.Context(), id, name, host, notes); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update server: %v", err)
		return
	}
	srv, err := h.store.GetServer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load server")
		return
	}
	writeJSON(w, http.StatusOK, srv)
}

func (h *Hub) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteServer(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete server")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Hub) handleSetServerAlertsMuted(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing server id")
		return
	}
	var body struct {
		Muted bool `json:"muted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.store.SetServerAlertsMuted(r.Context(), id, body.Muted); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update alerts muted state: %v", err)
		return
	}
	if body.Muted {
		h.mutedServers.Store(id, struct{}{})
	} else {
		h.mutedServers.Delete(id)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"muted": body.Muted})
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
