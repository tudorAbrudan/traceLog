package agent

import "testing"

func TestAgentLogSourcesURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"wss://example.com/tracelog", "https://example.com/tracelog/api/agent/log-sources"},
		{"ws://127.0.0.1:8090", "http://127.0.0.1:8090/api/agent/log-sources"},
		{"wss://x/", "https://x/api/agent/log-sources"},
	}
	for _, tc := range tests {
		got, err := AgentLogSourcesURL(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("%q -> got %q want %q", tc.in, got, tc.want)
		}
	}
}
