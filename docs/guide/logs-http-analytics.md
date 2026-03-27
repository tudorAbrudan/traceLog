# Logs, retention & HTTP analytics

This page explains how TraceLog handles **application/text logs**, **ingested copies vs files on disk**, **retention**, **HTTP access analytics**, and the **IP blacklist** (analytics only).

**Logs** page — filters, server scope, and purge (stored lines redacted in this screenshot):

![Logs viewer and purge controls](/screenshots/logs.png)

## Log Sources (Settings → Log Sources)

- **What it does:** Tells the agent which files to tail. Parsed lines are sent to the hub and stored in SQLite (`logs` table). The **Logs** page shows that stored copy, not a live `tail -f` of the file handle.
- **Scan for common log files:** Runs on the machine where the **hub process** runs (in `serve` mode, same as the agent). Only paths that exist are added (nginx, apache, syslog, etc.).
- **Manual add:** The path must exist on the host running TraceLog (validation). **Nginx** / **Apache** formats are checked against sample lines (access-log style). Use **Plain** for error logs, syslog-style lines, or generic app output.
- **Deleting rows:** On the **Logs** page, **Purge** removes data from TraceLog’s **database** only. It does **not** truncate `/var/log/...` on disk — use `logrotate` or server tools for that.

## Retention (Settings → General)

- **Retention days:** Metrics, Docker metrics, **ingested logs**, **access logs**, uptime results, alert history, and process metrics older than this window are deleted automatically (hourly job).
- **Collection interval:** How often the agent sends system (and related) metrics.

## HTTP Analytics (sidebar)

- **Data source:** Nginx (and similar) **access** lines parsed by the agent; stored in `access_logs`.
- **Top paths / top IPs / unique IPs:** Computed for the selected time range.
- **Top method + path:** Groups `METHOD` + `PATH` for quick insight into hot endpoints.
- **Bad requests:** Counts of **4xx/5xx** per IP; **Lines** shows recent matching rows. Status **≥ 400** defines “bad” here.
- **IP list (HTTP Analytics panel):** One IP or **CIDR** per line (`#` comments ignored for export). After **Save**, TraceLog **highlights** matching traffic and estimates volume (“Req. from IP list”). This is **analytics only** — it does **not** block clients. To **block**, use **nginx** `deny`, **firewall**, or **CDN**/WAF. The UI offers **Copy nginx deny snippet** / **Download .conf** from the same textarea (you paste into nginx and run `nginx -t` + reload; TraceLog does not change nginx). The feature lives on this page so you can tune the list while viewing **top IPs**; enforcement remains outside TraceLog.
- **WHOIS links:** Open external sites (e.g. ipwho.is, ipinfo.io); TraceLog does not run WHOIS on the server.

## API reference (auth required unless noted)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/health` | **Public.** `version`, `status`, `setup_done`. |
| GET | `/api/logs?server_id=&range=` | Query ingested log lines. |
| POST | `/api/logs/purge` | Purge ingested logs (CSRF). Body: `server_id`, `mode` `all` \| `older_than`, optional `range`, `source`. |
| GET | `/api/servers/{id}/access-stats?range=&top_n=` | HTTP analytics aggregates. |
| GET | `/api/servers/{id}/access-bad-requests?range=&ip=&limit=` | Recent 4xx/5xx lines. |
| GET | `/api/access-ip-policy` | JSON `{ "ips": ["1.2.3.4", "10.0.0.0/8"] }`. |
| PUT | `/api/access-ip-policy` | Save blacklist (CSRF). Body: `{ "ips": [...] }`. |

See the repository [README](https://github.com/tudorAbrudan/tracelog) for the full API table and release process.
