package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IPInfoData represents cached ipinfo.io data for an IP.
type IPInfoData struct {
	IP              string  `json:"ip"`
	Country         string  `json:"country,omitempty"`
	Region          string  `json:"region,omitempty"`
	City            string  `json:"city,omitempty"`
	IsVPN           bool    `json:"is_vpn,omitempty"`
	IsProxy         bool    `json:"is_proxy,omitempty"`
	IsBot           bool    `json:"is_bot,omitempty"`
	AbuseConfidence float64 `json:"abuse_confidence,omitempty"` // 0-100
	FetchedAt       string  `json:"fetched_at,omitempty"`
}

// CacheIPInfo stores ipinfo data in the cache.
func (s *Store) CacheIPInfo(ctx context.Context, ip string, data *IPInfoData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ipinfo_cache (ip, data, fetched_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
		ip, string(jsonData))
	return err
}

// GetCachedIPInfo retrieves cached ipinfo data for an IP. Returns nil if not cached or cache is stale (>7 days).
func (s *Store) GetCachedIPInfo(ctx context.Context, ip string) (*IPInfoData, error) {
	var jsonData string
	err := s.db.QueryRowContext(ctx,
		`SELECT data FROM ipinfo_cache WHERE ip = ? AND datetime(fetched_at) > datetime('now', '-7 days')`,
		ip).Scan(&jsonData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not cached or stale
		}
		return nil, err
	}
	var data IPInfoData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// FetchIPInfoFromAPI calls ipinfo.io API to get IP geolocation + abuse data.
// apiKey is the ipinfo.io bearer token; if empty, returns nil (not an error).
func FetchIPInfoFromAPI(ctx context.Context, ip string, apiKey string) (*IPInfoData, error) {
	if apiKey == "" {
		return nil, nil // API key not configured
	}

	// ipinfo.io response structure for /lite/{ip} endpoint
	type ipinfoResponse struct {
		IP      string `json:"ip"`
		Country string `json:"country"`
		Region  string `json:"region"`
		City    string `json:"city"`
		Loc     string `json:"loc"` // lat,lon
		Org     string `json:"org"`
		Postal  string `json:"postal"`
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipinfo.io/lite/"+ip, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch ipinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ipinfo API returned %d: %s", resp.StatusCode, string(body))
	}

	var raw ipinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	data := &IPInfoData{
		IP:      raw.IP,
		Country: raw.Country,
		Region:  raw.Region,
		City:    raw.City,
	}

	// Note: ipinfo.io /lite endpoint doesn't include abuse/vpn/proxy/bot data.
	// For that, you'd need the full endpoint or a separate abuse database API.
	// This implementation leaves those fields for future integration with
	// abuseipdb.com or other abuse scoring services.

	return data, nil
}

// IPThreatAssessment evaluates whether an IP is a threat.
type IPThreatAssessment struct {
	IP       string   `json:"ip"`
	Risk     string   `json:"risk"` // "high", "medium", "low", "unknown"
	Reasons  []string `json:"reasons,omitempty"`
	Decision string   `json:"decision"` // "block", "monitor", "allow"
	Score    int      `json:"score"`
}

// RecordIPThreatAlert marks an IP as having been alerted on (for auto-notification tracking).
func (s *Store) RecordIPThreatAlert(ctx context.Context, ip string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO ip_threat_alerts (ip, first_seen, last_alerted)
		 VALUES (?, COALESCE((SELECT first_seen FROM ip_threat_alerts WHERE ip = ?), CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)`,
		ip, ip)
	return err
}

// HasIPThreatBeenAlerted checks if we've already sent an alert for this IP (ever, not time-based).
func (s *Store) HasIPThreatBeenAlerted(ctx context.Context, ip string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ip_threat_alerts WHERE ip = ?`,
		ip).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AssessIPThreat returns a threat assessment based on ipinfo data and access stats.
// score is the existing threat score from HTTP Analytics (0-10+).
func AssessIPThreat(ipinfo *IPInfoData, trafficScore int) *IPThreatAssessment {
	assess := &IPThreatAssessment{
		Risk:  "low",
		Score: trafficScore,
	}

	if ipinfo == nil {
		assess.Risk = "unknown"
		assess.Decision = "monitor"
		return assess
	}

	assess.IP = ipinfo.IP

	// Check abuse confidence
	if ipinfo.AbuseConfidence > 50 {
		assess.Score += 5
		assess.Reasons = append(assess.Reasons, fmt.Sprintf("High abuse confidence (%.0f%%)", ipinfo.AbuseConfidence))
	} else if ipinfo.AbuseConfidence > 25 {
		assess.Score += 2
		assess.Reasons = append(assess.Reasons, fmt.Sprintf("Moderate abuse confidence (%.0f%%)", ipinfo.AbuseConfidence))
	}

	// Check VPN/Proxy
	if ipinfo.IsVPN {
		assess.Score += 1
		assess.Reasons = append(assess.Reasons, "VPN detected")
	}
	if ipinfo.IsProxy {
		assess.Score += 1
		assess.Reasons = append(assess.Reasons, "Proxy detected")
	}

	// Check bot
	if ipinfo.IsBot {
		assess.Score += 2
		assess.Reasons = append(assess.Reasons, "Bot detected")
	}

	// Determine risk level
	if assess.Score >= 6 {
		assess.Risk = "high"
		assess.Decision = "block"
	} else if assess.Score >= 3 {
		assess.Risk = "medium"
		assess.Decision = "monitor"
	} else {
		assess.Risk = "low"
		assess.Decision = "allow"
	}

	return assess
}
