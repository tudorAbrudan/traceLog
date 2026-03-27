package hub

import (
	"encoding/json"
	"net/http"
	"strings"
)

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
		"retention_days":                     true,
		"collection_interval":                true,
		"access_stats_exclude_ua_substrings": true,
	}
	for k, v := range settings {
		if !allowedKeys[k] {
			continue
		}
		if k == "access_stats_exclude_ua_substrings" {
			if strings.TrimSpace(v) == "" {
				v = "[]"
			}
			var tmp []string
			if err := json.Unmarshal([]byte(v), &tmp); err != nil {
				writeError(w, http.StatusBadRequest, "access_stats_exclude_ua_substrings must be a JSON array of strings")
				return
			}
		}
		if err := h.store.SetSetting(r.Context(), k, v); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save setting %s", k)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
