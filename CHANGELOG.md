# Changelog

All notable changes to TraceLog are documented here. The format is loosely based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

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
