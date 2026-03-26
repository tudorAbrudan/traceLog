# TraceLog web UI

Svelte 5 + Vite front-end; the production build is **embedded** in the `tracelog` binary (`go:embed`).

- **Develop:** from repo root, `make web-dev` or `cd web && npm run dev` (API proxied / pointed at a running hub as in your setup).
- **Ship:** `make build` builds the SPA into `internal/hub/dist/` then compiles Go.
- **Docs:** see repo [README](../README.md) and [docs/](../docs/) (VitePress site, including [Logs & HTTP analytics](https://tudorAbrudan.github.io/traceLog/guide/logs-http-analytics)).
