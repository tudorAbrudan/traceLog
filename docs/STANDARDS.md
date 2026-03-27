# TraceLog — Code Standards

## Go backend

### Structure
- One responsibility per file. Store queries in `internal/hub/store/`, handlers in `internal/hub/hub.go` (or a sub-handler file for large areas), models in `internal/models/`.
- New DB columns: add a new `migrationXXX` string constant in `internal/hub/store/sqlite.go` and register it in the `migrations` slice. Each migration runs exactly once, tracked by the `schema_version` table. `ALTER TABLE` is safe inside migration constants.
- No ORM. Raw SQL only. Use named parameter structs where the query has >3 args.

### SQL / gosec
- Dynamic `WHERE` clauses are allowed only when fragments are **fixed strings** (never user input). Document with `//nolint:gosec // G202: fragments are fixed SQL, args are bound`.
- All user-supplied values must be bound parameters (`?`), never concatenated.

### Error handling
- Return errors up the call stack. Do not swallow errors with `_ =`.
- HTTP handlers: `http.Error(w, "message", code)` for client errors; log + 500 for unexpected server errors.
- Never `panic` in request paths.

### Naming
- Exported types: `PascalCase`. Unexported helpers: `camelCase`.
- Store functions: verb + noun — `GetRecentAccessLogs`, `InsertLogLine`, `UpdateServerNote`.
- API route constants: `GET /api/servers/{id}/thing` — document in handler comment.

### Testing
- `go test -race -count=1 ./...` must pass before any commit.
- Test files live next to the code (`store_test.go`).
- Temp files in tests: `0o600` permissions (gosec G306).
- No mocking the DB layer — use a real SQLite database in tests (current tests use `t.TempDir()` for a temporary on-disk DB).

### Security
- Never expose internal error details to HTTP responses.
- API keys: compare with `crypto/subtle.ConstantTimeCompare`.
- Passwords: `bcrypt` only (already in use).

---

## Svelte / TypeScript frontend

### Structure
- Components in `web/src/lib/`. One component per file, named `PascalCase.svelte`.
- Shared types in `web/src/lib/types.ts` (or co-located if used only in one component).
- No global state manager. Use `sessionStorage` for cross-page values (pattern already in use: `contextServerId`).

### API calls
- Always handle errors explicitly — show a user-facing message, never silently swallow.
- Use `async/await`, not raw `.then()` chains.
- `onMount(async () => {...})` — type the function explicitly when TypeScript complains.

### Style
- Dark mode first. Tailwind utility classes. No inline `style=` unless truly dynamic.
- Charts use `uPlot` — do not introduce a second charting library.

### Checks
- `npm run check` (svelte-check + tsc) must pass.
- `npx eslint .` must pass with no errors.

---

## General

### Commits
- Format: `Area: short imperative description` (e.g. `store: add disk_history query`, `ui: slow requests table`).
- Update `CHANGELOG.md` under `## [Unreleased]` before tagging a release.
- One logical change per commit. Do not bundle unrelated fixes.

### Releases
- Tag format: `v0.x.y`. GoReleaser builds cross-platform.
- Always `npm ci` (not `npm install`) — keeps `package-lock.json` stable for GoReleaser's dirty-state check.
- `make lint && make test` must be green before tagging.

### What NOT to do
- Do not add runtime dependencies without explicit discussion (keep the binary lean).
- Do not introduce CGO — `modernc.org/sqlite` is used specifically to avoid it.
- Do not add a frontend router or SPA framework — the current single-page pattern is intentional.
- Do not add config-file parsing beyond what `cmd/tracelog/run.go` already does.
