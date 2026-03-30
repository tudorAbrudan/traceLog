package hub

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tudorAbrudan/tracelog/internal/hub/alerts"
)

// handleCreateIPThreatAlert creates an alert for a specific IP and sends email notification.
// POST /api/threat/alert-ip
// Body: {"ip": "1.2.3.4", "reason": "Recommended to block", "channel_id": "xyz"}
func (h *Hub) handleCreateIPThreatAlert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IP        string `json:"ip"`
		Reason    string `json:"reason,omitempty"`
		ChannelID string `json:"channel_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if body.IP == "" || body.ChannelID == "" {
		writeError(w, http.StatusBadRequest, "ip and channel_id required")
		return
	}

	// Create alert message
	reason := body.Reason
	if reason == "" {
		reason = "Threat detected"
	}
	message := "IP " + body.IP + " flagged for threat: " + reason

	// Fire alert immediately
	alert := &alerts.Alert{
		RuleID:    "ip_threat_" + strings.ReplaceAll(body.IP, ".", "_"),
		Metric:    "IP_THREAT",
		Value:     1,
		Threshold: 0,
		Message:   message,
	}

	if err := h.notifyAlert(r.Context(), body.ChannelID, alert); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to send alert")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "alert sent", "message": message})
}
