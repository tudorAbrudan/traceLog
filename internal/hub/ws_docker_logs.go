package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

// ErrAgentNotConnected is returned when no WebSocket agent is registered for the server.
var ErrAgentNotConnected = errors.New("agent not connected for this server")

type dockerLogResult struct {
	logs string
	err  string
}

func randomRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// requestDockerLogsFromAgent asks a connected remote agent to run docker logs.
func (h *Hub) requestDockerLogsFromAgent(ctx context.Context, serverID, container string, tail int) (string, error) {
	h.ws.mu.RLock()
	ac := h.ws.agents[serverID]
	h.ws.mu.RUnlock()
	if ac == nil {
		return "", ErrAgentNotConnected
	}

	reqID := randomRequestID()
	ch := make(chan dockerLogResult, 1)

	h.dockerLogMu.Lock()
	h.dockerLogWaiters[reqID] = ch
	h.dockerLogMu.Unlock()

	defer func() {
		h.dockerLogMu.Lock()
		delete(h.dockerLogWaiters, reqID)
		h.dockerLogMu.Unlock()
	}()

	payload, err := json.Marshal(map[string]any{
		"request_id": reqID,
		"container":  container,
		"tail":       tail,
	})
	if err != nil {
		return "", err
	}
	msg := wsMessage{Type: "docker_logs_request", Data: payload}
	raw, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	ac.writeMu.Lock()
	werr := ac.conn.Write(ctx, websocket.MessageText, raw)
	ac.writeMu.Unlock()
	if werr != nil {
		return "", fmt.Errorf("send request to agent: %w", werr)
	}

	select {
	case res := <-ch:
		if res.err != "" {
			return "", fmt.Errorf("%s", res.err)
		}
		return res.logs, nil
	case <-time.After(28 * time.Second):
		return "", fmt.Errorf("timeout waiting for agent (28s)")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (h *Hub) deliverDockerLogResponse(reqID, logs, errMsg string) {
	h.dockerLogMu.Lock()
	ch, ok := h.dockerLogWaiters[reqID]
	delete(h.dockerLogWaiters, reqID)
	h.dockerLogMu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- dockerLogResult{logs: logs, err: errMsg}:
	default:
	}
}
