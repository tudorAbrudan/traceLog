---
description: Add or update an entry in CHANGELOG.md under [Unreleased]
---

Update CHANGELOG.md with new changes. $ARGUMENTS

Process:
1. Read the current CHANGELOG.md
2. Find the `## [Unreleased]` section
3. If $ARGUMENTS describes changes: format them into the appropriate sub-sections (Added / Changed / Fixed / Documentation)
4. If no arguments: look at recent git commits (`git log --oneline -20`) and suggest what to add

Formatting rules:
- **Added** — new features or capabilities
- **Changed** — behavior changes to existing features
- **Fixed** — bug fixes
- **Documentation** — docs-only changes
- Each bullet: starts with bold feature area, e.g. `- **Store:** added docker_metrics index`
- Keep language consistent with existing CHANGELOG entries
