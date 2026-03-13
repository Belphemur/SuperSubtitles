# SuperSubtitles Agent Guide

## Project Overview

SuperSubtitles is a Go 1.26 gRPC service that scrapes and normalizes subtitle data from feliratok.eu. The module name is `SuperSubtitles`, and internal imports use `github.com/Belphemur/SuperSubtitles/v2/internal/...`.

## Build And Validation

Run commands from the repository root.

- `go build ./...`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `gofmt -s -l .`
- `golangci-lint run`

Integration tests in `internal/client/client_integration_test.go` auto-skip when `CI=true`. The `internal/config` package loads `config/config.yaml` during import, so tests that reach config indirectly depend on that file being present.

## Core Conventions

- Keep the standard Go layout: `cmd/` for executables and `internal/` for library code.
- Define interfaces in the same package as their implementations.
- Use `config.GetLogger()` for logging; do not create new logger instances.
- Wrap errors with `fmt.Errorf("...: %w", err)` and prefer custom error types when callers need structured handling.
- Client collection endpoints are streaming-first: production code should consume `Stream*` APIs directly.
- Prefer server-side streaming gRPC RPCs for collection endpoints.
- Use `sync.WaitGroup` for parallel HTTP fetches and preserve the existing batch size of 20 in show-subtitle flows.

## Testing

- Use the standard `testing` package only.
- Use `httptest.NewServer` for HTTP mocking.
- Follow `TestTypeName_MethodName` naming.
- Never rely on map iteration order in tests.
- Use `internal/testutil/html_fixtures.go` for generated HTML fixtures instead of hardcoded HTML.
- Use `internal/testutil` stream collection helpers in tests only.

## Documentation

Code changes must include matching documentation updates.

- Update `docs/grpc-api.md` for API behavior changes.
- Update `docs/data-flow.md` for operational flow changes.
- Update the relevant `docs/design-decisions/*.md` file when the architectural rationale changes.
- Keep docs concise and behavioral. Avoid copying method names or code into non-design-decision docs.

## Commits

Use conventional commits: `type(scope): subject`.

Examples:

- `fix(services): map archive failures to gRPC preconditions`
- `docs: move workspace instructions to AGENTS`
- `chore(ci): tighten lint workflow`

<!-- BEGIN BEADS INTEGRATION -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Dolt-powered version control with native sync
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update <id> --claim --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task atomically**: `bd update <id> --claim`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Auto-Sync

bd automatically syncs via Dolt:

- Each write auto-commits to Dolt history
- Use `bd dolt push`/`bd dolt pull` for remote sync
- No manual export/import needed!

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

<!-- END BEADS INTEGRATION -->
