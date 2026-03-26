# Configuration

TraceLog is configured through command-line flags and the web dashboard Settings page.

## Command-Line Flags

### Serve Mode (Hub + Agent)

```bash
tracelog serve [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8090` | HTTP port |
| `--bind` | `0.0.0.0` | Bind address |
| `--data` | `/var/lib/tracelog` or `~/.tracelog` | Data directory |
| `--metrics-token` | (empty) | If set, `/metrics` requires `Authorization: Bearer <token>` or `?token=` (see [Prometheus metrics](#prometheus-metrics)) |

### Hub Mode

```bash
tracelog hub [flags]
```

Same flags as serve mode.

## Prometheus metrics

TraceLog exposes a **Prometheus** text endpoint at **`GET /metrics`** (same port as the dashboard). No UI login is required; protect it in production.

**Metrics include:** build version, server counts, active agent WebSocket sessions, SQLite file size, cumulative ingest counters (system / docker / log / access / process), and HTTP request counts by handler bucket (`api`, `dashboard`, `health`, `metrics`, `websocket`, `other`).

### Optional authentication

If you set **`--metrics-token=SECRET`** or the environment variable **`TRACELOG_METRICS_TOKEN`**, scrapes must send:

- Header `Authorization: Bearer SECRET`, or  
- Query `http://host:8090/metrics?token=SECRET`

### Example `prometheus.yml`

```yaml
scrape_configs:
  - job_name: tracelog
    static_configs:
      - targets: ['127.0.0.1:8090']
    metrics_path: /metrics
    # If you use --metrics-token:
    # authorization:
    #   type: Bearer
    #   credentials: "your-secret-token"
```

### Agent Mode

```bash
tracelog agent [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--hub` | (required) | Hub base URL (e.g. `https://mon.example.com`) |
| `--key` | (required) | Server API key |

## Dashboard Settings

Access **Settings** from the sidebar. Short reference:

| Tab | What you configure | Effect |
|-----|--------------------|--------|
| **General** | Retention (1–30 days), collection interval | Hourly cleanup of old metrics, **ingested logs**, access logs, uptime, alerts, etc.; how often agents send system metrics. |
| **Log Sources** | Paths + format (plain / nginx / apache) | Agent tails files; lines are **stored in SQLite** (see [Logs & HTTP analytics](./logs-http-analytics.md)). Scan runs on the **hub host**. |
| **Notifications** | Email (SMTP JSON), webhooks | Alert delivery channels; test with **Test** on a channel. |
| **Servers** | Registered agents, API keys | Multi-server; deleting a server removes its metrics/logs rows for that `server_id`. |
| **Alerts** | Metric, threshold, duration, channel | Fires when conditions hold; uses notification channels. |
| **Account** | — | Password changes via CLI: `tracelog user reset-password …`. |
| **About** | — | Build **version** (from `/api/health` when released); link to docs. |

More detail: [Logs & HTTP analytics](./logs-http-analytics.md) (purge, blacklist, bad requests, WHOIS links).

## Data Storage

All data is stored in a SQLite database at `{data-dir}/tracelog.db` using WAL mode for concurrent reads.

### Retention

Data older than the configured retention period (default 30 days) is automatically cleaned up every hour. This includes:
- System metrics
- Docker metrics
- Ingested log lines (`logs` table)
- HTTP access log rows (`access_logs`)
- Uptime results
- Alert history
- Process metrics

This is separate from the **Logs** page **Purge** action, which removes stored lines immediately for a chosen server and optional age/source (see [Logs & HTTP analytics](./logs-http-analytics.md)).
