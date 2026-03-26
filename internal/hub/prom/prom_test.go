package prom

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	var b strings.Builder
	err := Render(&b, State{
		Version:       `v1.0.0"test`,
		ServersTotal:  2,
		ServersOnline: 1,
		AgentSessions: 0,
		DBSizeBytes:   4096,
		IngestSystem:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, `tracelog_up 1`) {
		t.Fatalf("missing tracelog_up: %s", out)
	}
	if !strings.Contains(out, `tracelog_servers_total 2`) {
		t.Fatal(out)
	}
	if !strings.Contains(out, "v1.0.0\\\"test") {
		t.Fatalf("label not escaped: %s", out)
	}
}

func TestEscapeLabel(t *testing.T) {
	if escapeLabel(`a"b\c`) != `a\"b\\c` {
		t.Fatal(escapeLabel(`a"b\c`))
	}
}
