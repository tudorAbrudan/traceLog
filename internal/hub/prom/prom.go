// Package prom renders Prometheus text exposition format (no external deps).
package prom

import (
	"fmt"
	"io"
	"strings"
)

// State is a point-in-time snapshot for scraping.
type State struct {
	Version       string
	ServersTotal  int
	ServersOnline int
	AgentSessions int
	DBSizeBytes   int64

	IngestSystem  uint64
	IngestDocker  uint64
	IngestLog     uint64
	IngestAccess  uint64
	IngestProcess uint64

	HTTPAPI       uint64
	HTTPDashboard uint64
	HTTPHealth    uint64
	HTTPMetrics   uint64
	HTTPWS        uint64
	HTTPOther     uint64
}

// Render writes Prometheus 0.0.4 text format.
func Render(w io.Writer, s State) error {
	v := escapeLabel(s.Version)
	b := &strings.Builder{}
	write := func(format string, args ...any) {
		fmt.Fprintf(b, format, args...)
	}

	write("# HELP tracelog_build_info TraceLog version as an info metric.\n")
	write("# TYPE tracelog_build_info gauge\n")
	write("tracelog_build_info{version=\"%s\"} 1\n", v)

	write("# HELP tracelog_up The hub process is up.\n")
	write("# TYPE tracelog_up gauge\n")
	write("tracelog_up 1\n")

	write("# HELP tracelog_servers_total Registered servers.\n")
	write("# TYPE tracelog_servers_total gauge\n")
	write("tracelog_servers_total %d\n", s.ServersTotal)

	write("# HELP tracelog_servers_online Servers currently marked online.\n")
	write("# TYPE tracelog_servers_online gauge\n")
	write("tracelog_servers_online %d\n", s.ServersOnline)

	write("# HELP tracelog_agent_websocket_connections Active agent WebSocket sessions.\n")
	write("# TYPE tracelog_agent_websocket_connections gauge\n")
	write("tracelog_agent_websocket_connections %d\n", s.AgentSessions)

	write("# HELP tracelog_database_size_bytes SQLite database file size.\n")
	write("# TYPE tracelog_database_size_bytes gauge\n")
	write("tracelog_database_size_bytes %d\n", s.DBSizeBytes)

	write("# HELP tracelog_ingest_total Samples ingested by the hub (cumulative).\n")
	write("# TYPE tracelog_ingest_total counter\n")
	write("tracelog_ingest_total{type=\"system\"} %d\n", s.IngestSystem)
	write("tracelog_ingest_total{type=\"docker\"} %d\n", s.IngestDocker)
	write("tracelog_ingest_total{type=\"log\"} %d\n", s.IngestLog)
	write("tracelog_ingest_total{type=\"access\"} %d\n", s.IngestAccess)
	write("tracelog_ingest_total{type=\"process\"} %d\n", s.IngestProcess)

	write("# HELP tracelog_http_requests_total HTTP requests handled by the hub.\n")
	write("# TYPE tracelog_http_requests_total counter\n")
	write("tracelog_http_requests_total{handler=\"api\"} %d\n", s.HTTPAPI)
	write("tracelog_http_requests_total{handler=\"dashboard\"} %d\n", s.HTTPDashboard)
	write("tracelog_http_requests_total{handler=\"health\"} %d\n", s.HTTPHealth)
	write("tracelog_http_requests_total{handler=\"metrics\"} %d\n", s.HTTPMetrics)
	write("tracelog_http_requests_total{handler=\"websocket\"} %d\n", s.HTTPWS)
	write("tracelog_http_requests_total{handler=\"other\"} %d\n", s.HTTPOther)

	_, err := io.WriteString(w, b.String())
	return err
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
