package store

import (
	"context"
	"fmt"
)

// InsertAlertHistory records a fired alert for the dashboard history list.
func (s *Store) InsertAlertHistory(ctx context.Context, ruleID, serverID, state, message string) error {
	if ruleID == "" {
		ruleID = "unknown"
	}
	if state == "" {
		state = "fired"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO alert_history (rule_id, server_id, state, message) VALUES (?, ?, ?, ?)`,
		ruleID, nullString(serverID), state, message)
	return err
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// AlertHistoryRow is one stored notification event.
type AlertHistoryRow struct {
	ID       int64  `json:"id"`
	RuleID   string `json:"rule_id"`
	ServerID string `json:"server_id"`
	State    string `json:"state"`
	Message  string `json:"message"`
	Ts       string `json:"ts"`
}

// ListAlertHistoryRecent returns newest rows first.
func (s *Store) ListAlertHistoryRecent(ctx context.Context, limit int) ([]AlertHistoryRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, rule_id, COALESCE(server_id, ''), state, COALESCE(message, ''), COALESCE(ts, '')
		 FROM alert_history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list alert history: %w", err)
	}
	defer rows.Close()

	var out []AlertHistoryRow
	for rows.Next() {
		var r AlertHistoryRow
		if err := rows.Scan(&r.ID, &r.RuleID, &r.ServerID, &r.State, &r.Message, &r.Ts); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []AlertHistoryRow{}
	}
	return out, nil
}
