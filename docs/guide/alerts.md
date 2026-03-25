# Alerts

TraceLog supports configurable alert rules that trigger notifications via email or webhooks.

## Alert Rules

Create rules in Settings → Alerts. Each rule defines:

| Field | Description |
|-------|-------------|
| **Metric** | What to monitor: `cpu_percent`, `mem_percent`, `disk_percent`, `load_1`, etc. |
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
- TLS option

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
