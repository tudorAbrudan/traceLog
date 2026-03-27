package agent

import (
	"fmt"
	"net/url"
	"strings"
)

// AgentLogSourcesURL builds the HTTPS GET URL for /api/agent/log-sources from the same base used for WebSocket
// (e.g. wss://host/tracelog → https://host/tracelog/api/agent/log-sources).
func AgentLogSourcesURL(hubBase string) (string, error) {
	raw := strings.TrimSpace(hubBase)
	if raw == "" {
		return "", fmt.Errorf("empty hub URL")
	}
	u, err := url.Parse(strings.TrimSuffix(raw, "/"))
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	default:
		return "", fmt.Errorf("agent hub URL must use ws:// or wss:// (got %q)", u.Scheme)
	}
	prefix := strings.TrimSuffix(u.Path, "/")
	suffix := "/api/agent/log-sources"
	if prefix == "" {
		u.Path = suffix
	} else {
		u.Path = prefix + suffix
	}
	return u.String(), nil
}
