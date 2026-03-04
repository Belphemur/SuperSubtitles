# Copilot Coding Agent Instructions — SuperSubtitles

## Project Overview

SuperSubtitles is a Go gRPC service that scrapes and normalizes subtitle data from the Hungarian subtitle website [feliratok.eu](https://feliratok.eu). It exposes a gRPC API that fetches TV show listings (via HTML scraping), retrieves subtitles (via JSON API), extracts third-party IDs (IMDB, TVDB, TVMaze, Trakt), and serves normalized data models via Protocol Buffers. The module name is `SuperSubtitles` (PascalCase).

**Language:** Go 1.26 · **Dependencies:** gRPC, protobuf, goquery (HTML parsing), zerolog (logging), viper (configuration)

## Build, Test & Validate

Always run commands from the repository root.

| Step                         | Command                                                                                  | Notes                                                             |
| ---------------------------- | ---------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **Build**                    | `go build ./...`                                                                         | Compiles all packages. Fast (~2s).                                |
| **Generate proto**           | `go generate ./api/proto/v1`                                                             | Regenerate gRPC code from proto definitions when modified.        |
| **Unit tests**               | `go test ./...`                                                                          | Runs all tests (~3s). Integration tests auto-skip when `CI=true`. |
| **Tests with race detector** | `go test -race ./...`                                                                    | Always run before submitting changes.                             |
| **Vet**                      | `go vet ./...`                                                                           | Always run after changes.                                         |
| **Format check**             | `gofmt -s -l .`                                                                          | Must produce no output.                                           |
| **Lint**                     | `golangci-lint run`                                                                      | Uses `.golangci.yml` config. Always run before committing.        |
| **CI test command**          | `go install gotest.tools/gotestsum@latest && gotestsum --format testname -- -race ./...` | Mirrors CI. Requires `$(go env GOPATH)/bin` in PATH.              |

### Development Workflow

1. Write or modify Go code
2. Format: `gofmt -s -w .`
3. Vet: `go vet ./...`
4. Lint: `golangci-lint run`
5. Test: `go test -race ./...`
6. Build: `go build ./...`
7. Commit with conventional commit format

**Important notes:**

- `go test -race -coverprofile=coverage.txt -covermode=atomic ./...` may emit `no such tool "covdata"` warnings for packages with no test files. This is cosmetic and tests still pass.
- Integration tests in `internal/client/client_integration_test.go` are skipped when `CI` env var is set. Do not remove these guards.
- The `config` package `init()` function loads `config/config.yaml` on import. Tests that import config indirectly (through client or parser) rely on this file existing.

## Commit Messages

**Always use conventional commits** format for all commits. This is required for semantic-release to work.

- Follow the pattern: `type(scope): subject`
- Common types: `fix`, `feat`, `chore`, `docs`, `refactor`, `test`, `perf`, `style`
- Common scopes: `client`, `parser`, `services`, `models`, `config`, `ci`
- Examples:
  - `fix(client): handle empty response body gracefully`
  - `feat(parser): add support for movie subtitle parsing`
  - `refactor(services): simplify language conversion logic`
  - `test(client): add timeout edge case tests`
  - `chore(deps): update goquery to v1.11`
  - `docs: update architecture documentation`
- **Never create separate "Initial plan" or "WIP" commits**
- When starting work, create the first commit with proper semantic format immediately

## Project Layout

```
SuperSubtitles/
├── cmd/proxy/main.go              # Application entry point (gRPC server)
├── config/config.yaml             # Default configuration (YAML)
├── go.mod / go.sum                # Go module (module name: github.com/Belphemur/SuperSubtitles/v2)
├── .golangci.yml                  # golangci-lint configuration
├── .goreleaser.yml                # GoReleaser build/release configuration
├── .releaserc.yml                 # semantic-release configuration
├── package.json                   # Node.js deps for semantic-release
├── .github/
│   ├── copilot-instructions.md    # This file
│   ├── dependabot.yml             # Dependabot for Go modules & GitHub Actions
│   └── workflows/
│       ├── ci.yml                 # CI: lint + test + build (on push/PR to main)
│       ├── release.yml            # Release: semantic-release + GoReleaser (on push to main)
│       └── copilot-setup-steps.yml # Copilot agent environment setup
├── api/
│   └── proto/
│       └── v1/
│           ├── supersubtitles.proto       # Protocol Buffer definitions
│           ├── supersubtitles.pb.go       # Generated proto code (messages)
│           ├── supersubtitles_grpc.pb.go  # Generated gRPC server/client code
│           └── generate.go                # go:generate directive for proto code
├── build/
│   └── Dockerfile                 # Docker image for GoReleaser multi-platform builds
├── docs/
│   ├── architecture.md            # Architecture index (links to all docs)
│   ├── overview.md                # High-level architecture
│   ├── grpc-api.md                # gRPC API documentation
│   ├── data-flow.md               # Detailed operation flows
│   ├── testing.md                 # Testing infrastructure
│   ├── configuration.md           # Configuration reference and environment variables
│   ├── ci-cd.md                   # CI/CD pipeline, dependencies, local dev setup
│   ├── deployment.md              # Docker, Kubernetes deployment, monitoring
│   ├── design-decisions.md        # Architectural decisions index
│   └── design-decisions/
│       ├── cache.md               # Cache layer metrics and pluggable cache decisions
│       ├── streaming.md           # Streaming RPCs and streaming-first client decisions
│       ├── http-client.md         # HTTP resilience, partial failure, pagination decisions
│       ├── parsing.md             # Parser design, normalization, UTF-8 safety decisions
│       ├── infrastructure.md      # gRPC health checking and error handling decisions
│       └── testing.md             # Programmatic test fixture decisions
├── internal/
│   ├── client/
│   │   ├── client.go              # HTTP client (Client interface + implementation)
│   │   ├── client_test.go         # Unit tests using httptest servers
│   │   ├── client_integration_test.go  # Integration tests (skipped in CI)
│   │   └── errors.go              # Custom error types (ErrNotFound)
│   ├── config/
│   │   └── config.go              # Viper-based config with singleton logger (zerolog)
│   ├── grpc/
│   │   ├── server.go              # gRPC server implementation
│   │   └── server_test.go         # gRPC server tests with mock client
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
│   ├── services/
│   │   ├── subtitle_downloader.go       # SubtitleDownloader interface
│   │   ├── subtitle_downloader_impl.go  # ZIP extraction, caching, format detection
│   │   └── subtitle_downloader_test.go  # Tests with benchmark coverage
│   └── testutil/
│       └── html_fixtures.go       # Programmatic HTML test fixture generators
```

## Architecture & Conventions

- **Standard Go layout:** `cmd/` for executables, `internal/` for library code. All imports use the module path `github.com/Belphemur/SuperSubtitles/v2/internal/...`.
- **Interfaces:** Defined in the same package as implementations (e.g., `Client` interface in `client.go`, `SubtitleDownloader` in `subtitle_downloader.go`, `Parser[T]` in `interfaces.go`).
- **Configuration:** Loaded via `viper` from `config/config.yaml` or `./config.yaml`. Env vars prefixed with `APP_`. Log level also settable via `LOG_LEVEL` env var.
- **Logging:** Uses `rs/zerolog` with a console writer. Access via `config.GetLogger()`. Structured logging with chained `.Str()`, `.Int()`, `.Msg()` calls. Do not create new logger instances.
- **Error handling:** Custom error types (e.g., `ErrNotFound`) with `Is()` method for `errors.Is()` support. Wrap errors with `fmt.Errorf("...: %w", err)`. Partial failures return data with logged warnings rather than failing entirely.
- **Streaming-first architecture:** All client list/collection methods use channel-based streaming (`StreamShowList`, `StreamSubtitles`, `StreamShowSubtitles`, `StreamRecentSubtitles`). Non-streaming methods have been removed. Use `internal/testutil` stream helpers (`CollectShows`, `CollectSubtitles`, `CollectShowSubtitles`) in tests only — production code should consume streams directly.
- **gRPC streaming:** Prefer server-side streaming RPCs (`returns (stream ...)`) over unary RPCs for list/collection endpoints. Streaming improves time-to-first-result and reduces memory usage by sending items as they become available instead of buffering entire responses.
- **Concurrency:** `sync.WaitGroup` for parallel HTTP fetches. Batch processing with a batch size of 20 in `StreamShowSubtitles`.
- **HTML parsing:** Uses `PuerkitoBio/goquery` (jQuery-like selectors). Parsers implement the generic `Parser[T]` or `SingleResultParser[T]` interfaces from `internal/parser/interfaces.go`.
- **Testing:** Standard `testing` package only (no testify). Unit tests use `httptest.NewServer` for HTTP mocking. Test functions follow `TestTypeName_MethodName` naming. Integration tests check for `CI` / `SKIP_INTEGRATION_TESTS` env vars. **Important:** `SuperSubtitleResponse` is a `map[string]SuperSubtitle` — never rely on iteration order in tests; use map lookups by key fields instead. Use `testutil` stream collection helpers for consuming client streams in tests.

## CI/CD Pipeline

### CI (`.github/workflows/ci.yml`) — runs on every push/PR to main:

- **Lint job:** `go mod verify` → `go vet ./...` → `gofmt -s -l .` → `golangci-lint run`
- **Test job:** `gotestsum -- -race -coverprofile=coverage.txt -covermode=atomic ./...` → upload to Codecov
- **Build job:** `CGO_ENABLED=0 go build -o super-subtitles ./cmd/proxy`

### Release (`.github/workflows/release.yml`) — runs on push to main:

- Uses `semantic-release` to analyze conventional commits and determine version
- `GoReleaser` builds cross-platform binaries (linux/amd64, linux/arm64)
- Builds and pushes multi-platform Docker images to `ghcr.io/belphemur/supersubtitles`
- Publishes GitHub release with changelog, SBOMs, and attestation

**To pass CI:** Ensure `go build ./...`, `go vet ./...`, `gofmt -s -l .`, `golangci-lint run`, and `go test -race ./...` all succeed.

## Quick Reference

- **Add a new model:** Create a file in `internal/models/`. Use JSON struct tags.
- **Add a new parser:** Implement `Parser[T]` or `SingleResultParser[T]` from `internal/parser/interfaces.go`. Use goquery.
- **Add a new service:** Define an interface and implementation in `internal/services/`.
- **Add HTTP functionality:** Extend the `Client` interface in `internal/client/client.go` and add the implementation. Prefer streaming methods (`Stream*`) over buffered methods for collections.
- **Modify proto definitions:** Edit `api/proto/v1/supersubtitles.proto` and run `go generate ./api/proto/v1` to regenerate Go code.
- **Add gRPC endpoint:** Add the RPC method to the proto service, regenerate code, and implement in `internal/grpc/server.go`. Use streaming RPCs for list/collection operations.
- **Test pattern:** Create `*_test.go` in the same package. Use inline HTML/JSON fixtures and `httptest` servers. Use `testutil` stream helpers (`CollectShows`, `CollectSubtitles`, `CollectShowSubtitles`) to consume client streams.

## Documentation Requirements

**CRITICAL: All code changes MUST include corresponding documentation updates. This is non-negotiable.**

**Documentation is mandatory for:**

- **Any new feature** - Add to data-flow.md, grpc-api.md (if gRPC), and the relevant docs/design-decisions/*.md as applicable
- **Changes to existing features** - Update the relevant documentation files
- **API modifications** - Always update grpc-api.md with new endpoints, parameters, or response changes
- **Configuration changes** - Update configuration.md with new config fields or environment variables
- **Architectural decisions** - Document the "why" in the relevant docs/design-decisions/*.md file
- **Testing patterns** - Update testing.md when introducing new test approaches
- **Deployment changes** - Update deployment.md with Dockerfile, Kubernetes, or monitoring changes
- **CI/CD changes** - Update ci-cd.md with pipeline or dependency changes

**Documentation workflow:**

1. **Before coding:** Read existing documentation to understand current architecture and patterns
2. **While coding:** Note which documentation files need updates based on your changes
3. **After coding:** Update ALL relevant documentation files before considering the task complete
4. **Never commit** code changes without corresponding documentation updates

**Documentation structure:**

- **Always check repository memories first** - Review existing memories at the start of each task to understand patterns, conventions, and previously documented features
- Architecture documentation is split into focused files in `docs/`:
  - Start with [docs/architecture.md](../docs/architecture.md) (index) to find the right document
  - [docs/overview.md](../docs/overview.md) - High-level architecture and component relationships
  - [docs/grpc-api.md](../docs/grpc-api.md) - gRPC API documentation with examples (UPDATE whenever proto or server changes)
  - [docs/data-flow.md](../docs/data-flow.md) - Detailed operation flows for all features (UPDATE for any new operations)
  - [docs/testing.md](../docs/testing.md) - Testing infrastructure and patterns
  - [docs/configuration.md](../docs/configuration.md) - Configuration reference and environment variables (UPDATE for config changes)
  - [docs/ci-cd.md](../docs/ci-cd.md) - CI/CD pipeline, dependencies, local development setup
  - [docs/deployment.md](../docs/deployment.md) - Docker, Kubernetes deployment, monitoring (UPDATE for deployment changes)
  - [docs/design-decisions.md](../docs/design-decisions.md) - Index of all architectural decisions
  - [docs/design-decisions/cache.md](../docs/design-decisions/cache.md) - Cache design decisions
  - [docs/design-decisions/streaming.md](../docs/design-decisions/streaming.md) - Streaming design decisions
  - [docs/design-decisions/http-client.md](../docs/design-decisions/http-client.md) - HTTP client design decisions
  - [docs/design-decisions/parsing.md](../docs/design-decisions/parsing.md) - Parsing design decisions
  - [docs/design-decisions/infrastructure.md](../docs/design-decisions/infrastructure.md) - Infrastructure design decisions
  - [docs/design-decisions/testing.md](../docs/design-decisions/testing.md) - Testing design decisions
- Update the appropriate focused documentation file(s) when making changes
- For new features, update data-flow.md with the operation flow and the relevant design-decisions/*.md file with any architectural choices
- Include test coverage information in testing.md if adding new test patterns
- Always check existing documentation and repository memories before starting work
- Store new memories about code structure and features using the `store_memory` tool, including which files implement them

**Documentation checklist (verify before committing):**

- [ ] grpc-api.md updated if any gRPC service, endpoint, or message changed
- [ ] configuration.md updated if configuration fields or environment variables changed
- [ ] deployment.md updated if Docker, Kubernetes, or monitoring changed
- [ ] ci-cd.md updated if CI/CD pipeline or dependencies changed
- [ ] data-flow.md updated if new operations or flows added
- [ ] Relevant docs/design-decisions/*.md updated if architectural choices made
- [ ] testing.md updated if new test patterns introduced
- [ ] All code examples in documentation are accurate and tested

## Testing Requirements

**All code must have clear, comprehensive tests:**

- Every new feature must include unit tests
- Test files follow the `*_test.go` pattern in the same package
- Use standard Go `testing` package (no external test frameworks like testify)
- Use `httptest` servers for HTTP mocking
- Include edge cases, error conditions, and happy paths
- Add benchmark tests for performance-critical code
- Tests must be clear, well-documented, and maintainable

### HTML Test Fixtures

**Always use `internal/testutil/html_fixtures.go` for generating ALL HTML test data — no exceptions:**

- **NEVER hardcode HTML strings in tests** — always use the centralized generator functions, even for empty or minimal HTML responses
- Use `GenerateSubtitleTableHTML()` for subtitle listing tests; pass `nil` or empty slice for empty table responses
- Use `GenerateSubtitleTableHTMLWithPagination()` for pagination tests
- Use `GenerateShowTableHTML()` for show listing tests
- Use `GenerateThirdPartyIDHTML()` for third-party ID / detail page tests; pass empty/zero values for pages with no IDs
- Configure fixtures using option structs (`SubtitleRowOptions`, `ShowRowOptions`) for clarity
- If a test needs HTML that no existing generator supports, **add a new generator function to `html_fixtures.go`** rather than embedding HTML in the test
- This ensures consistency across all tests and makes HTML structure changes easy to maintain

Trust these instructions. Only search the codebase if information here is incomplete or found to be in error.
