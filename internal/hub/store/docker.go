package store

import (
	"context"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (s *Store) InsertDockerMetrics(ctx context.Context, m *models.DockerMetrics) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO docker_metrics (server_id, ts, container_id, container_name, image, status, cpu_percent, mem_used, mem_limit, net_rx_bytes, net_tx_bytes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ServerID, m.Ts.UTC().Format(time.RFC3339), m.ContainerID, m.ContainerName, m.Image, m.Status,
		m.CPUPercent, m.MemUsed, m.MemLimit, m.NetRxBytes, m.NetTxBytes,
	)
	return err
}

func (s *Store) GetDockerMetrics(ctx context.Context, serverID string, since time.Time) ([]models.DockerMetrics, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT server_id, ts, container_id, container_name, image, status, cpu_percent, mem_used, mem_limit, net_rx_bytes, net_tx_bytes
		 FROM docker_metrics WHERE server_id = ? AND ts >= ? ORDER BY ts DESC`,
		serverID, since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.DockerMetrics
	for rows.Next() {
		var m models.DockerMetrics
		var ts string
		if err := rows.Scan(&m.ServerID, &ts, &m.ContainerID, &m.ContainerName, &m.Image, &m.Status,
			&m.CPUPercent, &m.MemUsed, &m.MemLimit, &m.NetRxBytes, &m.NetTxBytes); err != nil {
			return nil, err
		}
		m.Ts, _ = time.Parse(time.RFC3339, ts)
		result = append(result, m)
	}
	return result, nil
}
