package hub

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tudorAbrudan/tracelog/internal/hub/store"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

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
	normalizeLogSource(&ls)
	ls.ServerID = strings.TrimSpace(ls.ServerID)
	localID, err := h.EnsureLocalServer(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to resolve local server: %v", err)
		return
	}
	remoteOnly := ls.ServerID != "" && ls.ServerID != localID
	if err := validateLogSourceRecord(&ls, remoteOnly); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	norm, err := ValidateLogSourceIngestList(ls.IngestLevelsList)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	ls.IngestLevelsList = norm
	if err := h.store.CreateLogSource(r.Context(), &ls); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create log source: %v", err)
		return
	}
	h.reloadLogIngestRules(r.Context())
	writeJSON(w, http.StatusCreated, ls)
}

func (h *Hub) handleUpdateLogSource(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	var body struct {
		IngestLevels *[]string `json:"ingest_levels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.IngestLevels == nil {
		writeError(w, http.StatusBadRequest, "ingest_levels is required (use [] to clear and store all severities)")
		return
	}
	norm, err := ValidateLogSourceIngestList(*body.IngestLevels)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := h.store.UpdateLogSourceIngestLevels(r.Context(), id, norm); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update log source")
		return
	}
	h.reloadLogIngestRules(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "ingest_levels": norm})
}

func (h *Hub) handleDeleteLogSource(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteLogSource(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete log source")
		return
	}
	h.reloadLogIngestRules(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAgentLogSources returns file log sources for the server matching X-API-Key (remote agent polling).
func (h *Hub) handleAgentLogSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	key := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if key == "" {
		writeError(w, http.StatusUnauthorized, "Missing X-API-Key")
		return
	}
	srv, err := h.store.GetServerByAPIKey(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid API key")
		return
	}
	sources, err := h.store.ListLogSourcesForAgentServer(r.Context(), srv.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list log sources: %v", err)
		return
	}
	if sources == nil {
		sources = []models.LogSource{}
	}
	writeJSON(w, http.StatusOK, sources)
}
