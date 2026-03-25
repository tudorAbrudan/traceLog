package hub

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/tudorAbrudan/tracelog/internal/hub/store"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

type Hub struct {
	cfg    *models.Config
	store  *store.Store
	server *http.Server
	mux    *http.ServeMux
}

func New(cfg *models.Config) (*Hub, error) {
	s, err := store.New(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("initialize store: %w", err)
	}

	h := &Hub{
		cfg:   cfg,
		store: s,
		mux:   http.NewServeMux(),
	}

	h.registerRoutes()
	return h, nil
}

func (h *Hub) Store() *store.Store {
	return h.store
}

func (h *Hub) IngestMetrics(ctx context.Context, m *models.SystemMetrics) error {
	return h.store.InsertMetrics(ctx, m)
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
		Handler:      h.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
	}

	go h.store.StartRetentionWorker(ctx)

	slog.Info("Hub listening", "addr", addr)

	if err := h.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("hub server: %w", err)
	}
	return nil
}

func (h *Hub) Shutdown() {
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
	h.mux.HandleFunc("GET /api/health", h.handleHealth)
	h.mux.HandleFunc("GET /api/servers", h.handleListServers)
	h.mux.HandleFunc("GET /api/servers/{id}", h.handleGetServer)
	h.mux.HandleFunc("GET /api/servers/{id}/metrics", h.handleGetMetrics)
	h.mux.HandleFunc("POST /api/servers", h.handleCreateServer)

	h.mux.HandleFunc("POST /api/auth/login", h.handleLogin)
	h.mux.HandleFunc("POST /api/auth/logout", h.handleLogout)
	h.mux.HandleFunc("GET /api/auth/me", h.handleMe)

	h.mux.HandleFunc("GET /api/settings", h.handleGetSettings)
	h.mux.HandleFunc("PUT /api/settings", h.handleUpdateSettings)
	h.mux.HandleFunc("GET /api/settings/log-sources", h.handleGetLogSources)
	h.mux.HandleFunc("POST /api/settings/log-sources", h.handleCreateLogSource)

	// TODO: WebSocket endpoint for agents and dashboard
	// TODO: Serve embedded Svelte dashboard for all non-API routes
}
