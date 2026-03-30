package hub

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

// handleThreatIPInfo returns cached ipinfo data + threat assessment for given IPs.
// POST /api/threat/ipinfo
// Body: {"ips": ["1.2.3.4", ...], "traffic_scores": {"1.2.3.4": 5, ...}}
func (h *Hub) handleThreatIPInfo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IPs           []string       `json:"ips"`
		TrafficScores map[string]int `json:"traffic_scores,omitempty"` // IP -> score from HTTP Analytics
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	assessments := make([]*store.IPThreatAssessment, 0)
	for _, ip := range body.IPs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}

		ipinfo, err := h.store.GetCachedIPInfo(r.Context(), ip)
		if err != nil {
			// Log but continue
			continue
		}

		trafficScore := body.TrafficScores[ip]
		assessment := store.AssessIPThreat(ipinfo, trafficScore)
		assessments = append(assessments, assessment)
	}

	writeJSON(w, http.StatusOK, map[string]any{"assessments": assessments})
}
