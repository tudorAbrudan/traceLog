package store

import (
	"context"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (s *Store) InsertAccessLog(ctx context.Context, entry *models.AccessLogEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO access_logs (server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ServerID, entry.Ts.UTC().Format(time.RFC3339),
		entry.Method, entry.Path, entry.StatusCode,
		entry.DurationMs, entry.IP, entry.UserAgent, entry.BytesSent,
	)
	return err
}

type AccessLogStats struct {
	TotalRequests int            `json:"total_requests"`
	StatusCodes   map[string]int `json:"status_codes"`
	TopPaths      []PathCount    `json:"top_paths"`
	TopIPs        []IPCount      `json:"top_ips"`
	ErrorRate     float64        `json:"error_rate"`
	AvgDuration   float64        `json:"avg_duration_ms"`
}

type PathCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

type IPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

func (s *Store) GetAccessLogStats(ctx context.Context, serverID string, since time.Time) (*AccessLogStats, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	stats := &AccessLogStats{
		StatusCodes: make(map[string]int),
	}

	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(AVG(duration_ms), 0),
		 COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1.0 ELSE 0.0 END) / NULLIF(COUNT(*), 0) * 100, 0)
		 FROM access_logs WHERE server_id = ? AND ts >= ?`,
		serverID, sinceStr,
	).Scan(&stats.TotalRequests, &stats.AvgDuration, &stats.ErrorRate)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT CAST(status_code / 100 AS TEXT) || 'xx', COUNT(*)
		 FROM access_logs WHERE server_id = ? AND ts >= ?
		 GROUP BY CAST(status_code / 100 AS TEXT) ORDER BY 2 DESC`,
		serverID, sinceStr,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			var count int
			if err := rows.Scan(&code, &count); err != nil {
				return nil, err
			}
			stats.StatusCodes[code] = count
		}
	}

	rows, err = s.db.QueryContext(ctx,
		`SELECT path, COUNT(*) as cnt FROM access_logs
		 WHERE server_id = ? AND ts >= ? GROUP BY path ORDER BY cnt DESC LIMIT 10`,
		serverID, sinceStr,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pc PathCount
			if err := rows.Scan(&pc.Path, &pc.Count); err != nil {
				return nil, err
			}
			stats.TopPaths = append(stats.TopPaths, pc)
		}
	}

	rows, err = s.db.QueryContext(ctx,
		`SELECT ip, COUNT(*) as cnt FROM access_logs
		 WHERE server_id = ? AND ts >= ? GROUP BY ip ORDER BY cnt DESC LIMIT 10`,
		serverID, sinceStr,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ic IPCount
			if err := rows.Scan(&ic.IP, &ic.Count); err != nil {
				return nil, err
			}
			stats.TopIPs = append(stats.TopIPs, ic)
		}
	}

	return stats, nil
}

func (s *Store) GetRecentAccessLogs(ctx context.Context, serverID string, limit int) ([]models.AccessLogEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent
		 FROM access_logs WHERE server_id = ? ORDER BY ts DESC LIMIT ?`,
		serverID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.AccessLogEntry
	for rows.Next() {
		var e models.AccessLogEntry
		var ts string
		if err := rows.Scan(&e.ServerID, &ts, &e.Method, &e.Path, &e.StatusCode,
			&e.DurationMs, &e.IP, &e.UserAgent, &e.BytesSent); err != nil {
			continue
		}
		e.Ts, _ = time.Parse(time.RFC3339, ts)
		result = append(result, e)
	}
	return result, nil
}
