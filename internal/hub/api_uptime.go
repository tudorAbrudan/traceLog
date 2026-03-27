package hub

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/hub/uptime"
)

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
