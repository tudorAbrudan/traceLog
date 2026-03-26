package store

import (
	"context"
	"database/sql"
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
	TotalRequests   int               `json:"total_requests"`
	StatusCodes     map[string]int    `json:"status_codes"`
	TopPaths        []PathCount       `json:"top_paths"`
	TopIPs          []IPCount         `json:"top_ips"`
	UniqueIPCount   int               `json:"unique_ip_count"`
	TopMethodPaths  []MethodPathCount `json:"top_method_paths"`
	BadRequestsByIP []IPBadCount      `json:"bad_requests_by_ip"`
	ErrorRate       float64           `json:"error_rate"`
	AvgDuration     float64           `json:"avg_duration_ms"`
	// Filled by hub after store (blacklist matching), not by GetAccessLogStats.
	BlacklistedHits   int      `json:"blacklisted_hits"`
	BlacklistHitsNote string   `json:"blacklist_hits_note,omitempty"`
	BlacklistedInTop  []string `json:"blacklisted_in_top,omitempty"` // IPs in top tables that match policy
}

type PathCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

type IPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

type MethodPathCount struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Count  int    `json:"count"`
}

type IPBadCount struct {
	IP       string `json:"ip"`
	BadCount int    `json:"bad_count"`
}

// GetAccessLogStats aggregates HTTP access data. topN controls top paths, IPs, method+paths, and bad-by-IP rows (clamped 5–100).
func (s *Store) GetAccessLogStats(ctx context.Context, serverID string, since time.Time, topN int) (*AccessLogStats, error) {
	if topN < 5 {
		topN = 5
	}
	if topN > 100 {
		topN = 100
	}
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

	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT ip) FROM access_logs WHERE server_id = ? AND ts >= ? AND ip != ''`,
		serverID, sinceStr,
	).Scan(&stats.UniqueIPCount)

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
		 WHERE server_id = ? AND ts >= ? GROUP BY path ORDER BY cnt DESC LIMIT ?`,
		serverID, sinceStr, topN,
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
		 WHERE server_id = ? AND ts >= ? AND ip != '' GROUP BY ip ORDER BY cnt DESC LIMIT ?`,
		serverID, sinceStr, topN,
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

	rows, err = s.db.QueryContext(ctx,
		`SELECT method, path, COUNT(*) as cnt FROM access_logs
		 WHERE server_id = ? AND ts >= ? GROUP BY method, path ORDER BY cnt DESC LIMIT ?`,
		serverID, sinceStr, topN,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var mp MethodPathCount
			if err := rows.Scan(&mp.Method, &mp.Path, &mp.Count); err != nil {
				return nil, err
			}
			stats.TopMethodPaths = append(stats.TopMethodPaths, mp)
		}
	}

	badN := topN
	if badN > 50 {
		badN = 50
	}
	rows, err = s.db.QueryContext(ctx,
		`SELECT ip, COUNT(*) as cnt FROM access_logs
		 WHERE server_id = ? AND ts >= ? AND status_code >= 400 AND ip != ''
		 GROUP BY ip ORDER BY cnt DESC LIMIT ?`,
		serverID, sinceStr, badN,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ib IPBadCount
			if err := rows.Scan(&ib.IP, &ib.BadCount); err != nil {
				return nil, err
			}
			stats.BadRequestsByIP = append(stats.BadRequestsByIP, ib)
		}
	}

	return stats, nil
}

// GetAccessLogTopIPCounts returns IPs ordered by request count (for blacklist estimation).
func (s *Store) GetAccessLogTopIPCounts(ctx context.Context, serverID string, since time.Time, limit int) ([]IPCount, error) {
	if limit <= 0 {
		limit = 15000
	}
	if limit > 25000 {
		limit = 25000
	}
	sinceStr := since.UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx,
		`SELECT ip, COUNT(*) as cnt FROM access_logs
		 WHERE server_id = ? AND ts >= ? AND ip != '' GROUP BY ip ORDER BY cnt DESC LIMIT ?`,
		serverID, sinceStr, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []IPCount
	for rows.Next() {
		var ic IPCount
		if err := rows.Scan(&ic.IP, &ic.Count); err != nil {
			return nil, err
		}
		out = append(out, ic)
	}
	return out, rows.Err()
}

// QueryAccessBadRequests returns recent rows with status_code >= 400, optionally filtered by IP.
func (s *Store) QueryAccessBadRequests(ctx context.Context, serverID string, since time.Time, ip string, limit int) ([]models.AccessLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	sinceStr := since.UTC().Format(time.RFC3339)
	var (
		rows *sql.Rows
		err  error
	)
	if ip != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent
			 FROM access_logs WHERE server_id = ? AND ts >= ? AND status_code >= 400 AND ip = ?
			 ORDER BY ts DESC LIMIT ?`,
			serverID, sinceStr, ip, limit,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent
			 FROM access_logs WHERE server_id = ? AND ts >= ? AND status_code >= 400
			 ORDER BY ts DESC LIMIT ?`,
			serverID, sinceStr, limit,
		)
	}
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
	return result, rows.Err()
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
