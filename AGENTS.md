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
