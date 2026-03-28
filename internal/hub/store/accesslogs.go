package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type AccessTimelinePoint struct {
	Ts    time.Time `json:"ts"`
	Count int       `json:"count"`
}

type AccessTimeline struct {
	Points        []AccessTimelinePoint `json:"points"`
	BucketMinutes int                   `json:"bucket_minutes"`
}

// accessLogExcludeHubUIPrefixSQL drops rows whose path is under the hub’s public URL prefix (same as --url-prefix / TRACELOG_URL_PREFIX).
// normalizedPrefix is the hub’s public path (from --url-prefix / TRACELOG_URL_PREFIX), e.g. "/monitor"; "" = no filter (hub at site root).
func accessLogExcludeHubUIPrefixSQL(normalizedPrefix string) (cond string, args []any) {
	p := strings.TrimSpace(normalizedPrefix)
	if p == "" {
		return "", nil
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimSuffix(p, "/")
	p = strings.ToLower(p)
	if p == "" || p == "/" {
		return "", nil
	}
	cond = ` AND NOT (
  LOWER(TRIM(COALESCE(path,''))) = ?
  OR LOWER(TRIM(COALESCE(path,''))) LIKE ?
  OR LOWER(TRIM(COALESCE(path,''))) LIKE ?
)`
	args = []any{p, p + "/%", p + "?%"}
	return cond, args
}

// accessUAExcludeSQL builds AND NOT (INSTR(...) OR ...) to drop rows whose User-Agent contains any pattern (case-insensitive).
func accessUAExcludeSQL(patterns []string) (cond string, args []any) {
	var parts []string
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(p) > 200 {
			p = p[:200]
		}
		pat := strings.ToLower(p)
		parts = append(parts, `INSTR(LOWER(COALESCE(user_agent,'')), ?) > 0`)
		args = append(args, pat)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return ` AND NOT (` + strings.Join(parts, " OR ") + `)`, args
}

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
	TotalRequests      int                `json:"total_requests"`
	StatusCodes        map[string]int     `json:"status_codes"`
	TopPaths           []PathCount        `json:"top_paths"`
	TopIPs             []IPCount          `json:"top_ips"`
	UniqueIPCount      int                `json:"unique_ip_count"`
	TopMethodPaths     []MethodPathCount  `json:"top_method_paths"`
	BadRequestsByIP    []IPBadCount       `json:"bad_requests_by_ip"`
	ErrorRate          float64            `json:"error_rate"`
	AvgDuration        float64            `json:"avg_duration_ms"`
	TopPathsByDuration []PathDurationStat `json:"top_paths_by_duration"`
	// Filled by hub after store (blacklist matching), not by GetAccessLogStats.
	BlacklistedHits   int      `json:"blacklisted_hits"`
	BlacklistHitsNote string   `json:"blacklist_hits_note,omitempty"`
	BlacklistedInTop  []string `json:"blacklisted_in_top,omitempty"` // IPs in top tables that match policy
}

type PathDurationStat struct {
	Path  string  `json:"path"`
	AvgMs float64 `json:"avg_ms"`
	Count int     `json:"count"`
}

type PathCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

