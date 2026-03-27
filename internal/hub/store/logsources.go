package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type LogSourceRecord struct {
	ID               string   `json:"id"`
	ServerID         string   `json:"server_id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Path             string   `json:"path"`
	Container        string   `json:"container"`
	Format           string   `json:"format"`
	Enabled          bool     `json:"enabled"`
	IngestLevels     string   `json:"-"`                 // JSON array in DB; empty = ingest all severities
	IngestLevelsList []string `json:"ingest_levels,omitempty"`
}

func (s *Store) ListLogSources(ctx context.Context) ([]LogSourceRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, server_id, name, type, COALESCE(path, ''), COALESCE(container, ''), COALESCE(format, ''), enabled,
			COALESCE(ingest_levels, '')
		 FROM log_sources ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []LogSourceRecord
	for rows.Next() {
		var ls LogSourceRecord
		var enabled int
		if err := rows.Scan(&ls.ID, &ls.ServerID, &ls.Name, &ls.Type, &ls.Path, &ls.Container, &ls.Format, &enabled, &ls.IngestLevels); err != nil {
			return nil, err
		}
		ls.Enabled = enabled == 1
		if ls.IngestLevels != "" {
			var levels []string
			if err := json.Unmarshal([]byte(ls.IngestLevels), &levels); err == nil {
				ls.IngestLevelsList = levels
			}
		}
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
	var ingestJSON string
	if len(ls.IngestLevelsList) > 0 {
		b, err := json.Marshal(ls.IngestLevelsList)
		if err != nil {
			return fmt.Errorf("ingest_levels: %w", err)
		}
		ingestJSON = string(b)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO log_sources (id, server_id, name, type, path, container, format, enabled, ingest_levels)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ls.ID, ls.ServerID, ls.Name, ls.Type, ls.Path, ls.Container, ls.Format, enabled, nullIfEmpty(ingestJSON),
	)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// UpdateLogSourceIngestLevels sets which severities are stored for this source (JSON array). Empty clears the filter (all levels).
func (s *Store) UpdateLogSourceIngestLevels(ctx context.Context, id string, levels []string) error {
	var v any
	if len(levels) == 0 {
		v = nil
	} else {
		b, err := json.Marshal(levels)
		if err != nil {
			return err
		}
		v = string(b)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE log_sources SET ingest_levels = ? WHERE id = ?`, v, id)
	return err
}

func (s *Store) DeleteLogSource(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM log_sources WHERE id = ?`, id)
	return err
}

// ListLogSourcesForAgentServer returns enabled file log sources assigned to this server (remote agent tail).
func (s *Store) ListLogSourcesForAgentServer(ctx context.Context, serverID string) ([]models.LogSource, error) {
	if serverID == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, type, COALESCE(path, ''), COALESCE(container, ''), COALESCE(format, ''), COALESCE(ingest_levels, '')
		 FROM log_sources WHERE enabled = 1 AND type = 'file' AND TRIM(path) != '' AND server_id = ?`,
		serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.LogSource
	for rows.Next() {
		var name, typ, path, container, format, ingestJSON string
		if err := rows.Scan(&name, &typ, &path, &container, &format, &ingestJSON); err != nil {
			return nil, err
		}
		ls := models.LogSource{
			Name:      name,
			Path:      path,
			Type:      typ,
			Container: container,
			Format:    format,
			Enabled:   true,
		}
		if ingestJSON != "" {
			var levels []string
			if err := json.Unmarshal([]byte(ingestJSON), &levels); err == nil {
				ls.IngestLevels = levels
			}
		}
		out = append(out, ls)
	}
	return out, rows.Err()
}
