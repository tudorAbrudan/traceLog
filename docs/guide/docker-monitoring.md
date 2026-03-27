# Docker monitoring

The TraceLog agent can scrape `docker stats` and send samples to the hub. The UI shows containers, memory (including **% of the container cgroup limit**), CPU (as a **share of the host**, matching `docker stats`), and on-demand log lines from `docker logs`.

## Where to find it in the UI

- **Servers →** open a server **→ Docker & container logs** lists containers for that server (any host where the agent reports Docker metrics).
- From **Logs**, use **Open … → Docker** to jump to the same section on that server’s detail page.

Doc screenshots redact container names and raw log lines.

**Dark theme**

![Docker table and log viewer (dark)](/screenshots/docker.png)

**Light theme**

![Docker table and log viewer (light)](/screenshots/docker-light.png)

On-demand **Load logs** reads stdout/stderr via the agent and does **not** store lines in the database, so it does **not** drive log-based alerts.

## Alerts on Docker metrics

In **Settings → Alerts**, choose a Docker metric rule and bind it to the **server** whose agent runs Docker:

| Metric | Meaning |
|--------|---------|
| `docker_mem_pct` | `mem_used / mem_limit × 100` for each container (OOM risk vs **container** limit, not host RAM). |
| `docker_cpu_percent` | CPU % as reported by Docker (relative to **host**; can exceed 100% on multi-core). |

Optional **container name contains** filters which containers are evaluated (case-insensitive substring). Empty means all containers on that server.

Duration and cooldown behave like host metric rules: the condition must hold for the duration, then notifications respect the cooldown **per container** (same rule can alert separately for different containers).

## Getting container logs into TraceLog (for log alerts)

Log alerts in TraceLog apply only to lines **ingested** as normal log entries (`IngestLog`), not to the ephemeral **Load logs** UI.

### Recommended: log driver + file Log Source

1. Run containers with the **`json-file`** (default) or **`syslog`** driver so logs land on disk or in syslog.
2. In TraceLog **Settings → Log Sources**, add a **file** (or path your setup writes) that tails those log files.
3. Set **ingest levels** so error/critical lines are stored; **Settings → Alerts** log rules (`log_error`, etc.) can then notify.

This avoids running a continuous `docker logs` poller on every host and keeps volume predictable.

### Alternatives

- Ship container logs with your existing log stack (Fluent Bit, Vector, etc.) into files or APIs TraceLog already tails.
- A dedicated **docker logs tail** inside the agent (with deduplication and backoff) is possible but not enabled by default; prefer file-based ingestion when you can.
