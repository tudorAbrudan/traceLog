package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/agent/collector"
	"github.com/tudorAbrudan/tracelog/internal/agent/transport"
	"github.com/tudorAbrudan/tracelog/internal/hub"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

type Transport interface {
	SendMetrics(ctx context.Context, m *models.SystemMetrics) error
	SendDockerMetrics(ctx context.Context, metrics []models.DockerMetrics) error
	SendProcessMetrics(ctx context.Context, metrics []models.ProcessMetrics) error
	SendLog(ctx context.Context, entry *models.LogEntry) error
	SendAccessLog(ctx context.Context, entry *models.AccessLogEntry) error
	Close() error
}

type localTransport struct {
	hub *hub.Hub
}

func (t *localTransport) SendMetrics(ctx context.Context, m *models.SystemMetrics) error {
	return t.hub.IngestMetrics(ctx, m)
}

func (t *localTransport) SendDockerMetrics(ctx context.Context, metrics []models.DockerMetrics) error {
	return t.hub.IngestDockerMetrics(ctx, metrics)
}

func (t *localTransport) SendProcessMetrics(ctx context.Context, metrics []models.ProcessMetrics) error {
	return t.hub.IngestProcessMetrics(ctx, metrics)
}

func (t *localTransport) SendAccessLog(ctx context.Context, entry *models.AccessLogEntry) error {
	return t.hub.IngestAccessLog(ctx, entry)
}

func (t *localTransport) SendLog(ctx context.Context, entry *models.LogEntry) error {
	return t.hub.IngestLog(ctx, entry)
}

func (t *localTransport) Close() error { return nil }

type Option func(*Agent)

func WithLocalHub(h *hub.Hub) Option {
	return func(a *Agent) {
		a.transport = &localTransport{hub: h}
		serverID, err := h.EnsureLocalServer(context.Background())
		if err != nil {
			slog.Error("Failed to ensure local server", "error", err)
			a.serverID = "local"
		} else {
			a.serverID = serverID
		}
	}
}

func WithRemoteHub(hubURL, apiKey string) Option {
	return func(a *Agent) {
		wt := transport.NewWSTransport(hubURL, apiKey)
		a.transport = &wsTransportAdapter{wt: wt}
		a.hubURL = hubURL
		a.apiKey = apiKey
		a.serverID = "remote"
		a.wsTransport = wt
	}
}

type wsTransportAdapter struct {
	wt *transport.WSTransport
}

func (w *wsTransportAdapter) SendMetrics(ctx context.Context, m *models.SystemMetrics) error {
	return w.wt.SendMetrics(ctx, m)
}

func (w *wsTransportAdapter) SendDockerMetrics(ctx context.Context, metrics []models.DockerMetrics) error {
	return w.wt.SendDockerMetrics(ctx, metrics)
}

func (w *wsTransportAdapter) SendProcessMetrics(ctx context.Context, metrics []models.ProcessMetrics) error {
	return w.wt.SendProcessMetrics(ctx, metrics)
}

func (w *wsTransportAdapter) SendAccessLog(ctx context.Context, entry *models.AccessLogEntry) error {
	return w.wt.SendAccessLog(ctx, entry)
}

func (w *wsTransportAdapter) SendLog(ctx context.Context, entry *models.LogEntry) error {
	return w.wt.SendLog(ctx, entry)
}

func (w *wsTransportAdapter) Close() error {
	return w.wt.Close()
}

type Agent struct {
	cfg         *models.Config
	transport   Transport
	system      *collector.SystemCollector
	docker      *collector.DockerCollector
	process     *collector.ProcessCollector
	logs        *collector.LogCollector
	wsTransport *transport.WSTransport
	serverID    string
	hubURL      string
	apiKey      string
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

	if cfg.Collect.Docker {
		a.docker = collector.NewDockerCollector()
	}

	if cfg.Collect.Processes {
		a.process = collector.NewProcessCollector()
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

	if a.wsTransport != nil {
		go a.wsTransport.ConnectWithRetry(ctx)
	}

	// Start log collectors
	if len(a.cfg.Collect.LogSources) > 0 && a.transport != nil {
		a.logs = collector.NewLogCollector(a.cfg.Collect.LogSources, func(entry *models.LogEntry) {
			entry.ServerID = a.serverID
			if err := a.transport.SendLog(ctx, entry); err != nil {
				slog.Debug("Failed to send log entry", "error", err)
			}
		}, func(entry *models.AccessLogEntry) {
			entry.ServerID = a.serverID
			if err := a.transport.SendAccessLog(ctx, entry); err != nil {
				slog.Debug("Failed to send access log entry", "error", err)
			}
		})
		a.logs.Start(ctx)
		slog.Info("Log collectors started", "sources", len(a.cfg.Collect.LogSources))
	}

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
	if a.transport == nil {
		return
	}

	if a.system != nil {
		metrics, err := a.system.Collect(ctx)
		if err != nil {
			slog.Error("Failed to collect system metrics", "error", err)
		} else {
			metrics.ServerID = a.serverID
			if err := a.transport.SendMetrics(ctx, metrics); err != nil {
				slog.Error("Failed to send system metrics", "error", err)
			}
		}
	}

	if a.docker != nil {
		metrics, err := a.docker.Collect(ctx)
		if err != nil {
			slog.Debug("Failed to collect docker metrics", "error", err)
		} else if len(metrics) > 0 {
			for i := range metrics {
				metrics[i].ServerID = a.serverID
			}
			if err := a.transport.SendDockerMetrics(ctx, metrics); err != nil {
				slog.Error("Failed to send docker metrics", "error", err)
			}
		}
	}

	if a.process != nil {
		metrics, err := a.process.Collect(ctx)
		if err != nil {
			slog.Debug("Failed to collect process metrics", "error", err)
		} else if len(metrics) > 0 {
			for i := range metrics {
				metrics[i].ServerID = a.serverID
			}
			if err := a.transport.SendProcessMetrics(ctx, metrics); err != nil {
				slog.Error("Failed to send process metrics", "error", err)
			}
		}
	}
}

func (a *Agent) Shutdown() {
	if a.transport != nil {
		a.transport.Close()
	}
	slog.Info("Agent stopped")
}
