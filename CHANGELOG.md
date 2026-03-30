# Changelog

All notable changes to TraceLog are documented here. The format is loosely based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

## [v0.2.19] - 2026-03-30

### Added

- **HTTP Analytics — auto-alert for new IP threats:** when a **new IP** appears in "Recommended to block" panel with **BLOCK decision**, automatically send email notification via configured channel (one-time alert per IP). Settings → General has optional **"IP threat auto-alert"** channel selector.

### Fixed

- **ipinfo.io API key not persisting:** `ipinfo_io_api_key` setting now saved to database (was missing from settings whitelist).

### Changed

- **IP threat alerts simplified:** removed per-IP manual "📧 Alert" button and channel selector from HTTP Analytics (auto-alerts now handle all new threats). **Add to list** button remains for manual blacklist management.

## [v0.2.18] - 2026-03-30

### Added

- **HTTP Analytics — IP threat assessment (ipinfo.io integration):** `ipinfo_cache` table stores IP geo + abuse data; `GET /api/threat/ipinfo` returns cached data with threat decision (block/monitor/allow) based on abuse confidence score + traffic pattern analysis. **Recommended to block** panel now shows Country/Region, abuse confidence %, and decision badge per IP.
- **HTTP Analytics — IP threat email alerts:** configurable **notification channel selector** in the **Recommended to block** panel; **📧 Alert** button per IP sends email via selected channel when decision is "block". `POST /api/threat/alert-ip` endpoint logs alert to `alert_history`.
- **Alert history — channel tracking:** `alert_history.channel_id` column (migration 010) tracks which notification channel sent each alert; Settings → Alerts shows **Emails sent** table filtered to email-only notifications with recipient email and channel name.

## [v0.2.17] - 2026-03-28

### Fixed

- **HTTP Analytics — threat scoring not firing:** `top_ips` does not carry `error_count`; `scoredIP` now cross-references `bad_requests_by_ip` for accurate per-IP error count. Badges (THREAT/SUSPICIOUS/SCANNER/BOT/SUBNET) now appear immediately on first page load without requiring a blacklist save.
- **HTTP Analytics — Slow requests duration invisible:** renamed column header from "Ms" to "Duration", appended `ms` unit to value, added orange bold styling (`.slow-ms`).
- **HTTP Analytics — Bytes sent column:** fixed field reference from `row.bytes` to `row.bytes_sent`.

### Added

- **HTTP Analytics — "Recommended to block" panel:** appears at the top of the Clients tab when any IP scores ≥ 3. Shows IP, request count, error count, error %, threat level, and reason badges. One-click **Add to list** per IP; **Add all to IP list** button when multiple IPs qualify. Adds to the textarea (still requires **Save list** + nginx export to actually block).

## [v0.2.16] - 2026-03-28

### Added

- **HTTP Analytics — 4-tab layout:** **Overview** (summary stats + traffic timeline chart), **Paths** (top paths by volume + by avg duration), **Clients** (IP threat scoring + blacklist), **Requests** (bad, slow, recent).
- **HTTP Analytics — traffic timeline chart:** uPlot area chart of requests over time; bucket size adapts to the selected range (5 min for 1H → 1 day for 30D). Requires new `GET /api/servers/{id}/access-timeline` endpoint.
- **HTTP Analytics — automatic threat/scanner detection:** per-IP threat score computed from error rate, scanner path hits (`.env`, `wp-admin`, `/etc/passwd`, etc.), known bot User-Agents (`nuclei`, `zgrab`, `masscan`, …), and `/24` subnet clustering. Badges: **THREAT** (score ≥ 6), **SUSPICIOUS** (3–5), **SCANNER**, **BOT**, **SUBNET**.
- **HTTP Analytics — top paths by avg duration:** `top_paths_by_duration` added to `access-stats` response (min 3 requests, sorted slowest-first).
- **HTTP Analytics — bytes sent per IP:** `bytes_sent` column added to top-IPs stats.
- **Logs — Level and Source dropdowns:** column filters for Level and Source are now dropdowns populated from distinct values present in the loaded logs (levels sorted by severity, sources alphabetically). Selection resets automatically on server change.

