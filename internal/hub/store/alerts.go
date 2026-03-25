package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/tudorAbrudan/tracelog/internal/hub/alerts"
)

func (s *Store) ListAlertRules(ctx context.Context) ([]alerts.Rule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, COALESCE(server_id, ''), metric, operator, threshold,
		        COALESCE(duration_seconds, 0), COALESCE(cooldown_minutes, 30),
		        COALESCE(notify_channels, ''), enabled
		 FROM alert_rules ORDER BY metric`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []alerts.Rule
	for rows.Next() {
		var r alerts.Rule
		var enabled int
		if err := rows.Scan(&r.ID, &r.ServerID, &r.Metric, &r.Operator, &r.Threshold,
			&r.DurationS, &r.CooldownS, &r.ChannelID, &enabled); err != nil {
			return nil, err
		}
		r.CooldownS = r.CooldownS * 60
		r.Enabled = enabled == 1
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *Store) CreateAlertRule(ctx context.Context, r *alerts.Rule) error {
	if r.ID == "" {
		b := make([]byte, 8)
		rand.Read(b)
		r.ID = hex.EncodeToString(b)
	}
	enabled := 0
	if r.Enabled {
		enabled = 1
	}
	cooldownMin := r.CooldownS / 60
	if cooldownMin == 0 {
		cooldownMin = 30
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO alert_rules (id, server_id, metric, operator, threshold, duration_seconds, cooldown_minutes, notify_channels, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ServerID, r.Metric, r.Operator, r.Threshold, r.DurationS, cooldownMin, r.ChannelID, enabled,
	)
	return err
}

func (s *Store) DeleteAlertRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM alert_rules WHERE id = ?`, id)
	return err
}
