---
description: Scaffold a new SQLite migration for TraceLog using the addColumnIfMissing pattern
---

Add a new database migration for TraceLog. Migration details: $ARGUMENTS

Process:
1. Read `internal/hub/store/sqlite.go` to see the existing migration constants (migration001–migration006) and the `migrations` slice
2. Add a new `const migrationXXX = \`...\`` constant (next number in sequence) with your `ALTER TABLE` or `CREATE TABLE IF NOT EXISTS` SQL
3. Register it by appending to the `migrations` slice in `migrate()`
4. Update any relevant model structs in `internal/models/`
5. Update store query functions that SELECT from the affected table (add the new column)

Rules:
- Each migration runs exactly once, tracked by `schema_version` table — `ALTER TABLE` is safe here
- New tables: use `CREATE TABLE IF NOT EXISTS`
- Default values: choose safe defaults that don't break existing rows
- Document the migration in CHANGELOG.md under [Unreleased]