### Changed

- **HTTP Analytics — IP blacklist** moved to bottom of the Clients tab.

## [v0.2.15] - 2026-03-27

### Fixed

- **Runtime crash on null API responses:** Go nil slices marshal to JSON `null`; pages that called `.length` on the result would crash mid-render when a server had no metrics, processes, or access logs in the selected range. Added `?? []` null guards in `ServerDetail`, `Processes`, `HttpAnalytics`, and `Logs`.

## [v0.2.14] - 2026-03-27

### Added

- **Per-server alert muting:** toggle alert notifications on/off for individual servers from the Overview server cards (`alerts_muted` column, DB migration 008). API: `PATCH /api/servers/{id}/alerts-muted`. Muted state is loaded at hub startup and respected by `notifyAlert`.
- **Alert rule editing:** `PUT /api/alerts/{id}` endpoint; `api.updateAlertRule` on the frontend.
- **DB performance indexes** (migration 007): `idx_alert_history_rule`, `idx_alert_history_server`, `idx_access_logs_ip`.

### Changed

- **Error handling across the UI:** API errors are now surfaced inline (no more `alert()` / silent `console.error`) in Overview, Logs, HTTP Analytics, Processes, DockerLogsPanel, and ServerDetail — using a shared `LoadingState` component.
- **Purge feedback in Logs:** result message shown inline instead of `alert()`.
- **`api.ts` request helper:** only parses response JSON as an error body when `Content-Type` is `application/json`; successful responses no longer attempt a double-parse.
- **TypeScript:** tightened `any` types to `Record<string, unknown>` in `createLogSource`, `createUptimeCheck`, `createAlertRule`, `createNotificationChannel`.
- **`fmtBytes` utility:** extracted to `web/src/lib/utils/format.ts`; removed local copies from DockerLogsPanel, Processes, and ServerDetail.
- **gosec G201 suppression:** `sqlite.go` retention cleanup now carries `//nolint:gosec` comment explaining the hardcoded table/column source.

## [v0.2.13] - 2026-03-27

### Added

- **HTTP Analytics — slow requests:** table with configurable **minimum duration (ms)**; **`GET /api/servers/{id}/access-slow-requests`** (`range`, `min_ms`, `limit`). Uses the same User-Agent exclusions and hub UI path filter as aggregate stats; sorted slowest first.
- **Settings → Servers:** edit **name**, **registered host**, and **note** (free text included in **alert emails**); **`PUT /api/servers/{id}`**; migration **`servers.notes`**.
- **Alert notification bodies:** **Alert type** + short **What this is**; optional **Server note**; **Configured path on agent** / **Docker log source target** when the log source name matches Log Sources; **UI tip**; **Docker container (metric)** label for docker stats rules.

### Documentation

- **README**, **configuration**, **logs-http-analytics** updated for the above.

## [v0.2.12] - 2026-03-27

### Fixed

- **CI / lint (gosec):** document **`//nolint:gosec` (G202)** on **`QueryAccessBadRequests`** and **`GetRecentAccessLogs`** in `accesslogs.go` — dynamic `WHERE` uses only `accessLogExcludeHubUIPrefixSQL` fragments with bound args. **`store_test.go`:** test temp log file written with **`0o600`** (G306).

## [v0.2.11] - 2026-03-27

### Added

- **Documentation & presentation:** redacted dashboard screenshots under **`docs/public/screenshots/`** (Overview, Servers, Logs, Docker dark/light, Processes, Alerts, Log Sources, Uptime). **`scripts/redact_doc_screenshots.py`** regenerates them from raw PNG exports (sensitive regions obscured).
- **VitePress home** hero image; guide pages and **README** embed the same assets with captions noting sanitization.

## [v0.2.10] - 2026-03-27

### Added

- **Alert emails and webhooks** append context: **server name**, **registered host**, **server ID**, **log source** (path or tag) for log-based rules, and **Docker container** for Docker metric alerts. **`TRACELOG_PUBLIC_DASHBOARD_URL`** (hub environment) adds a **Dashboard URL** line for a direct link to the UI (include path prefix when using `--url-prefix` / `TRACELOG_URL_PREFIX`).

