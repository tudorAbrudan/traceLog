# Multi-Server Setup

Monitor multiple servers from a single TraceLog dashboard.

## Architecture

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

## Setup

### 1. Start the Hub

On your central server:

```bash
tracelog hub --port 8090 --bind 0.0.0.0
```

### 2. Create a Server Entry

Log into the dashboard and click "Add Server", or use the API:

```bash
curl -X POST http://hub:8090/api/servers \
  -H "Content-Type: application/json" \
  -d '{"name": "web-server-1", "host": "10.0.1.5"}'
```

Note the API key from the response.

### 3. Start the Agent

On each remote server:

```bash
tracelog agent --hub-url ws://hub:8090 --api-key tl_your_key_here
```

The agent connects outbound to the hub via WebSocket — no inbound ports needed.

## Security

- Agents authenticate with API keys
- WebSocket connections are encrypted when using HTTPS/WSS
- API keys can be rotated from the dashboard
- Agents only send data, they don't expose any ports
