# Changelog

All notable changes to TraceLog are documented here. The format is loosely based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

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
