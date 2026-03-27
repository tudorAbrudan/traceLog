---
description: Code review the current changes in TraceLog against project standards and security requirements
---

Review the current changes in this TraceLog session using the code-reviewer agent.

1. Run `git diff HEAD` to see all changes
2. Run `git diff --staged` to see staged changes
3. Apply the full review checklist from the code-reviewer agent
4. Return: Critical issues / Warnings / Suggestions / Verdict

Focus especially on:
- SQL injection vectors (user input in queries)
- gosec suppressions without proper justification
- Missing CHANGELOG.md updates
- TypeScript errors in Svelte components
