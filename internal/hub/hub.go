package hub

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/hub/alerts"
	"github.com/tudorAbrudan/tracelog/internal/hub/notify"
	"github.com/tudorAbrudan/tracelog/internal/hub/store"
	"github.com/tudorAbrudan/tracelog/internal/hub/uptime"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

type Hub struct {
	cfg         *models.Config
	store       *store.Store
	server      *http.Server
	mux         *http.ServeMux
	rateLimiter *loginRateLimiter
	ws          *wsHub
	alerts      *alerts.Engine
	uptime      *uptime.Checker
	notify      *notify.Manager

	ingestSystem, ingestDocker, ingestLog, ingestAccess, ingestProcess atomic.Uint64
	httpAPI, httpDashboard, httpHealth, httpMetrics, httpWS, httpOther  atomic.Uint64

	dockerLogMu      sync.Mutex
	dockerLogWaiters map[string]chan dockerLogResult

	spaOnce sync.Once
	spaH    http.Handler
}

func New(cfg *models.Config) (*Hub, error) {
	s, err := store.New(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("initialize store: %w", err)
	}

	notifyMgr := notify.NewManager()

	alertEngine := alerts.NewEngine(func(ctx context.Context, channelID string, alert *alerts.Alert) error {
		return notifyMgr.Send(ctx, channelID, "TraceLog Alert: "+alert.Metric, alert.Message)
	})

	uptimeChecker := uptime.NewChecker(func(result *uptime.Result) {
		if err := s.InsertUptimeResult(context.Background(), result); err != nil {
			slog.Error("Failed to store uptime result", "error", err)
		}
	})

	h := &Hub{
		cfg:              cfg,
		store:            s,
		mux:              http.NewServeMux(),
		rateLimiter:      newLoginRateLimiter(),
		ws:               newWSHub(),
		alerts:           alertEngine,
		uptime:           uptimeChecker,
		notify:           notifyMgr,
		dockerLogWaiters: make(map[string]chan dockerLogResult),
	}

	h.registerRoutes()
	return h, nil
}

func (h *Hub) Store() *store.Store {
	return h.store
}

func (h *Hub) IngestMetrics(ctx context.Context, m *models.SystemMetrics) error {
	if err := h.store.InsertMetrics(ctx, m); err != nil {
		return err
	}
	h.ingestSystem.Add(1)
	if h.alerts != nil {
		h.alerts.Evaluate(ctx, m.ServerID, m)
	}
	return nil
}

func (h *Hub) IngestDockerMetrics(ctx context.Context, metrics []models.DockerMetrics) error {
	for i := range metrics {
		if err := h.store.InsertDockerMetrics(ctx, &metrics[i]); err != nil {
			return err
		}
	}
	h.ingestDocker.Add(uint64(len(metrics)))
	return nil
}

func (h *Hub) IngestProcessMetrics(ctx context.Context, metrics []models.ProcessMetrics) error {
	if err := h.store.InsertProcessMetrics(ctx, metrics); err != nil {
		return err
	}
	h.ingestProcess.Add(uint64(len(metrics)))
	return nil
}

func (h *Hub) IngestAccessLog(ctx context.Context, entry *models.AccessLogEntry) error {
	if err := h.store.InsertAccessLog(ctx, entry); err != nil {
		return err
	}
	h.ingestAccess.Add(1)
	return nil
}

func (h *Hub) IngestLog(ctx context.Context, entry *models.LogEntry) error {
	if err := h.store.InsertLog(ctx, entry); err != nil {
		return err
	}
	h.ingestLog.Add(1)
	return nil
}

func (h *Hub) EnsureLocalServer(ctx context.Context) (string, error) {
	srv, err := h.store.EnsureLocalServer(ctx)
	if err != nil {
		return "", err
	}
	if err := h.store.UpdateServerStatus(ctx, srv.ID, "online"); err != nil {
		slog.Warn("EnsureLocalServer: update status", "error", err)
	}
	return srv.ID, nil
}

func (h *Hub) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return h.Start(ctx)
}

func (h *Hub) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", h.cfg.BindAddress, h.cfg.Port)

	h.server = &http.Server{
		Addr:         addr,
		Handler:      h.httpMetricsMiddleware(h.mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
	}

	go h.store.StartRetentionWorker(ctx)

	// Load and start saved uptime checks
	h.loadUptimeChecks(ctx)

	// Load alert rules
	h.loadAlertRules(ctx)

	// Load notification channels
	h.loadNotificationChannels(ctx)

	slog.Info("Hub listening", "addr", addr)

	if err := h.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("hub server: %w", err)
	}
	return nil
}

func (h *Hub) loadUptimeChecks(ctx context.Context) {
	checks, err := h.store.ListUptimeChecks(ctx)
	if err != nil {
		slog.Error("Failed to load uptime checks", "error", err)
		return
	}
	for _, c := range checks {
		c := c
		h.uptime.AddCheck(&c)
	}
	if len(checks) > 0 {
		slog.Info("Loaded uptime checks", "count", len(checks))
	}
}

func (h *Hub) loadAlertRules(ctx context.Context) {
	rules, err := h.store.ListAlertRules(ctx)
	if err != nil {
		slog.Error("Failed to load alert rules", "error", err)
		return
	}
	for _, r := range rules {
		r := r
		h.alerts.AddRule(&r)
	}
	if len(rules) > 0 {
		slog.Info("Loaded alert rules", "count", len(rules))
	}
}

