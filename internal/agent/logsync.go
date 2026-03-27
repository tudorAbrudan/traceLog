package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/agent/collector"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

func fetchRemoteLogSources(ctx context.Context, hubBase, apiKey string) ([]models.LogSource, error) {
	u, err := AgentLogSourcesURL(hubBase)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("User-Agent", "TraceLog-Agent/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		n := 200
		if len(body) < n {
			n = len(body)
		}
		return nil, fmt.Errorf("hub returned %s: %s", resp.Status, string(body[:n]))
	}
	var out []models.LogSource
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode log sources: %w", err)
	}
	return out, nil
}

func (a *Agent) runRemoteLogSourcePoller(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(12 * time.Second):
	}

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		a.syncRemoteLogSources()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *Agent) syncRemoteLogSources() {
	reqCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sources, err := fetchRemoteLogSources(reqCtx, a.hubURL, a.apiKey)
	if err != nil {
		slog.Debug("Remote log source sync failed", "error", err)
		return
	}
	b, err := json.Marshal(sources)
	if err != nil {
		return
	}
	sig := string(b)

	a.logMu.Lock()
	defer a.logMu.Unlock()

	if sig == a.lastRemoteSourcesSig {
		return
	}
	a.lastRemoteSourcesSig = sig

	if a.logSubCancel != nil {
		a.logSubCancel()
		a.logSubCancel = nil
	}
	a.logs = nil

	if len(sources) == 0 {
		slog.Info("Remote log sources: none (or empty) for this agent; tailing stopped")
		return
	}

	subCtx, cancel := context.WithCancel(context.Background())
	a.logSubCancel = cancel

	sendLog := func(entry *models.LogEntry) {
		entry.ServerID = a.serverID
		if err := a.transport.SendLog(context.Background(), entry); err != nil {
			slog.Debug("Failed to send log entry", "error", err)
		}
	}
	sendAccess := func(entry *models.AccessLogEntry) {
		entry.ServerID = a.serverID
		if err := a.transport.SendAccessLog(context.Background(), entry); err != nil {
			slog.Debug("Failed to send access log entry", "error", err)
		}
	}

	a.logs = collector.NewLogCollector(sources, sendLog, sendAccess)
	a.logs.Start(subCtx)
	slog.Info("Remote log sources applied", "count", len(sources))
}
