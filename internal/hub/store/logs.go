package store

import (
	"context"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (s *Store) InsertLog(ctx context.Context, entry *models.LogEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO logs (server_id, ts, source, level, message, metadata)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		entry.ServerID, entry.Ts.UTC().Format(time.RFC3339), entry.Source,
		entry.Level, entry.Message, entry.Metadata,
	)
	return err
}

func (s *Store) QueryLogs(ctx context.Context, serverID string, opts LogQueryOpts) ([]models.LogEntry, error) {
	query := `SELECT server_id, ts, source, level, message, metadata
			  FROM logs WHERE server_id = ?`
	args := []any{serverID}

	if opts.Source != "" {
		query += " AND source = ?"
		args = append(args, opts.Source)
	}
	if opts.Level != "" {
		query += " AND level = ?"
		args = append(args, opts.Level)
	}
	if opts.Search != "" {
		query += " AND message LIKE ?"
		args = append(args, "%"+opts.Search+"%")
	}
	if !opts.Since.IsZero() {
		query += " AND ts >= ?"
		args = append(args, opts.Since.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY ts DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	} else {
		query += " LIMIT 500"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.LogEntry
	for rows.Next() {
		var e models.LogEntry
		var ts string
		if err := rows.Scan(&e.ServerID, &ts, &e.Source, &e.Level, &e.Message, &e.Metadata); err != nil {
			return nil, err
		}
		e.Ts, _ = time.Parse(time.RFC3339, ts)
		result = append(result, e)
	}
	return result, nil
}

type LogQueryOpts struct {
	Source string
	Level  string
	Search string
	Since  time.Time
	Limit  int
}

func (s *Store) InsertAccessLog(ctx context.Context, entry *models.AccessLogEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO access_logs (server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ServerID, entry.Ts.UTC().Format(time.RFC3339), entry.Method, entry.Path,
		entry.StatusCode, entry.DurationMs, entry.IP, entry.UserAgent, entry.BytesSent,
	)
	return err
}
