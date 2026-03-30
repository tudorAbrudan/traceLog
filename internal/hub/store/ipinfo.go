package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

// IPThreatAssessment evaluates whether an IP is a threat.
type IPThreatAssessment struct {
	IP       string   `json:"ip"`
	Risk     string   `json:"risk"` // "high", "medium", "low", "unknown"
	Reasons  []string `json:"reasons,omitempty"`
	Decision string   `json:"decision"` // "block", "monitor", "allow"
	Score    int      `json:"score"`
}

// AssessIPThreat returns a threat assessment based on ipinfo data and access stats.
// score is the existing threat score from HTTP Analytics (0-10+).
func AssessIPThreat(ipinfo *IPInfoData, trafficScore int) *IPThreatAssessment {
	assess := &IPThreatAssessment{
		IP:    ipinfo.IP,
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