### Changed

- **Alerts:** `OriginServerID`, `LogSource`, and `DockerContainer` on fired alerts so global log rules attribute the correct server; **alert history** uses the origin server id when present.

### Documentation

- **Configuration:** subsection **Alert emails and webhooks** describes the above and `TRACELOG_PUBLIC_DASHBOARD_URL`.

## [v0.2.9] - 2026-03-27

### Added

- **Remote agent log tailing:** `GET /api/agent/log-sources` (authenticate with **`X-API-Key`**). Remote `tracelog agent` polls about every **2 minutes** and tails **file** log sources whose **Settings → Log Sources → agent** matches that server. Paths are **not** validated on the hub for remote rows.
- **Alert notification history:** each sent email/webhook is stored in **`alert_history`**; **Settings → Alerts → Recent alert notifications** lists the latest rows. API: **`GET /api/alert-history?limit=`** (session auth).

### Changed

- **Log source validation:** local hub sources still require the file on the hub machine; sources bound to another **server_id** skip hub-side path checks.
- **CI / Makefile:** run **`npm run check`** (Svelte + TypeScript); **Vitest** uses **`--passWithNoTests`** so the suite passes when no frontend tests exist.
- **UI:** **`contextServerId`** persisted in **sessionStorage**; **`onMount(async …)`** typing fixes in **App** and **Processes**.

### Documentation

- Multi-server, configuration, alerts, and product-scope updated for remote log sources and alert history.

## [v0.2.8] - 2026-03-27

### Fixed

- **CI / lint:** document and suppress gosec **G202** (SQL string concatenation) for HTTP analytics aggregate queries in `internal/hub/store/accesslogs.go`, where the dynamic `WHERE` clause uses only fixed SQL fragments plus `INSTR(…, ?)` patterns with bound arguments.

## [v0.2.7] - 2026-03-28

### Fixed

- **Release / GoReleaser:** keep `web/package-lock.json` aligned with `package.json` version and build the frontend with **`npm ci`** in the release workflow (instead of `npm install`) so the runner does not rewrite the lockfile and GoReleaser no longer fails with `git is in a dirty state`.

## [v0.2.6] - 2026-03-27

### Added

- **Log sources — ingest by severity:** per source, choose which levels are **stored** (critical, error, warn, info, debug, **deprecated**). None checked = all levels (unchanged behaviour). **Settings → Log Sources** → **Save levels**; **restart TraceLog** so the agent reloads. Hub drops non-matching lines as well. API: `PUT /api/log-sources/{id}` with `ingest_levels`; DB column `ingest_levels`.
- **Deprecated** log level: detected in plain/apache-style lines; query support and **Logs** filter “Deprecated or higher”; Docker logs panel heuristic.
- **HTTP analytics — ignore User-Agent:** **Settings → General**, one substring per line; those rows are excluded from **Top URL paths**, **Top method + path**, **Top IPs**, and summary aggregates (not from raw “Recent requests”). Default includes `TraceLog/1.0 Uptime Monitor`.
- **Uptime monitors:** 7-day **history strip** (green = up, red = down, up to 200 samples); note that checking a URL on the **same machine** only validates local reachability.
- **Processes (Linux):** hide processes that appear to run under **docker / containerd / kubepods** cgroups; short UI note. Non-Linux unchanged.

### Changed

- **HTTP Analytics:** wider single-column layout; paths use **word-wrap** instead of heavy truncation.
- **Overview:** server list poll only refreshes data; **sessionStorage** guard so single-server auto-nav does not repeat unexpectedly; `onMount` cleanup typing for Svelte 5.

### Fixed

- **Sign out:** also clears `tracelog-single-server-auto-nav-done` from session storage.

## [v0.2.5] - 2026-03-26

### Added

- **Log alert silences:** mute noisy ingested-log notifications by **substring** (case-insensitive), with optional **server** and **log rule** scope. API: `GET/POST /api/log-alert-silences`, `DELETE /api/log-alert-silences/{id}`; **Settings → Alerts → Log alert silences**.

### Fixed

