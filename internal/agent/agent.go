package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/agent/collector"
	"github.com/tudorAbrudan/tracelog/internal/hub"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

type Transport interface {
	SendMetrics(ctx context.Context, m *models.SystemMetrics) error
	Close() error
}

type localTransport struct {
	hub *hub.Hub
}

func (t *localTransport) SendMetrics(ctx context.Context, m *models.SystemMetrics) error {
	return t.hub.IngestMetrics(ctx, m)
}

func (t *localTransport) Close() error { return nil }

type Option func(*Agent)

func WithLocalHub(h *hub.Hub) Option {
	return func(a *Agent) {
		a.transport = &localTransport{hub: h}
		a.serverID = "local"
	}
}

func WithRemoteHub(hubURL, apiKey string) Option {
	return func(a *Agent) {
		// TODO: implement WebSocket transport
		a.hubURL = hubURL
		a.apiKey = apiKey
		a.serverID = "remote"
	}
}

type Agent struct {
	cfg       *models.Config
	transport Transport
	system    *collector.SystemCollector
	serverID  string
	hubURL    string
	apiKey    string
}

func New(cfg *models.Config, opts ...Option) (*Agent, error) {
	a := &Agent{
		cfg: cfg,
	}

	for _, opt := range opts {
		opt(a)
	}

	if cfg.Collect.System {
		a.system = collector.NewSystemCollector()
	}

	return a, nil
}

func (a *Agent) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return a.Start(ctx)
}

func (a *Agent) Start(ctx context.Context) error {
	interval := time.Duration(a.cfg.Collect.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("Agent started", "interval", interval, "system", a.cfg.Collect.System, "docker", a.cfg.Collect.Docker)

	a.collectAndSend(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			a.collectAndSend(ctx)
		}
	}
}

func (a *Agent) collectAndSend(ctx context.Context) {
	if a.system != nil && a.transport != nil {
		metrics, err := a.system.Collect(ctx)
		if err != nil {
			slog.Error("Failed to collect system metrics", "error", err)
			return
		}
		metrics.ServerID = a.serverID
		if err := a.transport.SendMetrics(ctx, metrics); err != nil {
			slog.Error("Failed to send metrics", "error", err)
		}
	}
}

func (a *Agent) Shutdown() {
	if a.transport != nil {
		a.transport.Close()
	}
	slog.Info("Agent stopped")
}
