package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type WSTransport struct {
	hubURL   string
	apiKey   string
	conn     *websocket.Conn
	mu       sync.Mutex
	buffer   []Message
	maxBuf   int
	serverID string
}

type Message struct {
	Type string      `json:"type"`
	Data json.RawMessage `json:"data"`
}

func NewWSTransport(hubURL, apiKey string) *WSTransport {
	return &WSTransport{
		hubURL: hubURL,
		apiKey: apiKey,
		maxBuf: 1000,
	}
}

func (t *WSTransport) SetServerID(id string) {
	t.serverID = id
}

func (t *WSTransport) Connect(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/ws/agent", t.hubURL)

	headers := http.Header{}
	headers.Set("X-API-Key", t.apiKey)

	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return fmt.Errorf("connect to hub %s: %w", url, err)
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
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		return
	}
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
