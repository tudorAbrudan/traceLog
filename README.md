# TraceLog

**Lightweight server monitoring in a single binary.** Track CPU, RAM, disk, network, Docker containers, logs, and uptime — with a beautiful dark-mode dashboard and zero dependencies.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **System Metrics** — CPU, memory, disk, network, load average, uptime
- **Docker Monitoring** — Container CPU, memory, network; view **container logs** from the server detail page (local server)
- **Log Aggregation** — Tail any log file, parse nginx/apache access logs, search & filter
- **Uptime Monitoring** — HTTP endpoint checks with response time tracking
- **Alerts** — Configurable threshold rules with email (SMTP) and webhook notifications
- **Beautiful Dashboard** — Dark mode, responsive, real-time charts with uPlot
- **Single Binary** — No runtime dependencies, embeds the web UI
- **Easy Install** — One-line installer with auto-detection; **`tracelog upgrade`** pulls the latest release with checksum verification
- **30-Day Retention** — Configurable automatic data cleanup
- **Multi-Server** — Hub/Agent architecture with WebSocket transport
- **Prometheus** — `GET /metrics` exposition format for Grafana / alerting (optional Bearer token)

## Quick Start

### Install (one command — with or without Go)

The same installer works on every supported server (Linux/macOS, `amd64` / `arm64`):

1. **If you have a [GitHub Release](https://github.com/tudorAbrudan/tracelog/releases)** with `tracelog_linux_amd64.tar.gz` (or `arm64`), it downloads the binary — **no Go needed**.
2. **If there is no release**, it uses **`go install`** when `go` is already on `PATH` (`GOTOOLCHAIN=auto` can fetch a newer toolchain).
3. **If Go is not installed**, it downloads an official **Go tarball from [go.dev](https://go.dev/dl/)** (~150MB), then runs `go install` once and deletes the temporary tree — **no manual Go install**.

```bash
curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/install.sh | bash
```

On Linux this also creates the `tracelog` user, `/var/lib/tracelog`, `/etc/tracelog/config.yaml`, and a **systemd** service on port **8090**. Open `http://your-server:8090` and log in (installer prints the initial `admin` password when it can).

Use `sudo bash` if the script asks for privilege escalation (it uses `sudo` internally for system paths).

### Uninstall (restore system to pre-install state)

Removes the **systemd** unit, **`/usr/local/bin/tracelog`**, and **`/etc/tracelog`**. You are prompted whether to delete **`/var/lib/tracelog`** (database and generated data). To remove **everything** without prompts:

```bash
curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo bash -s -- --yes
```

Equivalent non-interactive flag via environment:

```bash
TRACELOG_UNINSTALL_YES=1 curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/uninstall.sh | sudo -E bash
```

If you keep data, the `tracelog` system user may remain (owns `/var/lib/tracelog`); the script tells you how to remove it later.

**Alternative:** `sudo tracelog uninstall` (interactive; also removes `/etc/tracelog`).

### Manual install (only Go)

```bash
go install github.com/tudorAbrudan/tracelog/cmd/tracelog@latest
tracelog user create admin
tracelog serve
```

### Build from source

```bash
git clone https://github.com/tudorAbrudan/tracelog.git
cd tracelog
make build
```

## Architecture

TraceLog runs as a single binary with three modes:

| Mode | Description | Use Case |
|------|-------------|----------|
| `serve` | Hub + Agent combined | Single server (most common) |
| `hub` | Hub only | Central server in multi-server setup |
| `agent` | Agent only | Remote server reporting to a hub |

```
┌─────────────────────────────────────────┐
│              TraceLog Binary             │
├──────────────────┬──────────────────────┤
│      Hub         │       Agent          │
│  ┌────────────┐  │  ┌────────────────┐  │
│  │ HTTP API   │  │  │ System Metrics │  │
│  │ WebSocket  │◄─┼──│ Docker Stats   │  │
│  │ Dashboard  │  │  │ Log Tailing    │  │
│  │ SQLite DB  │  │  │ Auto-Detect    │  │
│  └────────────┘  │  └────────────────┘  │
└──────────────────┴──────────────────────┘
```

## Commands

```
tracelog serve          # Start hub + local agent
tracelog hub            # Start hub only
tracelog agent          # Start agent (connects to remote hub)

tracelog user create    # Create admin user
tracelog user list      # List all users
tracelog user reset-password <username>

tracelog status         # Show DB size, servers, retention
tracelog backup         # Backup database
tracelog restore <file> # Restore from backup

tracelog version        # Print version
tracelog help           # Show all commands
```

## Configuration

### Command-Line Flags

```bash
tracelog serve --port 8090 --bind 0.0.0.0 --data /var/lib/tracelog
tracelog agent --hub http://hub:8090 --key tl_your_key_here
```

### Reverse Proxy (nginx)

```nginx
server {
    listen 443 ssl;
    server_name monitor.example.com;

    ssl_certificate /etc/letsencrypt/live/monitor.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitor.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/ws/ {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Dashboard

The web dashboard provides:

- **Overview** — Server cards with status indicators
- **Server Detail** — CPU, memory, disk, network, load charts with time range selector (1h, 6h, 24h, 7d, 30d)
- **Logs** — Real-time log viewer with search, level filter, source filter
- **Uptime** — HTTP endpoint monitoring with response time graphs
- **Settings** — Data retention, collection interval, log sources, notifications, alerts, account management

## API

All endpoints require authentication (session cookie) except `/api/health` and `/api/auth/login`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/health` | Health check |
| POST | `/api/auth/login` | Login |
| POST | `/api/auth/logout` | Logout |
| GET | `/api/auth/me` | Current user info |
| GET | `/api/servers` | List servers |
| GET | `/api/servers/:id` | Get server |
| GET | `/api/servers/:id/metrics?range=1h` | Get metrics |
| GET | `/api/servers/:id/docker?range=1h` | Get Docker metrics |
| POST | `/api/servers` | Create server |
| GET | `/api/logs?server_id=X` | Query logs |
| GET | `/api/settings` | Get settings |
| PUT | `/api/settings` | Update settings |
| GET | `/api/detect` | Run auto-detection |

## Tech Stack

- **Backend**: Go, SQLite (WAL mode), `net/http`
- **Frontend**: Svelte 5, uPlot, Vite
- **Metrics**: gopsutil
- **Transport**: WebSocket (coder/websocket)
- **Notifications**: go-mail (SMTP), webhooks

## Development

```bash
# Start the Go backend
make dev

# In another terminal, start the Svelte dev server
make web-dev

# Build everything
make build

# Run linter
make lint

# Run tests
make test
```

## License

MIT — see [LICENSE](LICENSE).
