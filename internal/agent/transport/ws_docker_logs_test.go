package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func runHubDockerLogsTestServer(t *testing.T, handler func(context.Context, *websocket.Conn)) string {
	t.Helper()
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ws/agent", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("Accept: %v", err)
			return
		}
		defer func() { _ = c.CloseNow() }()
		handler(ctx, c)
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

func TestDockerLogsRequest_InvalidContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan struct{})
	wsURL := runHubDockerLogsTestServer(t, func(bg context.Context, hubConn *websocket.Conn) {
		defer close(done)
		rctx, rcancel := context.WithTimeout(bg, 10*time.Second)
		defer rcancel()
		payload, err := json.Marshal(map[string]any{
			"request_id": "rid-1",
			"container":  "",
			"tail":       10,
		})
		if err != nil {
			t.Errorf("marshal payload: %v", err)
			return
		}
		raw, err := json.Marshal(Message{Type: "docker_logs_request", Data: payload})
		if err != nil {
			t.Errorf("marshal msg: %v", err)
			return
		}
		if err := hubConn.Write(rctx, websocket.MessageText, raw); err != nil {
			t.Errorf("hub Write: %v", err)
			return
		}
		_, data, err := hubConn.Read(rctx)
		if err != nil {
			t.Errorf("hub Read: %v", err)
			return
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Errorf("unmarshal response: %v", err)
			return
		}
		if msg.Type != "docker_logs_response" {
			t.Errorf("want docker_logs_response, got %q", msg.Type)
			return
		}
		var body struct {
			RequestID string `json:"request_id"`
			Logs      string `json:"logs"`
			Error     string `json:"error"`
		}
		if err := json.Unmarshal(msg.Data, &body); err != nil {
			t.Errorf("unmarshal body: %v", err)
			return
		}
		if body.RequestID != "rid-1" {
			t.Errorf("request_id %q", body.RequestID)
		}
		if body.Error == "" {
			t.Error("expected error for empty container")
		}
	})

	wt := NewWSTransport(wsURL, "test-key")
	if err := wt.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	go wt.readLoop(ctx)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout waiting for hub simulator")
	}
}

func TestDockerLogsRequest_SemaphoreFull(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan struct{})
	wsURL := runHubDockerLogsTestServer(t, func(bg context.Context, hubConn *websocket.Conn) {
		defer close(done)
		rctx, rcancel := context.WithTimeout(bg, 10*time.Second)
		defer rcancel()
		payload, err := json.Marshal(map[string]any{
			"request_id": "rid-busy",
			"container":  "",
			"tail":       10,
		})
		if err != nil {
			t.Errorf("marshal payload: %v", err)
			return
		}
		raw, err := json.Marshal(Message{Type: "docker_logs_request", Data: payload})
		if err != nil {
			t.Errorf("marshal msg: %v", err)
			return
		}
		if err := hubConn.Write(rctx, websocket.MessageText, raw); err != nil {
			t.Errorf("hub Write: %v", err)
			return
		}
		_, data, err := hubConn.Read(rctx)
		if err != nil {
			t.Errorf("hub Read: %v", err)
			return
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Errorf("unmarshal response: %v", err)
			return
		}
		var body struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(msg.Data, &body); err != nil {
			t.Errorf("unmarshal body: %v", err)
			return
		}
		if body.Error != "too many concurrent docker log requests" {
			t.Errorf("error: %q", body.Error)
		}
	})

	wt := NewWSTransport(wsURL, "test-key")
	wt.dockerLogsSem = make(chan struct{}, 1)
	wt.dockerLogsSem <- struct{}{}
	if err := wt.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	go wt.readLoop(ctx)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout waiting for hub simulator")
	}
	<-wt.dockerLogsSem
}