func (h *Hub) loadNotificationChannels(ctx context.Context) {
	channels, err := h.store.ListNotificationChannels(ctx)
	if err != nil {
		slog.Error("Failed to load notification channels", "error", err)
		return
	}
	for _, ch := range channels {
		ch := ch
		h.notify.AddChannel(&ch)
	}
	if len(channels) > 0 {
		slog.Info("Loaded notification channels", "count", len(channels))
	}
}

func (h *Hub) Shutdown() {
	if h.uptime != nil {
		h.uptime.Stop()
	}
	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.server.Shutdown(ctx); err != nil {
			slog.Error("Hub shutdown error", "error", err)
		}
	}
	if h.store != nil {
		h.store.Close()
	}
	slog.Info("Hub stopped")
}

func (h *Hub) registerRoutes() {
	auth := h.authMiddleware
	csrf := h.csrfMiddleware

	// Public routes
	h.mux.HandleFunc("GET /metrics", h.handlePrometheusMetrics)
	h.mux.HandleFunc("GET /api/health", h.handleHealth)
	h.mux.HandleFunc("POST /api/auth/login", h.handleLogin)
	h.mux.HandleFunc("POST /api/auth/setup", h.handleSetup)

	// Protected routes (require session)
	h.mux.HandleFunc("POST /api/auth/logout", auth(h.handleLogout))
	h.mux.HandleFunc("GET /api/auth/me", auth(h.handleMe))
	h.mux.HandleFunc("POST /api/database/export", auth(csrf(h.handleDatabaseExport)))

	// Servers
	h.mux.HandleFunc("GET /api/servers", auth(h.handleListServers))
	h.mux.HandleFunc("GET /api/servers/{id}", auth(h.handleGetServer))
	h.mux.HandleFunc("GET /api/servers/{id}/metrics", auth(h.handleGetMetrics))
	h.mux.HandleFunc("GET /api/servers/{id}/docker", auth(h.handleGetDockerMetrics))
	h.mux.HandleFunc("GET /api/servers/{id}/docker/logs", auth(h.handleDockerContainerLogs))
	h.mux.HandleFunc("GET /api/servers/{id}/processes", auth(h.handleGetProcesses))
	h.mux.HandleFunc("POST /api/servers", auth(csrf(h.handleCreateServer)))
	h.mux.HandleFunc("DELETE /api/servers/{id}", auth(csrf(h.handleDeleteServer)))

	// Settings
	h.mux.HandleFunc("GET /api/settings", auth(h.handleGetSettings))
	h.mux.HandleFunc("PUT /api/settings", auth(csrf(h.handleUpdateSettings)))

	// Log sources
	h.mux.HandleFunc("GET /api/log-sources", auth(h.handleListLogSources))
	h.mux.HandleFunc("POST /api/log-sources", auth(csrf(h.handleCreateLogSource)))
	h.mux.HandleFunc("DELETE /api/log-sources/{id}", auth(csrf(h.handleDeleteLogSource)))

	// Logs
	h.mux.HandleFunc("GET /api/logs", auth(h.handleGetLogs))
	h.mux.HandleFunc("POST /api/logs/purge", auth(csrf(h.handlePurgeLogs)))

	// Access Logs / HTTP Analytics
	h.mux.HandleFunc("GET /api/servers/{id}/access-stats", auth(h.handleAccessLogStats))
	h.mux.HandleFunc("GET /api/servers/{id}/access-logs", auth(h.handleRecentAccessLogs))
	h.mux.HandleFunc("GET /api/servers/{id}/access-bad-requests", auth(h.handleAccessBadRequests))
	h.mux.HandleFunc("GET /api/access-ip-policy", auth(h.handleAccessIPPolicy))
	h.mux.HandleFunc("PUT /api/access-ip-policy", auth(csrf(h.handleAccessIPPolicy)))

	// Uptime checks
	h.mux.HandleFunc("GET /api/uptime", auth(h.handleListUptimeChecks))
	h.mux.HandleFunc("POST /api/uptime", auth(csrf(h.handleCreateUptimeCheck)))
	h.mux.HandleFunc("DELETE /api/uptime/{id}", auth(csrf(h.handleDeleteUptimeCheck)))
	h.mux.HandleFunc("GET /api/uptime/{id}/results", auth(h.handleGetUptimeResults))

	// Alert rules
	h.mux.HandleFunc("GET /api/alerts", auth(h.handleListAlertRules))
	h.mux.HandleFunc("POST /api/alerts", auth(csrf(h.handleCreateAlertRule)))
	h.mux.HandleFunc("DELETE /api/alerts/{id}", auth(csrf(h.handleDeleteAlertRule)))

	// Notification channels
	h.mux.HandleFunc("GET /api/notifications", auth(h.handleListNotificationChannels))
	h.mux.HandleFunc("POST /api/notifications", auth(csrf(h.handleCreateNotificationChannel)))
	h.mux.HandleFunc("DELETE /api/notifications/{id}", auth(csrf(h.handleDeleteNotificationChannel)))
	h.mux.HandleFunc("POST /api/notifications/{id}/test", auth(csrf(h.handleTestNotificationChannel)))

	// WebSocket for agent connections
	h.mux.HandleFunc("GET /api/ws/agent", h.handleAgentWS)

	// Detection
	h.mux.HandleFunc("GET /api/detect", auth(h.handleDetect))

	// Dashboard SPA
	h.mux.HandleFunc("GET /", h.handleDashboard)
}

func (h *Hub) dashboardSPA() http.Handler {
	h.spaOnce.Do(func() {
		prefix := models.NormalizeURLPathPrefix(h.cfg.URLPathPrefix)
		h.spaH = NewSPAHandler(prefix)
	})
	return h.spaH
}
