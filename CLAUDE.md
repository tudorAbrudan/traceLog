# TraceLog — Claude Context

## What this is
Lightweight server monitoring in a single binary (Go + Svelte). Hub/Agent architecture:
- **Hub** (`tracelog serve`) — dashboard, SQLite DB, WebSocket server, alert engine
- **Agent** (`tracelog agent --hub ... --key ...`) — pushes metrics/Docker/logs to a hub

## Repo layout
```
cmd/tracelog/        — main entrypoint (run.go wires up subcommands)
internal/
  hub/               — HTTP API, store (SQLite), alert engine, uptime, WebSocket hub
    store/           — all DB access (accesslogs, docker, settings, …)
  agent/
    collector/       — system metrics (gopsutil), Docker stats
    detect/          — log source detection
  installer/         — install/upgrade helpers
  models/            — shared structs (metrics, alerts, log lines, …)
  upgrade/           — self-upgrade logic
web/                 — Svelte 5 frontend (src/lib/*, App.svelte)
docs/                — VitePress docs (guide/*.md)
scripts/             — install.sh, redact screenshots
```

## Build & run
```bash
make build          # web-build + go build → ./tracelog binary
make dev            # go run ./cmd/tracelog serve (no embedded frontend)
make web-dev        # cd web && npm run dev  (Vite dev server)
make test           # web-build + go test -race ./... + vitest
make lint           # golangci-lint + svelte-check + eslint
make fmt            # gofmt + goimports + prettier
```

> `web-build` is required before `go test` or `lint` because `internal/hub` uses `//go:embed dist`.

## Stack
- **Go 1.26+** (go.mod: `go 1.26.1`), module `github.com/tudorAbrudan/tracelog`
- **SQLite** via `modernc.org/sqlite` (no CGO)
- **WebSocket** via `github.com/coder/websocket`
- **Email** via `github.com/wneessen/go-mail`
- **Frontend**: Svelte 5, Vite/Rolldown, uPlot for charts, TypeScript

## Conventions
- No external frameworks in Go — stdlib + the listed dependencies only
- Store layer: raw SQL, no ORM. Migrations are inline (addColumnIfMissing pattern)
- API routes: `internal/hub/hub.go` registers all handlers
- Alert rules are string constants; new metric types need a rule key + store query + frontend label
- Frontend state: sessionStorage for contextServerId; no global state manager
- gosec G202 (dynamic SQL fragments) — suppress with `//nolint:gosec` + comment explaining why safe

## Release process
- Version in git tags (`v0.x.y`); GoReleaser builds cross-platform binaries
- CHANGELOG.md updated before tagging; `make build` must pass cleanly
- `npm ci` (not `npm install`) in release — keeps package-lock.json stable

## Key env vars (runtime)
| Var | Purpose |
|-----|---------|
| `TRACELOG_DB` | SQLite path (default `./tracelog.db`) |
| `TRACELOG_PORT` | Listen port (default 8090) |
| `TRACELOG_URL_PREFIX` | Subpath prefix (e.g. `/tracelog`) |
| `TRACELOG_PUBLIC_DASHBOARD_URL` | Appended to alert emails as a link |
| `TRACELOG_DOMAIN` + `TRACELOG_LETSENCRYPT_EMAIL` | Auto-TLS via certbot |
