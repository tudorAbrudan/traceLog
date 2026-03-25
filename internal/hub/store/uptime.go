package store

import (
	"context"
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
