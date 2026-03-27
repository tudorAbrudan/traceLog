package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const maxSilencePatternLen = 512

type LogAlertSilence struct {
	ID         string `json:"id"`
	ServerID   string `json:"server_id"`
	Pattern    string `json:"pattern"`
	RuleMetric string `json:"rule_metric"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"created_at"`
}

func (s *Store) ListLogAlertSilences(ctx context.Context) ([]LogAlertSilence, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, COALESCE(server_id, ''), pattern, COALESCE(rule_metric, ''), enabled, COALESCE(created_at, '')
		 FROM log_alert_silences ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LogAlertSilence
	for rows.Next() {
		var r LogAlertSilence
		var en int
		if err := rows.Scan(&r.ID, &r.ServerID, &r.Pattern, &r.RuleMetric, &en, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Enabled = en == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateLogAlertSilence(ctx context.Context, r *LogAlertSilence) error {
	pat := strings.TrimSpace(r.Pattern)
	if pat == "" {
		return fmt.Errorf("pattern is required")
	}
	if len(pat) > maxSilencePatternLen {
		return fmt.Errorf("pattern too long (max %d)", maxSilencePatternLen)
	}
	if r.ID == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		r.ID = hex.EncodeToString(b)
	}
	en := 0
	if r.Enabled {
		en = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO log_alert_silences (id, server_id, pattern, rule_metric, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, strings.TrimSpace(r.ServerID), pat, strings.TrimSpace(r.RuleMetric), en, now,
	)
	return err
}

func (s *Store) DeleteLogAlertSilence(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM log_alert_silences WHERE id = ?`, id)
	return err
}

// IsLogAlertSilenced returns true if this log line should not trigger the given log alert rule (ruleMetric like log_error).
func (s *Store) IsLogAlertSilenced(ctx context.Context, serverID, message, ruleMetric string) (bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT server_id, pattern, rule_metric FROM log_alert_silences WHERE enabled = 1`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	msgLower := strings.ToLower(message)
	for rows.Next() {
		var silServer, pattern, silRule string
		if err := rows.Scan(&silServer, &pattern, &silRule); err != nil {
			return false, err
		}
		if silServer != "" && silServer != serverID {
			continue
		}
		if silRule != "" && silRule != ruleMetric {
			continue
		}
		pat := strings.TrimSpace(pattern)
		if pat == "" {
			continue
		}
		if strings.Contains(msgLower, strings.ToLower(pat)) {
			return true, nil
		}
	}
	return false, rows.Err()
}
