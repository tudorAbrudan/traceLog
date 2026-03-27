---
description: Scaffold a new API endpoint following TraceLog patterns (Go handler + store query + Svelte fetch call)
---

Scaffold a new API endpoint for TraceLog. The endpoint details: $ARGUMENTS

Follow this exact process:
1. Read `internal/hub/hub.go` to understand where to register the route
2. Find the most similar existing endpoint and read it fully as a pattern
3. Read the relevant store file in `internal/hub/store/` to understand query patterns
4. Implement in this order:
   a. Store query function (in the appropriate store file)
   b. HTTP handler (in hub.go or a sub-file if the area is large)
   c. Route registration in hub.go
   d. Svelte fetch call in the appropriate `web/src/lib/` component
5. After implementing, note what needs to be added to CHANGELOG.md under [Unreleased]

Conventions to follow:
- Store function: `Verb + Noun` naming
- Handler: document the route + params in a comment above
- Bound SQL parameters only (no concatenation of user input)
- Return JSON; use existing response patterns
