---
description: Run linters for TraceLog (Go + Svelte/TypeScript) and report issues
---

Run linters and type checks for TraceLog. $ARGUMENTS

If no arguments: run everything.
If "go": run only Go linters.
If "web": run only frontend checks.

**Go linting:**
```bash
golangci-lint run ./...
```

**Frontend checks:**
```bash
cd web && npm run check   # svelte-check + tsc
cd web && npx eslint .
```

Note: `golangci-lint run` requires web/dist to exist (hub uses go:embed). If dist is missing, run `make web-build` first or use `go vet ./...` as a lighter alternative.

Report:
- All errors (must fix)
- Warnings if any
- "Clean" if nothing found
