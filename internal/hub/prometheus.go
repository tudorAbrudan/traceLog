package hub

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/tudorAbrudan/tracelog/internal/hub/prom"
)

func (h *Hub) metricsToken() string {
	if h.cfg.MetricsToken != "" {
		return h.cfg.MetricsToken
	}
	return os.Getenv("TRACELOG_METRICS_TOKEN")
}

func (h *Hub) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	if tok := h.metricsToken(); tok != "" {
		if r.Header.Get("Authorization") != "Bearer "+tok && r.URL.Query().Get("token") != tok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	ctx := r.Context()
	servers, err := h.store.ListServers(ctx)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	online := 0
	for _, s := range servers {
		if s.Status == "online" {
			online++
		}
	}

	var dbSize int64
	if st, err := os.Stat(filepath.Join(h.cfg.DataDir, "tracelog.db")); err == nil {
		dbSize = st.Size()
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_ = prom.Render(w, prom.State{
		Version:       h.cfg.Version,
		ServersTotal:  len(servers),
		ServersOnline: online,
		AgentSessions: h.ws.connectedAgents(),
		DBSizeBytes:   dbSize,

		IngestSystem:  h.ingestSystem.Load(),
		IngestDocker:  h.ingestDocker.Load(),
		IngestLog:     h.ingestLog.Load(),
		IngestAccess:  h.ingestAccess.Load(),
		IngestProcess: h.ingestProcess.Load(),

		HTTPAPI:       h.httpAPI.Load(),
		HTTPDashboard: h.httpDashboard.Load(),
		HTTPHealth:    h.httpHealth.Load(),
		HTTPMetrics:   h.httpMetrics.Load(),
		HTTPWS:        h.httpWS.Load(),
		HTTPOther:     h.httpOther.Load(),
	})
}

func (h *Hub) httpMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		h.incrHTTPBucket(handlerBucket(r.URL.Path))
	})
}

func handlerBucket(path string) string {
	switch {
	case path == "/metrics":
		return "metrics"
	case strings.HasPrefix(path, "/api/health"):
		return "health"
	case strings.HasPrefix(path, "/api/ws/"):
		return "websocket"
	case strings.HasPrefix(path, "/api/"):
		return "api"
	case path == "/" || strings.HasPrefix(path, "/assets/"):
		return "dashboard"
	default:
		return "other"
	}
}

func (h *Hub) incrHTTPBucket(bucket string) {
	var a *atomic.Uint64
	switch bucket {
	case "api":
		a = &h.httpAPI
	case "dashboard":
		a = &h.httpDashboard
	case "health":
		a = &h.httpHealth
	case "metrics":
		a = &h.httpMetrics
	case "websocket":
		a = &h.httpWS
	default:
		a = &h.httpOther
	}
	a.Add(1)
}
