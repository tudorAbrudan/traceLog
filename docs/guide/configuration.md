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

### Hub Mode

```bash
tracelog hub [flags]
```

Same flags as serve mode.

### Agent Mode

```bash
tracelog agent [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--hub-url` | (required) | Hub WebSocket URL |
| `--api-key` | (required) | Server API key |
| `--interval` | `10` | Collection interval (seconds) |

## Dashboard Settings

Access Settings from the sidebar in the web dashboard:

- **General** — Data retention (1-30 days), collection interval
- **Log Sources** — Configure which log files to monitor
- **Notifications** — SMTP email and webhook channels
- **Servers** — Manage remote agents and API keys
- **Alerts** — CPU, memory, disk threshold rules
- **Account** — Current user info

## Data Storage

All data is stored in a SQLite database at `{data-dir}/tracelog.db` using WAL mode for concurrent reads.

### Retention

Data older than the configured retention period (default 30 days) is automatically cleaned up every hour. This includes:
- System metrics
- Docker metrics
- Logs
- Access logs
- Uptime results
- Alert history
