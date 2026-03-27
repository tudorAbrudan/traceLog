package hub

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tudorAbrudan/tracelog/internal/hub/alerts"
	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

func (h *Hub) handleListAlertHistory(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			limit = n
		}
	}
	rows, err := h.store.ListAlertHistoryRecent(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list alert history: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

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
	if alerts.IsDockerResourceMetric(rule.Metric) {
		rule.DockerContainer = strings.TrimSpace(rule.DockerContainer)
		if strings.TrimSpace(rule.ServerID) == "" {
			writeError(w, http.StatusBadRequest, "server_id is required for Docker metric alerts")
			return
		}
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

func (h *Hub) handleUpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var rule alerts.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	rule.ID = id
	if rule.Metric == "" || rule.Operator == "" {
		writeError(w, http.StatusBadRequest, "metric and operator are required")
		return
	}
	if alerts.IsDockerResourceMetric(rule.Metric) {
		rule.DockerContainer = strings.TrimSpace(rule.DockerContainer)
		if strings.TrimSpace(rule.ServerID) == "" {
			writeError(w, http.StatusBadRequest, "server_id is required for Docker metric alerts")
			return
		}
	}
	if err := h.store.UpdateAlertRule(r.Context(), &rule); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update alert rule: %v", err)
		return
	}
	h.alerts.RemoveRule(id)
	h.alerts.AddRule(&rule)
	writeJSON(w, http.StatusOK, rule)
}

func (h *Hub) handleListLogAlertSilences(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListLogAlertSilences(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list log alert silences: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Hub) handleCreateLogAlertSilence(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ServerID   string `json:"server_id"`
		Pattern    string `json:"pattern"`
		RuleMetric string `json:"rule_metric"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	rm := strings.TrimSpace(body.RuleMetric)
	if rm != "" {
		switch rm {
		case "log_critical", "log_error", "log_warn":
		default:
			writeError(w, http.StatusBadRequest, "rule_metric must be empty or one of log_critical, log_error, log_warn")
			return
		}
	}
	sid := strings.TrimSpace(body.ServerID)
	if sid != "" {
		if _, err := h.store.GetServer(r.Context(), sid); err != nil {
			writeError(w, http.StatusBadRequest, "Unknown server_id")
			return
		}
	}
	rec := &store.LogAlertSilence{
		ServerID:   sid,
		Pattern:    body.Pattern,
		RuleMetric: rm,
		Enabled:    true,
	}
	if err := h.store.CreateLogAlertSilence(r.Context(), rec); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (h *Hub) handleDeleteLogAlertSilence(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteLogAlertSilence(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete silence")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
