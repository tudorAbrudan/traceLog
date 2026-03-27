# TraceLog

**Lightweight server monitoring in a single binary.** Track CPU, RAM, disk, network, Docker containers, logs, and uptime — with a beautiful dark-mode dashboard and zero dependencies.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **System Metrics** — CPU, memory, disk, network, load average, uptime
- **Docker Monitoring** — Container CPU, memory, network; view **container logs** from the server detail page (local server)
- **Log Aggregation** — Tail log files, validate formats, scan common paths; **Logs** viewer + optional **purge** of stored lines (DB only, not disk files)
- **HTTP Analytics** — Top paths, IPs, method+path, unique IPs, **4xx/5xx** per IP with drill-down, optional **IP blacklist** (highlight + volume estimate; blocking is done in nginx/firewall)
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

On Linux the installer sets up **production-style** defaults: **nginx** proxies **HTTP (80)** to TraceLog on **127.0.0.1:8090** (port 8090 is not exposed publicly). Open **`http://your-server-ip/`** and log in (installer prints the initial `admin` password when it can). For **HTTPS**, point DNS at the host and run with `TRACELOG_DOMAIN=your.domain` and `TRACELOG_LETSENCRYPT_EMAIL=you@example.com` so **certbot** can obtain a certificate. To skip nginx and bind on all interfaces like a dev setup: `TRACELOG_NO_PROXY=1`.

Use `sudo bash` if the script asks for privilege escalation (it uses `sudo` internally for system paths).

#### HTTPS subpath on an existing site (e.g. cadourile.ro)

One-liner (installer writes snippets under `/etc/nginx/conf.d/` and `/etc/nginx/snippets/`, and **tries** to patch your vhost when **`TRACELOG_NGINX_SITE`** matches the filename in `sites-enabled`):

```bash
sudo TRACELOG_URL_PREFIX=/tracelog TRACELOG_NGINX_SITE=cadourile.ro bash -s < <(curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/install.sh)
```

Optional: **`TRACELOG_PUBLIC_DOMAIN=cadourile.ro`** if the banner should show a hostname different from the vhost filename.

**If the installer cannot edit nginx** (unusual vhost layout, or you did not set `TRACELOG_NGINX_SITE`), it prints **step-by-step instructions** in the terminal. Manually:

1. Open your **HTTPS** vhost — on Ubuntu/Debian this is often:
   - **`/etc/nginx/sites-enabled/cadourile.ro`**  
   (discover: `sudo grep -r server_name /etc/nginx/sites-enabled/`)
2. Inside the `server { }` block that has **`listen 443`** (ssl) and **`server_name`**, add **after** `server_name` (same indentation as the other directives):

```nginx
    include /etc/nginx/snippets/tracelog-subpath-loc.conf;
```

Example fragment:

```nginx
server {
    server_name cadourile.ro www.cadourile.ro;
    include /etc/nginx/snippets/tracelog-subpath-loc.conf;
    ...
}
```

3. Test and reload: **`sudo nginx -t && sudo systemctl reload nginx`**

The installer always creates **`/etc/nginx/conf.d/tracelog-subpath-map.conf`** (WebSocket `map`) and **`/etc/nginx/snippets/tracelog-subpath-loc.conf`** (`location /tracelog/` → `http://127.0.0.1:8090/`). You only need to add the **`include`** in your site if auto-inject did not run.

**Agents:** hub URL **`https://cadourile.ro/tracelog`** (no trailing slash is fine).

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

Screenshots below use **sanitized** exports (hosts, URLs, log lines, container names, and similar details are redacted) for safe use in docs and README. To refresh them from new raw captures, run `python3 scripts/redact_doc_screenshots.py /path/to/raw/pngs` (see script for expected filename fragments).

| Overview — metrics & charts | Servers — multi-host cards |
|----------------------------|----------------------------|
| ![Overview dashboard](docs/public/screenshots/overview.png) | ![Servers list](docs/public/screenshots/servers.png) |

| Logs — search & purge (stored copy) | Docker — container stats & on-demand logs |
|-------------------------------------|---------------------------------------------|
| ![Logs page](docs/public/screenshots/logs.png) | ![Docker section](docs/public/screenshots/docker.png) |

