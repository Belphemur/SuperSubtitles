# SuperSubtitles — CI/CD Pipeline

## CI Workflow (`.github/workflows/ci.yml`)

Runs on every push and PR to `main`:

**Lint Job:**

```bash
go mod verify
go vet ./...
gofmt -s -l .
golangci-lint run
```

**Test Job:**

The test job uses a matrix strategy to split tests into 4 parallel groups, each with a `valkey/valkey:latest` service container (Redis-compatible, exposed on `localhost:6379`) and `REDIS_ADDRESS=localhost:6379`:

| Group | Packages |
| --- | --- |
| `parser-models-errors` | `./internal/parser/...`, `./internal/models/...`, `./internal/apperrors/...` |
| `client` | `./internal/client/...` |
| `services-grpc-metrics` | `./internal/services/...`, `./internal/grpc/...`, `./internal/metrics/...` |
| `cache` | `./internal/cache/...` |

Each group runs:

```bash
# Service: valkey/valkey:latest on localhost:6379
REDIS_ADDRESS=localhost:6379 gotestsum --format testname -- -race -coverprofile=coverage.txt -covermode=atomic <packages>
# Upload coverage to Codecov with per-group flag
```

**Build Job:**

```bash
CGO_ENABLED=0 go build -o super-subtitles ./cmd/proxy
# Upload artifact
```

## Release Workflow (`.github/workflows/release.yml`)

Runs on push to `main`:

1. **Semantic Release**: Analyzes conventional commits to determine version
2. **GoReleaser Build**: Cross-platform binaries (linux/amd64, linux/arm64)
3. **Docker Images**: Multi-platform images pushed to `ghcr.io/belphemur/supersubtitles`
4. **GitHub Release**: Published with changelog, SBOMs, and attestation

## Copilot Setup (`.github/workflows/copilot-setup-steps.yml`)

Prepares Copilot agent environment:

- Installs Go 1.26
- Installs gopls (Go language server)
- Installs golangci-lint
- Downloads Go dependencies

## Dependencies

| Package                              | Purpose                                 | Version Constraint |
| ------------------------------------ | --------------------------------------- | ------------------ |
| `github.com/PuerkitoBio/goquery`     | jQuery-like HTML parsing                | Latest             |
| `github.com/rs/zerolog`              | Structured JSON/console logging         | Latest             |
| `github.com/spf13/viper`             | Configuration management                | Latest             |
| `github.com/hashicorp/golang-lru/v2` | In-memory LRU cache (memory backend)    | v2                 |
| `github.com/redis/go-redis/v9`       | Redis/Valkey client (redis cache backend) | v9               |
| `github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus` | gRPC Prometheus interceptors | Latest |
| `github.com/prometheus/client_golang` | Prometheus client library              | Latest             |
| `github.com/failsafe-go/failsafe-go` | HTTP retry and resilience policies      | Latest             |
| `archive/zip` (stdlib)               | ZIP file extraction for season packs    | stdlib             |

### Dependency Management

Dependencies are managed via Go modules (`go.mod`). Dependabot and Renovate are configured to automatically update dependencies.

## Local Development

### Prerequisites

- Go 1.26+
- golangci-lint (for linting)
- gotestsum (optional, for pretty test output)

### Setup

```bash
# Clone repository
git clone https://github.com/Belphemur/SuperSubtitles.git
cd SuperSubtitles

# Install dependencies
go mod download

# Run tests
go test -race ./...

# Build
go build ./cmd/proxy
```
