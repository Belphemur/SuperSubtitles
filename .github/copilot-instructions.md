# Copilot Coding Agent Instructions — SuperSubtitles

## Project Overview

SuperSubtitles is a Go proxy service that scrapes and normalizes subtitle data from the Hungarian subtitle website [feliratok.eu](https://feliratok.eu). It fetches TV show listings (via HTML scraping), retrieves subtitles (via JSON API), extracts third-party IDs (IMDB, TVDB, TVMaze, Trakt), and converts everything into normalized data models. The module name is `SuperSubtitles` (PascalCase).

**Language:** Go 1.25 · **Dependencies:** goquery (HTML parsing), zerolog (logging), viper (configuration)

## Build, Test & Validate

Always run commands from the repository root: `/home/runner/work/SuperSubtitles/SuperSubtitles` (or wherever the repo is cloned).

| Step | Command | Notes |
|------|---------|-------|
| **Build** | `go build ./...` | Compiles all packages. Fast (~2s). |
| **Unit tests** | `go test ./...` | Runs all tests (~3s). Integration tests auto-skip when `CI=true`. |
| **Tests with race detector** | `go test -race ./...` | Use this before submitting changes. |
| **Vet/lint** | `go vet ./...` | Always run after changes. No external linter configured. |
| **CI test command** | `go install gotest.tools/gotestsum@latest && gotestsum --format testname -- -race ./...` | Mirrors CI. Requires `$(go env GOPATH)/bin` in PATH. |

**Important notes:**
- There are no `Makefile`, `npm`, or other build tools. Pure Go toolchain only.
- `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` may emit `no such tool "covdata"` warnings for packages with no test files. This is cosmetic and tests still pass.
- Integration tests in `internal/client/client_integration_test.go` are skipped when `CI` env var is set. Do not remove these guards.
- The `config` package `init()` function loads `config/config.yaml` on import. Tests that import config indirectly (through client or parser) rely on this file existing.

## Project Layout

```
SuperSubtitles/
├── cmd/proxy/main.go              # Application entry point
├── config/config.yaml             # Default configuration (YAML)
├── go.mod / go.sum                # Go module (module name: SuperSubtitles)
├── .github/workflows/go_test.yml  # CI: runs go test with race detector + coverage
├── internal/
│   ├── client/
│   │   ├── client.go              # HTTP client (Client interface + implementation)
│   │   ├── client_test.go         # Unit tests using httptest servers
│   │   ├── client_integration_test.go  # Integration tests (skipped in CI)
│   │   └── errors.go              # Custom error types (ErrNotFound)
│   ├── config/
│   │   └── config.go              # Viper-based config with singleton logger (zerolog)
│   ├── models/
│   │   ├── show.go                # Show struct
│   │   ├── subtitle.go            # Subtitle, SubtitleCollection, SuperSubtitleResponse
│   │   ├── show_subtitles.go      # ShowSubtitles (composite model)
│   │   ├── quality.go             # Quality enum with JSON marshaling
│   │   ├── third_party_ids.go     # IMDB/TVDB/TVMaze/Trakt IDs
│   │   └── update_check.go        # Update check request/response models
│   ├── parser/
│   │   ├── interfaces.go          # Generic Parser[T] and SingleResultParser[T] interfaces
│   │   ├── show_parser.go         # HTML parser for show listings (goquery)
│   │   ├── show_parser_test.go    # Tests with inline HTML fixtures
│   │   ├── third_party_parser.go  # HTML parser for third-party IDs
│   │   └── third_party_parser_test.go
│   └── services/
│       ├── subtitle_converter.go       # SubtitleConverter interface
│       ├── subtitle_converter_impl.go  # Converts API response → normalized models
│       └── subtitle_converter_test.go  # Includes benchmark tests
```

## Architecture & Conventions

- **Standard Go layout:** `cmd/` for executables, `internal/` for library code. All imports use the module path `SuperSubtitles/internal/...`.
- **Interfaces:** Defined in the same package as implementations (e.g., `Client` interface in `client.go`, `SubtitleConverter` in `subtitle_converter.go`, `Parser[T]` in `interfaces.go`).
- **Configuration:** Loaded via `viper` from `config/config.yaml` or `./config.yaml`. Env vars prefixed with `APP_`. Log level also settable via `LOG_LEVEL` env var.
- **Logging:** Uses `rs/zerolog` with a console writer. Access via `config.GetLogger()`. Structured logging with chained `.Str()`, `.Int()`, `.Msg()` calls. Do not create new logger instances.
- **Error handling:** Custom error types (e.g., `ErrNotFound`) with `Is()` method for `errors.Is()` support. Wrap errors with `fmt.Errorf("...: %w", err)`. Partial failures return data with logged warnings rather than failing entirely.
- **Concurrency:** `sync.WaitGroup` for parallel HTTP fetches. Batch processing with a batch size of 20 in `GetShowSubtitles`.
- **HTML parsing:** Uses `PuerkitoBio/goquery` (jQuery-like selectors). Parsers implement the generic `Parser[T]` or `SingleResultParser[T]` interfaces from `internal/parser/interfaces.go`.
- **Testing:** Standard `testing` package only (no testify). Unit tests use `httptest.NewServer` for HTTP mocking. Test functions follow `TestTypeName_MethodName` naming. Integration tests check for `CI` / `SKIP_INTEGRATION_TESTS` env vars.

## CI Pipeline

The single GitHub Actions workflow (`.github/workflows/go_test.yml`) runs on every push and pull request:
1. Checks out the code
2. Sets up Go using the version from `go.mod`
3. Installs `gotestsum`
4. Runs: `gotestsum --junitfile test-results/junit.xml --format testname -- -race -coverprofile=coverage.txt -covermode=atomic ./...`
5. Uploads test results and coverage to Codecov

**To pass CI:** Ensure `go build ./...`, `go vet ./...`, and `go test -race ./...` all succeed.

## Quick Reference

- **Add a new model:** Create a file in `internal/models/`. Use JSON struct tags.
- **Add a new parser:** Implement `Parser[T]` or `SingleResultParser[T]` from `internal/parser/interfaces.go`. Use goquery.
- **Add a new service:** Define an interface and implementation in `internal/services/`.
- **Add HTTP functionality:** Extend the `Client` interface in `internal/client/client.go` and add the implementation.
- **Test pattern:** Create `*_test.go` in the same package. Use inline HTML/JSON fixtures and `httptest` servers.

Trust these instructions. Only search the codebase if information here is incomplete or found to be in error.
