---
description: Run TraceLog tests (make test) and report failures clearly
---

Run the full test suite for TraceLog and report results.

Steps:
1. Run `make test` (includes web-build + go test -race + vitest)
2. If it fails, identify and show:
   - Which test failed and why
   - The exact error message
   - Which file/function is affected
3. If all pass, confirm clean.

Note: `make test` requires a web/dist to exist (it runs web-build first). If you only want Go tests quickly: `go test -race -count=1 ./...`
