# Product scope & FAQ

This page describes what TraceLog is **for**, what it **does today**, and how to think about **edge cases** (new Docker containers, multiple installs on one host, noisy log alerts).

## Is this the goal of the product?

**Broadly, yes.** TraceLog is meant to:

| Intent | Supported? | Notes |
|--------|------------|--------|
| Monitor servers (load, RAM, disk, CPU, etc.) | Yes | Per **server** (each registered agent / host). Charts and metric alerts. |
| Centralize application logs | Yes | **Log Sources** tail files (and similar paths); lines are stored and searchable. You choose **ingest levels** per source (e.g. store `error` and above, or include `deprecated`). |
| Include Docker-related signals | Partially | **Docker metrics** (per container, from `docker stats`) and on-demand **Load logs** in the UI. **Alerts** on container memory/CPU use **Settings → Alerts** (`docker_mem_pct`, `docker_cpu_percent`). |
| Notifications for thresholds and log severity | Yes | Metric alerts (host + Docker), uptime checks, and **ingested log** rules (`log_critical`, `log_error`, `log_warn`). |
| Suppress notifications for specific log text | Yes | **Log alert silences** (Settings → Alerts): **case-insensitive substring** on the message, with optional **server** and **rule** scope. The line is still **stored**; only the **notification** is skipped. |

So: **monitoring + log ingestion + alerting + silencing noisy known messages** matches the product direction. Gaps and nuances are called out below.

## New Docker containers after install — are they detected automatically?

**Yes.** The agent does **not** keep a fixed list of containers. On each collection cycle it runs `docker stats` (no stream) and reports **every running container** returned by Docker at that moment.

- A **new** container appears on the **next** successful scrape after it starts (subject to your **collection interval**).
- A **stopped** container disappears from the latest snapshot until it runs again.

No extra registration step is required in TraceLog for new containers.

## Two or more TraceLog installs on the same host — cumulative view?

**Not automatically.** Each `tracelog serve` instance is its **own hub**: its own database, UI, and alert rules. Two full stacks on one machine means **two separate dashboards** — they do **not** merge into one.

**Recommended patterns:**

- **One hub, one agent on that host** — single source of truth for that server.
- **One hub, many agents** — use **Multi-Server** so each machine is a **separate server** in one UI; that is your “centralized” view across hosts.
- **Two hubs on the same OS** — usually a mistake for operations (duplicate work, possible duplicate metrics if both run local agents). Prefer **one** hub unless you intentionally isolate environments (e.g. prod vs lab) and accept separate UIs.

Docker metrics are **per agent host**: one agent sees the Docker daemon on **that** host only.

## Hub vs monitored host (multi-server)

- **Hub** — Runs the dashboard, SQLite database, and (in `serve` mode) a **local** agent for the same machine. You create **server rows** and **API keys** here; **Settings** (channels, most alert rules, log sources for local files) apply to the **whole hub**, not to a single monitored card.
- **Monitored host** — Runs **`tracelog agent`** with `--hub` and `--key`. It pushes metrics/Docker/processes to the hub. **Log Sources** in Settings can be assigned to that server’s id; the agent **polls** the hub for file paths and tails them on the **agent host** (see [Multi-Server Setup](./multi-server.md)).

For step-by-step setup, CLI flags (`--hub` / `--key`), and which UI pages are **per-server** vs **global**, see [Multi-Server Setup](./multi-server.md).

## Example: silence “critical” alerts for a known noisy message

You have a rule for **ingested log · critical only**, but you do **not** want email when the message contains a known benign fragment, e.g. `missing file for id=4`.

1. Go to **Settings → Alerts → Log alert silences**.
2. Add a silence with **pattern** `missing file for id=4` (substring match).
3. Optionally set **server** to limit it to one host, and **rule** to e.g. `log_critical` so other rules are unaffected.

Matching is **substring**, not full regex. For multiple phrases, add **multiple** silence rows.

## Container logs and “critical error in Docker”

- **Ingested** lines (from a **Log Source** file/syslog path, etc.) go through **severity classification** and can trigger **log** alerts, including for `critical`.
- **Load logs** in the Docker section reads `docker logs` **on demand** for the UI only — those lines are **not** stored and **do not** trigger log alerts.

To alert on errors **inside** containers, ingest them via a **file** (e.g. json-file log path) or another supported path — see [Docker monitoring](./docker-monitoring.md).

## Ingest levels and `deprecated`

Per **Log Source**, you can restrict which severities are **stored** (and thus eligible for search and log-based alerts). If you **ingest** the **`deprecated`** level, stored `deprecated` lines match the **`log_warn`** alert rule (along with warn, error, and critical). They do **not** match **`log_error`** or **`log_critical`** alone. There is no separate `log_deprecated` rule — use **`log_warn`** if you want notifications on deprecated lines (and tune silences if some deprecations are noise).

## What is *not* covered (today)

- **Per-process list inside containers** in the UI — the **Processes** page intentionally hides cgroup workloads on Linux; use **Server → Docker** for **container-level** CPU/memory, or SSH / `docker top` for PIDs inside a container.
- **Automatic `docker logs` streaming** into the DB — not the default path; file/syslog ingestion is preferred for volume and alerting.
- **Regex or multi-field silences** — only **message substring** silences for log notifications.
- **Merging two hubs** into one view — operate one hub per logical control plane, or export/aggregate externally.

For alert types and Docker metrics in detail, see [Alerts](./alerts.md) and [Docker monitoring](./docker-monitoring.md).
