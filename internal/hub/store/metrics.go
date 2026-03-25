package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (s *Store) InsertMetrics(ctx context.Context, m *models.SystemMetrics) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO metrics (
			server_id, ts, cpu_percent,
			mem_used, mem_total, swap_used, swap_total,
			disk_used, disk_total, disk_read_bytes, disk_write_bytes,
			net_rx_bytes, net_tx_bytes,
			load_1, load_5, load_15, uptime
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ServerID, m.Ts.UTC().Format(time.RFC3339), m.CPUPercent,
		m.MemUsed, m.MemTotal, m.SwapUsed, m.SwapTotal,
		m.DiskUsed, m.DiskTotal, m.DiskReadBytes, m.DiskWriteBytes,
		m.NetRxBytes, m.NetTxBytes,
		m.Load1, m.Load5, m.Load15, m.Uptime,
	)
	return err
}

func (s *Store) QueryMetrics(ctx context.Context, serverID string, from, to time.Time) ([]models.SystemMetrics, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT server_id, ts, cpu_percent,
			mem_used, mem_total, swap_used, swap_total,
			disk_used, disk_total, disk_read_bytes, disk_write_bytes,
			net_rx_bytes, net_tx_bytes,
			load_1, load_5, load_15, uptime
		FROM metrics
		WHERE server_id = ? AND ts >= ? AND ts <= ?
		ORDER BY ts ASC`,
		serverID, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.SystemMetrics
	for rows.Next() {
		var m models.SystemMetrics
		var ts string
		if err := rows.Scan(
			&m.ServerID, &ts, &m.CPUPercent,
			&m.MemUsed, &m.MemTotal, &m.SwapUsed, &m.SwapTotal,
			&m.DiskUsed, &m.DiskTotal, &m.DiskReadBytes, &m.DiskWriteBytes,
			&m.NetRxBytes, &m.NetTxBytes,
			&m.Load1, &m.Load5, &m.Load15, &m.Uptime,
		); err != nil {
			return nil, err
		}
		m.Ts, _ = time.Parse(time.RFC3339, ts)
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (s *Store) CreateServer(ctx context.Context, name, host string) (*models.Server, error) {
	id := generateID()
	apiKey := "tl_" + generateID()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO servers (id, name, host, api_key, status, created_at)
		VALUES (?, ?, ?, ?, 'pending', ?)`,
		id, name, host, apiKey, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}

	return &models.Server{
		ID:        id,
		Name:      name,
		Host:      host,
		APIKey:    apiKey,
		Status:    "pending",
		CreatedAt: time.Now(),
	}, nil
}

func (s *Store) ListServers(ctx context.Context) ([]models.Server, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, COALESCE(host, ''), COALESCE(api_key, ''), status,
			COALESCE(last_seen_at, ''), created_at
		FROM servers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var srv models.Server
		var lastSeen, created string
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Host, &srv.APIKey, &srv.Status, &lastSeen, &created); err != nil {
			return nil, err
		}
		srv.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeen)
		srv.CreatedAt, _ = time.Parse(time.RFC3339, created)
		servers = append(servers, srv)
	}
	return servers, rows.Err()
}

func (s *Store) GetServer(ctx context.Context, id string) (*models.Server, error) {
	var srv models.Server
	var lastSeen, created string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(host, ''), COALESCE(api_key, ''), status,
			COALESCE(last_seen_at, ''), created_at
		FROM servers WHERE id = ?`, id,
	).Scan(&srv.ID, &srv.Name, &srv.Host, &srv.APIKey, &srv.Status, &lastSeen, &created)
	if err != nil {
		return nil, fmt.Errorf("server %s not found: %w", id, err)
	}
	srv.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeen)
	srv.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &srv, nil
}

func (s *Store) GetServerByAPIKey(ctx context.Context, apiKey string) (*models.Server, error) {
	var srv models.Server
	var lastSeen, created string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(host, ''), api_key, status,
			COALESCE(last_seen_at, ''), created_at
		FROM servers WHERE api_key = ?`, apiKey,
	).Scan(&srv.ID, &srv.Name, &srv.Host, &srv.APIKey, &srv.Status, &lastSeen, &created)
	if err != nil {
		return nil, err
	}
	srv.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeen)
	srv.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &srv, nil
}

func (s *Store) DeleteServer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM servers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	s.db.ExecContext(ctx, `DELETE FROM metrics WHERE server_id = ?`, id)
	s.db.ExecContext(ctx, `DELETE FROM docker_metrics WHERE server_id = ?`, id)
	s.db.ExecContext(ctx, `DELETE FROM logs WHERE server_id = ?`, id)
	s.db.ExecContext(ctx, `DELETE FROM access_logs WHERE server_id = ?`, id)
	return nil
}

func (s *Store) UpdateServerStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE servers SET status = ?, last_seen_at = ? WHERE id = ?`,
		status, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

func (s *Store) EnsureLocalServer(ctx context.Context) (*models.Server, error) {
	var srv models.Server
	var lastSeen, created string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(host, ''), api_key, status, COALESCE(last_seen_at, ''), created_at
		 FROM servers WHERE host = 'localhost' LIMIT 1`,
	).Scan(&srv.ID, &srv.Name, &srv.Host, &srv.APIKey, &srv.Status, &lastSeen, &created)
	if err == nil {
		srv.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeen)
		srv.CreatedAt, _ = time.Parse(time.RFC3339, created)
		return &srv, nil
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "this-server"
	}
	return s.CreateServer(ctx, hostname, "localhost")
}

func generateID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}
