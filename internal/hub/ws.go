package hub

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/websocket"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type agentConn struct {
	serverID string
	conn     *websocket.Conn
}

type wsHub struct {
	mu     sync.RWMutex
	agents map[string]*agentConn
}

func newWSHub() *wsHub {
	return &wsHub{
		agents: make(map[string]*agentConn),
	}
}

type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (h *Hub) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	server, err := h.store.GetServerByAPIKey(r.Context(), apiKey)
	if err != nil {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("WebSocket accept failed", "error", err)
		return
	}
	defer conn.CloseNow()

	h.store.UpdateServerStatus(r.Context(), server.ID, "online")
	slog.Info("Agent connected", "server", server.Name, "id", server.ID)

	h.ws.mu.Lock()
	h.ws.agents[server.ID] = &agentConn{serverID: server.ID, conn: conn}
	h.ws.mu.Unlock()

	defer func() {
		h.ws.mu.Lock()
		delete(h.ws.agents, server.ID)
		h.ws.mu.Unlock()
		h.store.UpdateServerStatus(context.Background(), server.ID, "offline")
		slog.Info("Agent disconnected", "server", server.Name)
	}()

	ctx := r.Context()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			slog.Warn("Agent read error", "server", server.Name, "error", err)
			return
		}

		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			slog.Warn("Invalid message from agent", "error", err)
			continue
		}

		h.processAgentMessage(ctx, server.ID, &msg)
	}
}

func (h *Hub) processAgentMessage(ctx context.Context, serverID string, msg *wsMessage) {
	switch msg.Type {
	case "metrics":
		var m models.SystemMetrics
		if err := json.Unmarshal(msg.Data, &m); err != nil {
			slog.Warn("Invalid metrics", "error", err)
			return
		}
		m.ServerID = serverID
		if err := h.store.InsertMetrics(ctx, &m); err != nil {
			slog.Error("Failed to store metrics", "error", err)
		}

	case "docker_metrics":
		var metrics []models.DockerMetrics
		if err := json.Unmarshal(msg.Data, &metrics); err != nil {
			slog.Warn("Invalid docker metrics", "error", err)
			return
		}
		for i := range metrics {
			metrics[i].ServerID = serverID
			if err := h.store.InsertDockerMetrics(ctx, &metrics[i]); err != nil {
				slog.Error("Failed to store docker metrics", "error", err)
			}
		}

	case "process_metrics":
		var metrics []models.ProcessMetrics
		if err := json.Unmarshal(msg.Data, &metrics); err != nil {
			slog.Warn("Invalid process metrics", "error", err)
			return
		}
		for i := range metrics {
			metrics[i].ServerID = serverID
		}
		if err := h.store.InsertProcessMetrics(ctx, metrics); err != nil {
			slog.Error("Failed to store process metrics", "error", err)
		}

	case "access_log":
		var entry models.AccessLogEntry
		if err := json.Unmarshal(msg.Data, &entry); err != nil {
			slog.Warn("Invalid access log entry", "error", err)
			return
		}
		entry.ServerID = serverID
		if err := h.store.InsertAccessLog(ctx, &entry); err != nil {
			slog.Error("Failed to store access log entry", "error", err)
		}

	case "log":
		var entry models.LogEntry
		if err := json.Unmarshal(msg.Data, &entry); err != nil {
			slog.Warn("Invalid log entry", "error", err)
			return
		}
		entry.ServerID = serverID
		if err := h.store.InsertLog(ctx, &entry); err != nil {
			slog.Error("Failed to store log entry", "error", err)
		}

	default:
		slog.Warn("Unknown message type from agent", "type", msg.Type)
	}
}
