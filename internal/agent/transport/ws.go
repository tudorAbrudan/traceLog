package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/tudorAbrudan/tracelog/internal/hub/dockerlogs"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

// Limits concurrent docker logs fetches so many UI requests cannot fork unbounded "docker logs" processes.
const maxConcurrentDockerLogRequests = 8

type WSTransport struct {
	hubURL   string
	apiKey   string
	conn     *websocket.Conn
	mu       sync.Mutex
	buffer   []Message
	maxBuf   int
	serverID string

	dockerLogsSem chan struct{} // acquire before running docker logs for hub
}

type Message struct {
	Type string      `json:"type"`
	Data json.RawMessage `json:"data"`
}

func NewWSTransport(hubURL, apiKey string) *WSTransport {
	return &WSTransport{
		hubURL:        hubURL,
		apiKey:        apiKey,
		maxBuf:        1000,
		dockerLogsSem: make(chan struct{}, maxConcurrentDockerLogRequests),
	}
}

func (t *WSTransport) SetServerID(id string) {
	t.serverID = id
}

func (t *WSTransport) Connect(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/ws/agent", strings.TrimSuffix(t.hubURL, "/"))

	headers := http.Header{}
	headers.Set("X-API-Key", t.apiKey)

	conn, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return fmt.Errorf("connect to hub %s: %w", url, err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	t.mu.Lock()
	t.conn = conn
	t.mu.Unlock()

	slog.Info("Connected to hub", "url", url)

	if len(t.buffer) > 0 {
		t.flushBuffer(ctx)
	}

	return nil
}

func (t *WSTransport) ConnectWithRetry(ctx context.Context) {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := t.Connect(ctx); err != nil {
			slog.Warn("Failed to connect to hub, retrying...", "error", err, "backoff", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		backoff = time.Second
		t.readLoop(ctx)
		t.mu.Lock()
		if t.conn != nil {
			_ = t.conn.Close(websocket.StatusGoingAway, "reconnecting")
			t.conn = nil
		}
		t.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (t *WSTransport) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		t.mu.Lock()
		c := t.conn
		t.mu.Unlock()
		if c == nil {
			return
		}
		_, data, err := c.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("Hub read ended, reconnecting", "error", err)
			return
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			slog.Warn("Invalid message from hub", "error", err)
			continue
		}
		switch msg.Type {
		case "docker_logs_request":
			go t.handleDockerLogsRequest(msg.Data)
		default:
			slog.Debug("Unknown hub message type", "type", msg.Type)
		}
	}
}

func (t *WSTransport) sendDockerLogsResponse(requestID, logs, errStr string) {
	resp, mErr := json.Marshal(map[string]any{
		"request_id": requestID,
		"logs":       logs,
		"error":      errStr,
	})
	if mErr != nil {
		return
	}
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer sendCancel()
	if sErr := t.send(sendCtx, Message{Type: "docker_logs_response", Data: resp}); sErr != nil {
		slog.Warn("docker_logs_response send failed", "error", sErr)
	}
}

func (t *WSTransport) handleDockerLogsRequest(raw json.RawMessage) {
	var payload struct {
		RequestID string `json:"request_id"`
		Container string `json:"container"`
		Tail      int    `json:"tail"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		slog.Warn("docker_logs_request: invalid payload", "error", err)
		return
	}
	select {
	case t.dockerLogsSem <- struct{}{}:
		defer func() { <-t.dockerLogsSem }()
	default:
		t.sendDockerLogsResponse(payload.RequestID, "", "too many concurrent docker log requests")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 32*time.Second)
	defer cancel()
	logs, err := dockerlogs.Fetch(ctx, payload.Container, payload.Tail)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	t.sendDockerLogsResponse(payload.RequestID, logs, errStr)
}

func (t *WSTransport) SendMetrics(ctx context.Context, m *models.SystemMetrics) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	msg := Message{Type: "metrics", Data: data}
	return t.send(ctx, msg)
}

func (t *WSTransport) SendDockerMetrics(ctx context.Context, metrics []models.DockerMetrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return t.send(ctx, Message{Type: "docker_metrics", Data: data})
}

func (t *WSTransport) SendProcessMetrics(ctx context.Context, metrics []models.ProcessMetrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return t.send(ctx, Message{Type: "process_metrics", Data: data})
}

func (t *WSTransport) SendAccessLog(ctx context.Context, entry *models.AccessLogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return t.send(ctx, Message{Type: "access_log", Data: data})
}

func (t *WSTransport) SendLog(ctx context.Context, entry *models.LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return t.send(ctx, Message{Type: "log", Data: data})
}

func (t *WSTransport) send(ctx context.Context, msg Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		if len(t.buffer) < t.maxBuf {
			t.buffer = append(t.buffer, msg)
		}
		return fmt.Errorf("not connected, buffered (%d messages)", len(t.buffer))
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := t.conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.conn = nil
		if len(t.buffer) < t.maxBuf {
			t.buffer = append(t.buffer, msg)
		}
		return fmt.Errorf("send failed, buffered: %w", err)
	}

	return nil
}

func (t *WSTransport) flushBuffer(ctx context.Context) {
	t.mu.Lock()
	buf := t.buffer
	t.buffer = nil
	t.mu.Unlock()

	for _, msg := range buf {
		data, _ := json.Marshal(msg)
		if t.conn != nil {
			if err := t.conn.Write(ctx, websocket.MessageText, data); err != nil {
				slog.Warn("Failed to flush buffered message", "error", err)
				return
			}
		}
	}
	if len(buf) > 0 {
		slog.Info("Flushed buffered messages", "count", len(buf))
	}
}

func (t *WSTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn != nil {
		return t.conn.Close(websocket.StatusNormalClosure, "agent shutting down")
	}
	return nil
}
