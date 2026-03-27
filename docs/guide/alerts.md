# Alerts

TraceLog supports configurable alert rules that trigger notifications via email or webhooks.

**Settings → Alerts** — metric dropdown (system, Docker, ingested log levels). Example UI (audit rows redacted):

![Alert rules and metric types](/screenshots/alerts.png)

## Alert Rules

Create rules in Settings → Alerts. Each rule defines:

| Field | Description |
|-------|-------------|
| **Metric** | What to monitor: `cpu_percent`, `mem_percent`, `disk_percent`, `load_1`, Docker metrics (`docker_mem_pct`, `docker_cpu_percent`), or ingested log levels — see below. |
| **Operator** | Comparison: `>`, `>=`, `<`, `<=` |
| **Threshold** | Trigger value (e.g., 90 for 90%) |
| **Duration** | How long the condition must persist before firing |
| **Cooldown** | Minimum time between repeat alerts (default 5 min) |
| **Channel** | Which notification channel to use |

### Default Rules

TraceLog comes with these rules (disabled by default):

- CPU > 90% for 5 minutes
- Memory > 90% for 5 minutes
- Disk > 90% for 5 minutes

## Notification Channels

### Email (SMTP)

Configure in Settings → Notifications:

- SMTP Host & Port
- Username & Password
- From/To addresses
- TLS: for submission on port **587** (e.g. Gmail), use `"starttls": true` and `"use_tls": false`. For SMTPS on port **465**, use `"use_tls": true`.

### Webhooks

Send alerts to any HTTP endpoint:

- Custom URL
- HTTP method (POST/PUT)
- Custom headers

The webhook payload:

```json
{
  "subject": "Alert: CPU > 90%",
  "body": "cpu_percent is 95.2 (threshold: > 90.0)",
  "time": "2024-01-15T10:30:00Z"
}
```

Compatible with Slack, Discord, PagerDuty, and other webhook receivers.

## Docker container metrics

Rules using **`docker_mem_pct`** or **`docker_cpu_percent`** require a **server** (the agent host that scrapes Docker). Optionally set a **container name substring** to limit which containers are checked.

- **`docker_mem_pct`** — memory used as a percentage of the **container’s** cgroup limit (useful when host RAM looks fine but a capped container is near OOM).
- **`docker_cpu_percent`** — CPU usage as reported by Docker (**host-relative**; can exceed 100% on multi-core).

Cooldown is tracked **per container** for these rules. See [Docker monitoring](./docker-monitoring.md) for UI placement and how container logs relate to log alerts.

## Ingested log level rules

Rules such as **`log_error`** / **`log_critical`** fire when a **stored** log line matches the severity. Lines pulled only via the UI “Load logs” for Docker are not stored and do not trigger these rules. Ingest container logs via log files or another supported source if you need alerts on application errors inside containers.

The **`log_warn`** rule also matches **`deprecated`** lines (when that level is ingested). See [Product scope & FAQ](./product-scope.md) for goals, Docker auto-discovery, and multi-install notes.

## Log alert silences

In **Settings → Alerts**, **Log alert silences** suppress **notifications** for ingested lines whose **message** contains a given **substring** (case-insensitive). Optional **server** and **rule metric** narrow the scope. The line remains **stored** and searchable.

Example: silence pattern `missing file for id=4` with rule `log_critical` to skip notifications for that known message while keeping other critical alerts.

Details: [Product scope & FAQ](./product-scope.md#example-silence-critical-alerts-for-a-known-noisy-message).

## Recent alert notifications

**Settings → Alerts → Recent alert notifications** lists rows logged when the hub **sends** an email or webhook for a rule (newest first). Use it to confirm delivery; it is not a full audit log. Rows are stored in the **`alert_history`** table and are subject to normal DB [retention](./configuration.md#retention) cleanup.