| Uptime — HTTP checks from the hub | Settings — alert rule types |
|-----------------------------------|-----------------------------|
| ![Uptime monitors](docs/public/screenshots/uptime.png) | ![Alerts settings](docs/public/screenshots/alerts.png) |

The web dashboard provides:

- **Overview** — Server cards with status indicators
- **Server Detail** — CPU, memory, disk, network, load charts with time range selector (1h, 6h, 24h, 7d, 30d)
- **Logs** — Stored log lines (search, level filter); **purge** copy in DB by age or all
- **HTTP Analytics** — Traffic breakdown, bad requests, blacklist editor, WHOIS links
- **Uptime** — HTTP endpoint monitoring with response time graphs
- **Settings** — Retention, collection interval, log sources (with validation), Gmail/SMTP help, notifications, servers, alerts; **About** shows build version

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
| PUT | `/api/servers/:id` | Update server name, host, notes (CSRF) |
| GET | `/api/logs?server_id=X` | Query ingested logs |
| POST | `/api/logs/purge` | Purge ingested logs (CSRF) |
| GET | `/api/servers/:id/access-stats` | HTTP analytics (`range`, `top_n`) |
| GET | `/api/servers/:id/access-bad-requests` | Recent 4xx/5xx (`range`, `ip`, `limit`) |
| GET | `/api/servers/:id/access-slow-requests` | Slowest requests (`range`, `min_ms`, `limit`; same UA/path filters as stats) |
| GET | `/api/access-ip-policy` | IP blacklist JSON |
| PUT | `/api/access-ip-policy` | Save blacklist (CSRF) |
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

The dashboard is embedded from **`internal/hub/dist/`**, which is **generated** by Vite (`npm run build` in `web/`) and **not committed**. Run `make web-build` (or `make build`) before `go build` / `go test` so `//go:embed dist` is satisfied.

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

## Changelog & versioning

- **[CHANGELOG.md](CHANGELOG.md)** — User-facing changes per release; update **Unreleased** as you merge features, then add a `## [vX.Y.Z]` section before tagging.
- **Runtime version** — `tracelog version` and `GET /api/health` → `version`. Development builds show `dev`; release binaries get the tag via **ldflags** (see below).

## Publishing a release (GitHub)

Releases are built automatically with **[GoReleaser](https://goreleaser.com/)** when you push a **version tag**:

```bash
# 1) Update CHANGELOG.md (move Unreleased → vX.Y.Z), commit
git tag v0.2.0   # semver; must start with v
git push origin v0.2.0
```

Workflow [`.github/workflows/release.yml`](.github/workflows/release.yml) runs on `v*`, builds the embedded frontend, then uploads:

- `tracelog_linux_amd64.tar.gz`, `tracelog_linux_arm64.tar.gz`, same for `darwin`
- `checksums.txt`

After the first release, **`install.sh`** can download the tarball (no Go on the server). The Go module path stays `github.com/tudorAbrudan/tracelog`; the **GitHub repo** name used for releases is **`traceLog`**.

**Local release (optional):** install [goreleaser](https://goreleaser.com/install/), set `GITHUB_TOKEN`, then `goreleaser release --clean`.

### Login after install (admin password)

The installer prints **`Login: admin / <password>`** when it can run `tracelog user create` against the same database as the service. The systemd unit sets **`Environment=TRACELOG_DATA_DIR=/var/lib/tracelog`** so the hub and CLI use **`/var/lib/tracelog/tracelog.db`**.

If no password appeared:

1. Open the URL in the browser — if there are no users yet, use the **first-time setup** flow.
2. Or set a new password (as root):

   ```bash
   sudo -u tracelog env TRACELOG_DATA_DIR=/var/lib/tracelog /usr/local/bin/tracelog user reset-password admin
   ```

3. List users: `sudo -u tracelog env TRACELOG_DATA_DIR=/var/lib/tracelog /usr/local/bin/tracelog user list`

After upgrading from an older install, reload systemd if you added `TRACELOG_DATA_DIR`: `sudo systemctl daemon-reload && sudo systemctl restart tracelog`.

### One binary vs “one installer”

- **`tracelog`** is a **single binary** (UI embedded): that *is* the application.
- **`install.sh`** is a **one-liner** you pipe to `bash`: it places that binary under `/usr/local/bin`, adds systemd, config, and data dirs — no separate installer executable.

## License

MIT — see [LICENSE](LICENSE).
