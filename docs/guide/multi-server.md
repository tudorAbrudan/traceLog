# Multi-Server Setup

Monitor multiple servers from a single TraceLog dashboard.

## Architecture

The **hub** stores data and serves the UI. It does **not** poll other machines. Each monitored host runs an **agent** that opens an **outbound** WebSocket to the hub (no inbound ports required on the agent).

```
┌──────────────┐     WebSocket     ┌──────────────┐
│  Server A    │────────────────►  │              │
│  (agent)     │                   │   Hub        │
└──────────────┘                   │   Server     │
                                   │              │
┌──────────────┐     WebSocket     │  Dashboard   │
│  Server B    │────────────────►  │  + API       │
│  (agent)     │                   │  + SQLite    │
└──────────────┘                   └──────────────┘
```

**Overview — server cards** (sensitive fields redacted in doc images):

![Server cards on the hub Overview](/screenshots/servers.png)

**Logs — per-server dropdown** (log table redacted):

![Logs page with server selector](/screenshots/logs.png)

## What you configure where

### On each **monitored** server (remote host)

| Item | Action |
|------|--------|
| TraceLog binary | Installed; run **`tracelog agent`** (not `serve`) |
| Hub URL | `wss://` or `ws://` pointing at your hub. If the UI is behind a path prefix (e.g. `/tracelog`), the WebSocket URL typically uses the **same origin and path** as in the browser. |
| API key | Created on the hub (**Overview → Add Server** or **Settings → Servers**). One key per server row. |
| Network | Outbound access from this host to the hub (firewall). |
| Docker | Optional: the agent reports Docker metrics when Docker is available and collection is enabled. |

You do **not** configure the remote host inside the browser beyond creating its server row and key on the hub.

### On the **hub** (machine that runs the dashboard)

| Item | Action |
|------|--------|
| Mode | **`tracelog serve`** (hub + embedded local agent) or **`tracelog hub`** (hub only). |
| Users / login | Created on this instance. |
| **Add Server** | One row + API key per monitored host. |
| **Settings → Log Sources** | In **`serve`** mode, enabled file sources are read by the **local** agent on **this** machine only (paths must exist here). They are not pushed to remote agents. |

### Remote log files (hub-assigned sources)

1. In the dashboard, **Settings → Log Sources** add a **file** source and set **Agent** to the **remote** server (not “This hub”). The path must exist **on that agent’s host**; the hub does not check the file remotely.
2. The **`tracelog agent`** polls **`GET /api/agent/log-sources`** about every **2 minutes** (same base URL as WebSocket, header **`X-API-Key`**) and starts or updates tailers. No agent restart is required when you change sources in the UI.
3. **Local hub** sources still require a **TraceLog restart** after changes so the embedded agent reloads them.

See also [Configuration](./configuration.md#agent-mode-remote).

## Setup

### 1. Start the Hub

On your central server:

```bash
tracelog hub --port 8090 --bind 0.0.0.0
```

(Or use `tracelog serve` for a combined hub + local monitoring install.)

### 2. Create a Server Entry

In the UI: **Overview → + Add Server**, or API:

```bash
curl -X POST http://hub:8090/api/servers \
  -H "Content-Type: application/json" \
  -H "Cookie: session=..." \
  -d '{"name": "web-server-1", "host": "10.0.1.5"}'
```

Save the **`api_key`** from the response (or copy it from **Settings → Servers**).

### 3. Start the Agent

On each remote server:

```bash
tracelog agent --hub ws://hub:8090 --key tl_your_key_here
```

Flags are **`--hub`** and **`--key`** (not `--hub-url` / `--api-key`). Use **`wss://`** when the hub is behind HTTPS.

The agent connects **outbound** to the hub — no inbound agent ports.

## UI: per-server vs hub-wide

| Area | Scope |
|------|--------|
| **Overview** | All servers; click a card to open **Server detail** for that host. |
| **Server detail** (`server:…`) | Metrics and Docker for **that** server only. |
| **Logs**, **Processes**, **HTTP Analytics** | Each page has a **server** dropdown; data is for the selected server. Opening a server from **Overview** sets a **context** so these pages pre-select the same server when you switch tabs. |
| **Settings** | **Hub-wide**: notification channels, log sources (local `serve` host), alert rules, account. Some rules include a **server** field (e.g. host metrics, Docker metrics). |
| **Uptime** | **Hub-wide**: HTTP checks run **from the hub**, not tied to a monitored server card. |

**Notifications** are driven by rules in **Settings**; alert payloads include `server_id` where applicable. There is no separate “notification inbox” per server in the UI.

## Navigation tips

- **Metrics / Docker for one host:** **Overview** → click server card → **Server detail**.
- **Logs for that host:** **Logs** → pick the server in the dropdown (or open the server from Overview first to pre-fill the dropdown).
- You do **not** have to return to Overview only to switch servers on **Logs** / **Processes** / **HTTP Analytics** — use each page’s dropdown.

## Security

- Agents authenticate with API keys.
- Use **WSS** when the hub is served over HTTPS.
- API keys can be managed from the dashboard (**Settings → Servers**).
- Agents only **send** data; they do not listen for inbound monitoring traffic.
