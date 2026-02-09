# SuperSubtitles — Configuration & Deployment

## Configuration

Configuration is loaded from `config/config.yaml` using Viper. Environment variables are supported with `APP_` prefix.

### Configuration Fields

| Field                     | Description                           | Default                                                                            | Env Var                       |
| ------------------------- | ------------------------------------- | ---------------------------------------------------------------------------------- | ----------------------------- |
| `proxy_connection_string` | HTTP proxy URL (optional)             | `""`                                                                               | `APP_PROXY_CONNECTION_STRING` |
| `super_subtitle_domain`   | Base URL for feliratok.eu             | `https://feliratok.eu`                                                             | `APP_SUPER_SUBTITLE_DOMAIN`   |
| `client_timeout`          | HTTP client timeout (Go duration)     | `30s`                                                                              | `APP_CLIENT_TIMEOUT`          |
| `user_agent`              | User-Agent header for HTTP requests   | `Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0` | `APP_USER_AGENT`              |
| `server.port`             | Server listening port                 | `8080`                                                                             | `APP_SERVER_PORT`             |
| `server.address`          | Server listening address              | `localhost`                                                                        | `APP_SERVER_ADDRESS`          |
| `log_level`               | Zerolog level (debug/info/warn/error) | `info`                                                                             | `LOG_LEVEL` (direct bind)     |

### Example Configuration

```yaml
proxy_connection_string: ""
super_subtitle_domain: "https://feliratok.eu"
client_timeout: "30s"
user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0"
log_level: "info"

server:
  port: 8080
  address: "localhost"
```

### Environment Variables

```bash
# Override log level
export LOG_LEVEL=debug

# Override domain
export APP_SUPER_SUBTITLE_DOMAIN=https://feliratok.eu

# Override timeout
export APP_CLIENT_TIMEOUT=60s
```

## CI/CD Pipeline

### CI Workflow (`.github/workflows/ci.yml`)

Runs on every push and PR to `main`:

**Lint Job:**

```bash
go mod verify
go vet ./...
gofmt -s -l .
golangci-lint run
```

**Test Job:**

```bash
gotestsum --format testname -- -race -coverprofile=coverage.txt -covermode=atomic ./...
# Upload to Codecov
```

**Build Job:**

```bash
CGO_ENABLED=0 go build -o super-subtitles ./cmd/proxy
# Upload artifact
```

### Release Workflow (`.github/workflows/release.yml`)

Runs on push to `main`:

1. **Semantic Release**: Analyzes conventional commits to determine version
2. **GoReleaser Build**: Cross-platform binaries (linux/amd64, linux/arm64)
3. **Docker Images**: Multi-platform images pushed to `ghcr.io/belphemur/supersubtitles`
4. **GitHub Release**: Published with changelog, SBOMs, and attestation

### Copilot Setup (`.github/workflows/copilot-setup-steps.yml`)

Prepares Copilot agent environment:

- Installs Go 1.25
- Installs gopls (Go language server)
- Installs golangci-lint
- Downloads Go dependencies

## Dependencies

| Package                              | Purpose                                 | Version Constraint |
| ------------------------------------ | --------------------------------------- | ------------------ |
| `github.com/PuerkitoBio/goquery`     | jQuery-like HTML parsing                | Latest             |
| `github.com/rs/zerolog`              | Structured JSON/console logging         | Latest             |
| `github.com/spf13/viper`             | Configuration management                | Latest             |
| `github.com/hashicorp/golang-lru/v2` | LRU cache for ZIP file caching (1h TTL) | v2                 |
| `archive/zip` (stdlib)               | ZIP file extraction for season packs    | stdlib             |

### Dependency Management

Dependencies are managed via Go modules (`go.mod`). Dependabot is configured to automatically update dependencies weekly.

## Docker Deployment

### Multi-Platform Images

GoReleaser builds Docker images for:

- `linux/amd64`
- `linux/arm64`

Images are pushed to: `ghcr.io/belphemur/supersubtitles:latest` and `ghcr.io/belphemur/supersubtitles:v<version>`

### Dockerfile

Located at `build/Dockerfile`, used by GoReleaser for multi-platform builds.

## Local Development

### Prerequisites

- Go 1.25+
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
go test ./...

# Build
go build ./cmd/proxy
```

### Development Workflow

1. Write or modify Go code
2. Format: `gofmt -s -w .`
3. Vet: `go vet ./...`
4. Lint: `golangci-lint run`
5. Test: `go test -race ./...`
6. Build: `go build ./...`
7. Commit with conventional commit format

## Monitoring

### Logging

The application uses structured logging with zerolog:

```go
logger := config.GetLogger()
logger.Info().
    Str("showID", showID).
    Int("subtitleCount", count).
    Msg("Fetched subtitles")
```

Log levels: `debug`, `info`, `warn`, `error`

### Metrics

Currently no metrics collection. Future consideration for:

- Request counts
- Response times
- Error rates
- Cache hit rates
