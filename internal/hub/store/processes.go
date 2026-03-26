package store

import (
	"context"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (s *Store) InsertProcessMetrics(ctx context.Context, metrics []models.ProcessMetrics) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback() //nolint:errcheck // expected sql.ErrTxDone after Commit
	}()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO process_metrics (server_id, ts, pid, name, cmdline, status, cpu_percent, mem_rss, mem_vms, threads)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range metrics {
		_, err := stmt.ExecContext(ctx,
			m.ServerID, m.Ts.UTC().Format(time.RFC3339), m.PID, m.Name, m.Cmdline,
			m.Status, m.CPU, m.MemRSS, m.MemVMS, m.Threads,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetProcessMetrics(ctx context.Context, serverID string, since time.Time) ([]models.ProcessMetrics, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT server_id, ts, pid, name, cmdline, status, cpu_percent, mem_rss, mem_vms, threads
		 FROM process_metrics WHERE server_id = ? AND ts >= ? ORDER BY ts DESC LIMIT 500`,
		serverID, since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProcessMetrics
	for rows.Next() {
		var m models.ProcessMetrics
		var ts string
		if err := rows.Scan(&m.ServerID, &ts, &m.PID, &m.Name, &m.Cmdline,
			&m.Status, &m.CPU, &m.MemRSS, &m.MemVMS, &m.Threads); err != nil {
			return nil, err
		}
		m.Ts, _ = time.Parse(time.RFC3339, ts)
		result = append(result, m)
	}
	return result, nil
}

func (s *Store) GetLatestProcesses(ctx context.Context, serverID string) ([]models.ProcessMetrics, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT server_id, ts, pid, name, cmdline, status, cpu_percent, mem_rss, mem_vms, threads
		 FROM process_metrics
		 WHERE server_id = ? AND ts = (SELECT MAX(ts) FROM process_metrics WHERE server_id = ?)
		 ORDER BY cpu_percent DESC`,
		serverID, serverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProcessMetrics
	for rows.Next() {
		var m models.ProcessMetrics
		var ts string
		if err := rows.Scan(&m.ServerID, &ts, &m.PID, &m.Name, &m.Cmdline,
			&m.Status, &m.CPU, &m.MemRSS, &m.MemVMS, &m.Threads); err != nil {
			return nil, err
		}
		m.Ts, _ = time.Parse(time.RFC3339, ts)
		result = append(result, m)
	}
	return result, nil
}
