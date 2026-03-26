package hub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func testHubDockerLogs(t *testing.T) *Hub {
	t.Helper()
	return &Hub{
		ws:               newWSHub(),
		dockerLogWaiters: make(map[string]chan dockerLogResult),
	}
}

func TestErrAgentNotConnected(t *testing.T) {
	h := testHubDockerLogs(t)
	_, err := h.requestDockerLogsFromAgent(context.Background(), "missing-server", "c1", 10)
	if !errors.Is(err, ErrAgentNotConnected) {
		t.Fatalf("got %v, want ErrAgentNotConnected", err)
	}
}

func TestDeliverDockerLogResponse(t *testing.T) {
	h := testHubDockerLogs(t)
	ch := make(chan dockerLogResult, 1)
	h.dockerLogMu.Lock()
	h.dockerLogWaiters["req-1"] = ch
	h.dockerLogMu.Unlock()

	h.deliverDockerLogResponse("req-1", "hello\n", "")
	select {
	case r := <-ch:
		if r.logs != "hello\n" || r.err != "" {
			t.Fatalf("unexpected result: %+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestDeliverDockerLogResponse_ErrorString(t *testing.T) {
	h := testHubDockerLogs(t)
	ch := make(chan dockerLogResult, 1)
	h.dockerLogMu.Lock()
	h.dockerLogWaiters["req-2"] = ch
	h.dockerLogMu.Unlock()

	h.deliverDockerLogResponse("req-2", "", "boom")
	r := <-ch
	if r.err != "boom" || r.logs != "" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestDeliverDockerLogResponse_UnknownID(t *testing.T) {
	h := testHubDockerLogs(t)
	h.deliverDockerLogResponse("no-such", "x", "") // must not panic
}

func TestRequestDockerLogsFromAgent_RoundTrip(t *testing.T) {
	h := testHubDockerLogs(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ready := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ws/agent", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("Accept: %v", err)
			return
		}
		defer func() { _ = c.CloseNow() }()
		h.ws.mu.Lock()
		h.ws.agents["srv1"] = &agentConn{serverID: "srv1", conn: c}
		h.ws.mu.Unlock()
		close(ready)

		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			var msg wsMessage
			if json.Unmarshal(data, &msg) != nil {
				continue
			}
			h.processAgentMessage(context.Background(), "srv1", &msg)
		}
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	go func() {
		client, resp, err := websocket.Dial(ctx, wsURL+"/api/ws/agent", nil)
		if err != nil {
			t.Errorf("Dial: %v", err)
			return
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		defer func() { _ = client.CloseNow() }()
		<-ready
		_, data, err := client.Read(ctx)
		if err != nil {
			t.Errorf("client Read: %v", err)
			return
		}
		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Errorf("unmarshal: %v", err)
			return
		}
		if msg.Type != "docker_logs_request" {
			t.Errorf("want docker_logs_request, got %q", msg.Type)
			return
		}
		var pl struct {
			RequestID string `json:"request_id"`
			Container string `json:"container"`
			Tail      int    `json:"tail"`
		}
		if err := json.Unmarshal(msg.Data, &pl); err != nil {
			t.Errorf("payload: %v", err)
			return
		}
		respBody, err := json.Marshal(map[string]any{
			"request_id": pl.RequestID,
			"logs":       "line-a\nline-b\n",
			"error":      "",
		})
		if err != nil {
			t.Errorf("marshal resp body: %v", err)
			return
		}
		raw, err := json.Marshal(wsMessage{Type: "docker_logs_response", Data: respBody})
		if err != nil {
			t.Errorf("marshal msg: %v", err)
			return
		}
		if err := client.Write(ctx, websocket.MessageText, raw); err != nil {
			t.Errorf("client Write: %v", err)
		}
	}()

	<-ready

	out, err := h.requestDockerLogsFromAgent(ctx, "srv1", "nginx", 50)
	if err != nil {
		t.Fatal(err)
	}
	if out != "line-a\nline-b\n" {
		t.Fatalf("logs: %q", out)
	}
}
