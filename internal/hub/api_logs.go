package hub

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

func (h *Hub) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server_id required")
		return
	}
	opts := store.LogQueryOpts{
		Source: r.URL.Query().Get("source"),
		Search: r.URL.Query().Get("search"),
		Limit:  500,
	}
	if lv := strings.TrimSpace(r.URL.Query().Get("level")); lv != "" {
		opts.Level = lv
	}
	if sm := strings.TrimSpace(r.URL.Query().Get("severity_min")); sm != "" && opts.Level == "" {
		opts.SeverityMin = sm
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

func (h *Hub) handlePurgeLogs(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ServerID string `json:"server_id"`
		Mode     string `json:"mode"` // "all" or "older_than"
		Range    string `json:"range,omitempty"`
		Source   string `json:"source,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if _, err := h.store.GetServer(r.Context(), body.ServerID); err != nil {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}

	var before time.Time
	switch body.Mode {
	case "all", "":
		before = time.Time{}
	case "older_than":
		d, err := parseRange(body.Range)
		if err != nil || d <= 0 {
			writeError(w, http.StatusBadRequest, "range is required for older_than (e.g. 24h, 7d, 30d)")
			return
		}
		before = time.Now().Add(-d)
	default:
		writeError(w, http.StatusBadRequest, "mode must be all or older_than")
		return
	}

	n, err := h.store.DeleteIngestedLogs(r.Context(), body.ServerID, strings.TrimSpace(body.Source), before)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to purge logs: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
}
