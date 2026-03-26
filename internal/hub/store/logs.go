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

// DeleteIngestedLogs removes rows from the logs table (TraceLog’s copy), not files on disk.
// If before is zero, all matching rows for the server are removed.
// If before is non-zero, only rows with ts strictly before before are removed.
// If source is non-empty, only that log source name is affected.
func (s *Store) DeleteIngestedLogs(ctx context.Context, serverID, source string, before time.Time) (int64, error) {
	var (
		q    string
		args []any
	)
	switch {
	case source != "" && !before.IsZero():
		q = `DELETE FROM logs WHERE server_id = ? AND source = ? AND ts < ?`
		args = []any{serverID, source, before.UTC().Format(time.RFC3339)}
	case source != "" && before.IsZero():
		q = `DELETE FROM logs WHERE server_id = ? AND source = ?`
		args = []any{serverID, source}
	case source == "" && !before.IsZero():
		q = `DELETE FROM logs WHERE server_id = ? AND ts < ?`
		args = []any{serverID, before.UTC().Format(time.RFC3339)}
	default:
		q = `DELETE FROM logs WHERE server_id = ?`
		args = []any{serverID}
	}
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

type LogQueryOpts struct {
	Source string
	Level  string
	Search string
	Since  time.Time
	Limit  int
}

