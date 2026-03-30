package hub

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tudorAbrudan/tracelog/internal/hub/alerts"
	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

// handleThreatIPInfo returns ipinfo data + threat assessment for given IPs.
// Tries cache first, then fetches from ipinfo.io API if configured and not cached.
// Auto-sends email alert for NEW IPs with BLOCK decision if channel configured in settings.
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

	// Get ipinfo.io API key + auto-alert channel from settings
	apiKey, _ := h.store.GetSetting(r.Context(), "ipinfo_io_api_key")
	autoAlertChannelID, _ := h.store.GetSetting(r.Context(), "ip_threat_auto_alert_channel")

	assessments := make([]*store.IPThreatAssessment, 0)
	for _, ip := range body.IPs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}

		// Try cache first
		ipinfo, err := h.store.GetCachedIPInfo(r.Context(), ip)
		if err != nil {
			slog.Debug("get cached ipinfo", "ip", ip, "error", err)
			continue
		}

		// If not cached and API key configured, fetch from ipinfo.io
		if ipinfo == nil && apiKey != "" {
			apiData, err := store.FetchIPInfoFromAPI(r.Context(), ip, apiKey)
			if err != nil {
				slog.Debug("fetch ipinfo from API", "ip", ip, "error", err)
				// Continue without the data rather than failing the whole request
			} else if apiData != nil {
				// Cache the fetched data
				if err := h.store.CacheIPInfo(r.Context(), ip, apiData); err != nil {
					slog.Debug("cache ipinfo", "ip", ip, "error", err)
				}
				ipinfo = apiData
			}
		}

		trafficScore := body.TrafficScores[ip]
		assessment := store.AssessIPThreat(ipinfo, trafficScore)
		assessments = append(assessments, assessment)

		// Auto-alert if NEW IP with BLOCK decision and channel configured
		if autoAlertChannelID != "" && assessment.Decision == "block" {
			wasAlerted, err := h.store.HasIPThreatBeenAlerted(r.Context(), ip)
			if err != nil {
				slog.Debug("check ip threat alert", "ip", ip, "error", err)
			} else if !wasAlerted {
				// New IP with BLOCK decision: send alert
				alert := &alerts.Alert{
					RuleID:  "ip_threat_" + strings.ReplaceAll(ip, ".", "_"),
					Metric:  "IP_THREAT_NEW",
					Message: "New IP threat detected: " + ip + " (" + assessment.Risk + " risk). Reasons: " + strings.Join(assessment.Reasons, "; "),
				}
				if err := h.notifyAlert(r.Context(), autoAlertChannelID, alert); err != nil {
					slog.Debug("send ip threat alert", "ip", ip, "channel", autoAlertChannelID, "error", err)
				}
				// Mark as alerted
				if err := h.store.RecordIPThreatAlert(r.Context(), ip); err != nil {
					slog.Debug("record ip threat alert", "ip", ip, "error", err)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"assessments": assessments})
}
