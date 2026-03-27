package main

import (
	"context"
	"encoding/json"

	"github.com/tudorAbrudan/tracelog/internal/hub"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

// logSourcesForLocalAgent returns enabled file log sources from the hub database
// that apply to this machine in serve mode (embedded agent).
// Records with empty server_id are treated as local; otherwise server_id must match
// the auto-created local server row.
func logSourcesForLocalAgent(ctx context.Context, h *hub.Hub) ([]models.LogSource, error) {
	localID, err := h.EnsureLocalServer(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.Store().ListLogSources(ctx)
	if err != nil {
		return nil, err
	}
	var out []models.LogSource
	for _, r := range rows {
		if !r.Enabled || r.Type != "file" || r.Path == "" {
			continue
		}
		if r.ServerID != "" && r.ServerID != localID {
			continue
		}
		src := models.LogSource{
			Name:      r.Name,
			Path:      r.Path,
			Type:      r.Type,
			Format:    r.Format,
			Container: r.Container,
			Enabled:   true,
		}
		if r.IngestLevels != "" {
			var levels []string
			if err := json.Unmarshal([]byte(r.IngestLevels), &levels); err == nil {
				src.IngestLevels = levels
			}
		}
		out = append(out, src)
	}
	return out, nil
}
