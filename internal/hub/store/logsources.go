package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type LogSourceRecord struct {
	ID        string `json:"id"`
	ServerID  string `json:"server_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	Container string `json:"container"`
	Format    string `json:"format"`
	Enabled   bool   `json:"enabled"`
}

func (s *Store) ListLogSources(ctx context.Context) ([]LogSourceRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, server_id, name, type, COALESCE(path, ''), COALESCE(container, ''), COALESCE(format, ''), enabled
		 FROM log_sources ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []LogSourceRecord
	for rows.Next() {
		var ls LogSourceRecord
		var enabled int
		if err := rows.Scan(&ls.ID, &ls.ServerID, &ls.Name, &ls.Type, &ls.Path, &ls.Container, &ls.Format, &enabled); err != nil {
			return nil, err
		}
		ls.Enabled = enabled == 1
		sources = append(sources, ls)
	}
	return sources, nil
}

func (s *Store) CreateLogSource(ctx context.Context, ls *LogSourceRecord) error {
	if ls.ID == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate log source id: %w", err)
		}
		ls.ID = hex.EncodeToString(b)
	}
	enabled := 0
	if ls.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO log_sources (id, server_id, name, type, path, container, format, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ls.ID, ls.ServerID, ls.Name, ls.Type, ls.Path, ls.Container, ls.Format, enabled,
	)
	return err
}

func (s *Store) DeleteLogSource(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM log_sources WHERE id = ?`, id)
	return err
}
