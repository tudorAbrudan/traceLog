---
description: Prepare a TraceLog release — update CHANGELOG, verify build/lint/tests, produce git commands
---

Prepare a TraceLog release. $ARGUMENTS

Use the release-manager agent to:
1. Check CHANGELOG.md [Unreleased] section
2. Run `make lint` and `make test` and report results
3. Update CHANGELOG.md with the new version and today's date
4. Produce the exact git commands needed to tag and release

If no version is specified in $ARGUMENTS, suggest the next patch version based on CHANGELOG.md.