- **Dashboard navigation:** stop unexpected jumps back to the single-server view on the periodic server list refresh; auto-open only on first successful load (and after adding the only server). **Sidebar** and **server cards** set suppress for explicit navigation; **sessionStorage** restores the last page after reload (cleared on sign out).
- **Server detail:** remove stray `dockerRows` assignments that caused `ReferenceError: dockerRows is not defined` in the browser console.

## [v0.2.4] - 2026-03-26

### Added

- **`tracelog serve`:** load **Log Sources** from the hub database at startup so ingested file logs and HTTP analytics work after configuring sources (restart required when sources change).
- **Logs:** severity filter for stored lines — **Critical only**, **Error or higher**, **Warning or higher**, etc.; API `severity_min` query param; **`critical`** log level in the agent parser (plain/apache/nginx fallback).
- **Docker logs panel:** same severity menu filters raw output client-side (keyword heuristics).
- **Processes:** **Refresh** button for an immediate snapshot.
- **Alerts:** rules on **ingested log level** (`log_critical`, `log_error`, `log_warn`) with configurable cooldown; fires when a matching line is stored (not for raw Docker “Load logs” UI text). **Settings → Alerts** documents metrics vs log rules.

### Fixed

- **Alert engine:** metric evaluation uses a proper write lock so firing no longer races with `lastFired` updates; log rules are skipped during system-metric evaluation.

### Changed

- **Mobile:** hamburger menu, off-canvas sidebar, backdrop, safer horizontal scroll for tables; **Settings** layout tweaks on narrow screens.
- **Settings / HTTP Analytics:** copy clarifications (backup scope, log sources, empty analytics).

## [v0.2.3] - 2026-03-26

### Added

- **Hub database backup (UI + API):** `POST /api/database/export` (authenticated + CSRF); confirm with your TraceLog password, then download a SQLite snapshot (`VACUUM INTO`). **Settings → Account → Database backup.**

### Fixed

- **Login rate limiting:** after five failed attempts within one minute from an IP, enforce a fifteen-minute lockout; reset the limiter on successful login; align 429 messaging; small note on the login screen.

### Changed

- **Overview:** if there is only one server, open its detail automatically; use **Overview** in the sidebar or **Back** from the server page to see the grid again.
- **Docker container logs** moved from the server detail page to **Logs** (for the selected server when it is the local hub host).
- **HTTP Analytics:** clearer copy for IP rankings and table headings.

## [v0.2.2] - 2026-03-26

### Fixed

- **Release / GoReleaser:** stop committing `internal/hub/dist/` (Vite output changes every build and made the GitHub Actions tree dirty). CI runs `make web-build` before `go test` and lint.

### Added

- **HTTP Analytics**: unique IP count, top method+path, top IPs/paths (configurable depth), IP blacklist (exact IP + CIDR) for highlighting and estimated hit counts, bad requests (4xx/5xx) per IP and drill-down lines, external WHOIS/ipinfo links.
- **Logs (UI)**: purge stored ingested log lines from SQLite (by age or all) — does not delete files on disk; optional filter by log source name.
- **Log sources**: JSON tags on config models so “Scan for common log files” works; manual add validates path, file type, format vs sample lines; clearer validation errors.
- **Notifications (UI)**: Gmail SMTP example, App Password steps (Romanian), insert-template button.
- **API**: `POST /api/logs/purge`, `GET/PUT /api/access-ip-policy`, `GET /api/servers/{id}/access-bad-requests`, extended `GET /api/servers/{id}/access-stats` (`top_n`, extra fields).

### Changed

- Documentation and in-app Settings hints describe what each option does and how it differs from OS log files.

---

## How to ship a new version

1. Update this file: move items from **Unreleased** under a new section `## [vX.Y.Z] - YYYY-MM-DD`.
2. Commit on `main` (or your release branch).
3. Tag and push (triggers [GoReleaser](https://goreleaser.com/) via `.github/workflows/release.yml`):

   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

4. The workflow sets the binary version via `-ldflags "-X main.version={{.Version}}"`.
5. Optional: after release, mention highlights in GitHub **Releases** notes (can paste from CHANGELOG).

Pre-release builds show `dev` until built with GoReleaser or manual `-ldflags`.