type IPCount struct {
	IP        string `json:"ip"`
	Count     int    `json:"count"`
	BytesSent int64  `json:"bytes_sent"`
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
// excludeUASubstrings removes matching User-Agent rows from all aggregates (e.g. TraceLog’s own uptime probes).
// excludeHubPathPrefix is the same normalized prefix as at install (not hardcoded): drop those paths from aggregates; empty skips.
//
//nolint:gosec // G202: baseWhere is fixed SQL + accessUAExcludeSQL (INSTR…?) + accessLogExcludeHubUIPrefixSQL (? bound); UA/prefix bound as args.
func (s *Store) GetAccessLogStats(ctx context.Context, serverID string, since time.Time, topN int, excludeUASubstrings []string, excludeHubPathPrefix string) (*AccessLogStats, error) {
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
	uaCond, uaArgs := accessUAExcludeSQL(excludeUASubstrings)
	pathCond, pathArgs := accessLogExcludeHubUIPrefixSQL(excludeHubPathPrefix)
	baseArgs := []any{serverID, sinceStr}
	baseWhere := `server_id = ? AND ts >= ?` + uaCond + pathCond

	rowArgs := func() []any {
		return append(append(append([]any{}, baseArgs...), uaArgs...), pathArgs...)
	}

	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(AVG(duration_ms), 0),
		 COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1.0 ELSE 0.0 END) / NULLIF(COUNT(*), 0) * 100, 0)
		 FROM access_logs WHERE `+baseWhere,
		rowArgs()...,
	).Scan(&stats.TotalRequests, &stats.AvgDuration, &stats.ErrorRate)
	if err != nil {
		return nil, err
	}

	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT ip) FROM access_logs WHERE `+baseWhere+` AND ip != ''`,
		rowArgs()...,
	).Scan(&stats.UniqueIPCount)

	rows, err := s.db.QueryContext(ctx,
		`SELECT CAST(status_code / 100 AS TEXT) || 'xx', COUNT(*)
		 FROM access_logs WHERE `+baseWhere+`
		 GROUP BY CAST(status_code / 100 AS TEXT) ORDER BY 2 DESC`,
		rowArgs()...,
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

	qArgs := rowArgs()
	rows, err = s.db.QueryContext(ctx,
		`SELECT path, COUNT(*) as cnt FROM access_logs
		 WHERE `+baseWhere+` GROUP BY path ORDER BY cnt DESC LIMIT ?`,
		append(qArgs, topN)...,
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
		`SELECT ip, COUNT(*) as cnt, COALESCE(SUM(bytes_sent), 0) AS bytes_sent FROM access_logs
		 WHERE `+baseWhere+` AND ip != '' GROUP BY ip ORDER BY cnt DESC LIMIT ?`,
		append(qArgs, topN)...,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ic IPCount
			if err := rows.Scan(&ic.IP, &ic.Count, &ic.BytesSent); err != nil {
				return nil, err
			}
			stats.TopIPs = append(stats.TopIPs, ic)
		}
	}

	rows, err = s.db.QueryContext(ctx,
		`SELECT method, path, COUNT(*) as cnt FROM access_logs
		 WHERE `+baseWhere+` GROUP BY method, path ORDER BY cnt DESC LIMIT ?`,
		append(qArgs, topN)...,
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
		 WHERE `+baseWhere+` AND status_code >= 400 AND ip != ''
		 GROUP BY ip ORDER BY cnt DESC LIMIT ?`,
		append(qArgs, badN)...,
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

	rows, err = s.db.QueryContext(ctx,
		`SELECT path, AVG(duration_ms) AS avg_ms, COUNT(*) AS cnt
		 FROM access_logs
		 WHERE `+baseWhere+` AND path != '' AND duration_ms > 0
		 GROUP BY path
		 HAVING cnt >= 3
		 ORDER BY avg_ms DESC
		 LIMIT ?`,
		append(qArgs, topN)...,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pd PathDurationStat
			if err := rows.Scan(&pd.Path, &pd.AvgMs, &pd.Count); err != nil {
				return nil, err
			}
			stats.TopPathsByDuration = append(stats.TopPathsByDuration, pd)
		}
	}

	return stats, nil
}

// GetAccessLogTopIPCounts returns IPs ordered by request count (for blacklist estimation).
//
//nolint:gosec // G202: same WHERE as GetAccessLogStats (UA exclude + hub path prefix + bound args).
func (s *Store) GetAccessLogTopIPCounts(ctx context.Context, serverID string, since time.Time, limit int, excludeUASubstrings []string, excludeHubPathPrefix string) ([]IPCount, error) {
	if limit <= 0 {
		limit = 15000
	}
	if limit > 25000 {
		limit = 25000
	}
	sinceStr := since.UTC().Format(time.RFC3339)
	uaCond, uaArgs := accessUAExcludeSQL(excludeUASubstrings)
	pathCond, pathArgs := accessLogExcludeHubUIPrefixSQL(excludeHubPathPrefix)
	baseWhere := `server_id = ? AND ts >= ?` + uaCond + pathCond
	base := []any{serverID, sinceStr}
	q := append(append(append([]any{}, base...), uaArgs...), pathArgs...)
	rows, err := s.db.QueryContext(ctx,
		`SELECT ip, COUNT(*) as cnt FROM access_logs
		 WHERE `+baseWhere+` AND ip != '' GROUP BY ip ORDER BY cnt DESC LIMIT ?`,
		append(q, limit)...,
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
//
//nolint:gosec // G202: pathCond from accessLogExcludeHubUIPrefixSQL only (fixed SQL + ?-bound prefix args).
func (s *Store) QueryAccessBadRequests(ctx context.Context, serverID string, since time.Time, ip string, limit int, excludeHubPathPrefix string) ([]models.AccessLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	sinceStr := since.UTC().Format(time.RFC3339)
	pathCond, pathArgs := accessLogExcludeHubUIPrefixSQL(excludeHubPathPrefix)
	var (
		rows *sql.Rows
		err  error
	)
	if ip != "" {
		args := append([]any{serverID, sinceStr, ip}, pathArgs...)
		args = append(args, limit)
		rows, err = s.db.QueryContext(ctx,
			`SELECT server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent
			 FROM access_logs WHERE server_id = ? AND ts >= ? AND status_code >= 400 AND ip = ?`+pathCond+`
			 ORDER BY ts DESC LIMIT ?`,
			args...,
		)
	} else {
		args := append([]any{serverID, sinceStr}, pathArgs...)
		args = append(args, limit)
		rows, err = s.db.QueryContext(ctx,
			`SELECT server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent
			 FROM access_logs WHERE server_id = ? AND ts >= ? AND status_code >= 400`+pathCond+`
			 ORDER BY ts DESC LIMIT ?`,
			args...,
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

// QueryAccessSlowRequests returns rows with duration_ms >= minDurationMs, newest slowest first.
// Uses the same User-Agent exclude patterns and hub UI path filter as HTTP analytics aggregates.
//
//nolint:gosec // G202: uaCond + pathCond from helpers only (fixed SQL + bound args).
func (s *Store) QueryAccessSlowRequests(ctx context.Context, serverID string, since time.Time, minDurationMs float64, limit int, excludeUASubstrings []string, excludeHubPathPrefix string) ([]models.AccessLogEntry, error) {
	if minDurationMs <= 0 {
		minDurationMs = 500
	}
	if minDurationMs > 3_600_000 { // 1 hour — avoid absurd thresholds
		minDurationMs = 3_600_000
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	sinceStr := since.UTC().Format(time.RFC3339)
	uaCond, uaArgs := accessUAExcludeSQL(excludeUASubstrings)
	pathCond, pathArgs := accessLogExcludeHubUIPrefixSQL(excludeHubPathPrefix)
	args := []any{serverID, sinceStr}
	args = append(args, uaArgs...)
	args = append(args, pathArgs...)
	args = append(args, minDurationMs, limit)

	rows, err := s.db.QueryContext(ctx,
		`SELECT server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent
		 FROM access_logs WHERE server_id = ? AND ts >= ?`+uaCond+pathCond+` AND duration_ms >= ?
		 ORDER BY duration_ms DESC, ts DESC LIMIT ?`,
		args...,
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
	return result, rows.Err()
}

//nolint:gosec // G202: pathCond from accessLogExcludeHubUIPrefixSQL only (fixed SQL + ?-bound prefix args).
func (s *Store) GetRecentAccessLogs(ctx context.Context, serverID string, limit int, excludeHubPathPrefix string) ([]models.AccessLogEntry, error) {
	pathCond, pathArgs := accessLogExcludeHubUIPrefixSQL(excludeHubPathPrefix)
	args := append([]any{serverID}, pathArgs...)
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx,
		`SELECT server_id, ts, method, path, status_code, duration_ms, ip, user_agent, bytes_sent
		 FROM access_logs WHERE server_id = ?`+pathCond+` ORDER BY ts DESC LIMIT ?`,
		args...,
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

// GetAccessTimeline returns request counts bucketed by time for chart rendering.
// bucketSec is an integer computed from a fixed switch — not user input.
//
//nolint:gosec // G202: bucketSec is an integer computed from a fixed switch, not user input; other conditions use bound args only.
func (s *Store) GetAccessTimeline(ctx context.Context, serverID, since, hubPathPrefix string) (*AccessTimeline, error) {
	bucketMin := 60
	switch since {
	case "1h":
		bucketMin = 5
	case "6h":
		bucketMin = 15
	case "24h":
		bucketMin = 60
	case "7d":
		bucketMin = 360
	case "30d":
		bucketMin = 1440
	}
	bucketSec := bucketMin * 60

	var rangeStart time.Time
	switch since {
	case "1h":
		rangeStart = time.Now().Add(-1 * time.Hour)
	case "6h":
		rangeStart = time.Now().Add(-6 * time.Hour)
	case "7d":
		rangeStart = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		rangeStart = time.Now().Add(-30 * 24 * time.Hour)
	default: // "24h" and any unrecognised value
		rangeStart = time.Now().Add(-24 * time.Hour)
	}

	hubPathExclude := ""
	if hubPathPrefix != "" {
		hubPathExclude = hubPathPrefix
	}

	// bucketSec is an integer from a fixed switch — safe to interpolate; remaining args are bound parameters.
	rows, err := s.db.QueryContext(ctx, `
        SELECT
            (CAST(strftime('%s', ts) AS INTEGER) / ?) * ? AS bucket_ts,
            COUNT(*) AS cnt
        FROM access_logs
        WHERE server_id = ? AND ts >= ?
          AND NOT INSTR(path, ?)
        GROUP BY bucket_ts
        ORDER BY bucket_ts ASC
    `, bucketSec, bucketSec, serverID, rangeStart.UTC().Format(time.RFC3339), hubPathExclude)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []AccessTimelinePoint
	for rows.Next() {
		var bucketTs int64
		var cnt int
		if err := rows.Scan(&bucketTs, &cnt); err != nil {
			return nil, err
		}
		points = append(points, AccessTimelinePoint{
			Ts:    time.Unix(bucketTs, 0).UTC(),
			Count: cnt,
		})
	}
	if points == nil {
		points = []AccessTimelinePoint{}
	}
	return &AccessTimeline{Points: points, BucketMinutes: bucketMin}, nil
}
