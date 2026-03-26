package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/hub/uptime"
)

func (s *Store) InsertUptimeResult(ctx context.Context, r *uptime.Result) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO uptime_results (check_id, ts, status_code, duration_ms, error)
		 VALUES (?, ?, ?, ?, ?)`,
		r.CheckID, r.Ts.UTC().Format(time.RFC3339), r.StatusCode, r.ResponseTime, r.Error,
	)
	return err
}

func (s *Store) GetUptimeResults(ctx context.Context, checkID string, since time.Time) ([]uptime.Result, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT check_id, ts, status_code, duration_ms, error
		 FROM uptime_results WHERE check_id = ? AND ts >= ? ORDER BY ts DESC LIMIT 1000`,
		checkID, since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []uptime.Result
	for rows.Next() {
		var r uptime.Result
		var ts, errStr string
		if err := rows.Scan(&r.CheckID, &ts, &r.StatusCode, &r.ResponseTime, &errStr); err != nil {
			return nil, err
		}
		r.Ts, _ = time.Parse(time.RFC3339, ts)
		r.Error = errStr
		results = append(results, r)
	}
	return results, nil
}

func (s *Store) ListUptimeChecks(ctx context.Context) ([]uptime.Check, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, url, interval_seconds, timeout_seconds, enabled
		 FROM uptime_checks ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []uptime.Check
	for rows.Next() {
		var c uptime.Check
		var enabled int
		if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.Interval, &c.Timeout, &enabled); err != nil {
			return nil, err
		}
		c.Enabled = enabled == 1
		checks = append(checks, c)
	}
	return checks, nil
}

func (s *Store) CreateUptimeCheck(ctx context.Context, c *uptime.Check) error {
	if c.ID == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate uptime check id: %w", err)
		}
		c.ID = hex.EncodeToString(b)
	}
	if c.Interval == 0 {
		c.Interval = 60
	}
	if c.Timeout == 0 {
		c.Timeout = 10
	}
	enabled := 0
	if c.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO uptime_checks (id, name, url, interval_seconds, timeout_seconds, enabled)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.URL, c.Interval, c.Timeout, enabled,
	)
	return err
}

func (s *Store) DeleteUptimeCheck(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM uptime_checks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM uptime_results WHERE check_id = ?`, id)
	return err
}
