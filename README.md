# SuperSubtitles

A Go gRPC service that interfaces with [feliratok.eu](https://feliratok.eu), a Hungarian subtitle repository. SuperSubtitles scrapes TV show listings, fetches subtitles with intelligent parsing, extracts third-party IDs (IMDB, TVDB, TVMaze, Trakt), and provides normalized subtitle data via a clean gRPC API. Supports downloading individual episodes from season packs with intelligent caching.

## Features

- **gRPC API** — Clean, type-safe API with Protocol Buffers for all operations
- **Show Listing Scraping** — Fetches and deduplicates TV shows from multiple feliratok.eu endpoints in parallel
- **Subtitle Fetching** — Retrieves subtitle metadata (language, quality, season/episode, uploader, download URLs) via JSON API
- **Third-Party ID Extraction** — Automatically extracts IMDB, TVDB, TVMaze, and Trakt IDs from show detail pages
- **Data Normalization** — Converts Hungarian language names to ISO 639-1 codes, parses quality strings (360p–2160p), builds proper download URLs
- **Update Checking** — Queries for new subtitles since a given content ID
- **Smart Subtitle Downloads**:
  - Supports multiple formats (SRT, ASS, VTT, SUB, ZIP)
  - Episode extraction from season pack ZIPs with intelligent file matching
  - Intelligent content-type detection via magic numbers and MIME types
  - LRU caching with 1-hour TTL for ZIP files
  - ZIP bomb detection to prevent malicious archives
- **Partial Failure Resilience** — Returns available data even when some endpoints fail, logging warnings instead of crashing
- **Graceful Shutdown** — Handles SIGTERM/SIGINT with proper cleanup of in-flight requests

## Quick Start

### Prerequisites

- Go 1.25 or later
- `protoc` (Protocol Buffer compiler) for regenerating proto code
- `golangci-lint` (for linting)
- `pnpm` (for semantic-release dependencies, optional)

### Build

```bash
# Build all packages
go build ./...

# Build the proxy binary
go build -o super-subtitles ./cmd/proxy
```

### Run

```bash
# Run the gRPC server
./super-subtitles
```

The server will:

1. Load configuration from `config/config.yaml`
2. Start gRPC server on configured address and port
3. Enable gRPC reflection for introspection tools
4. Wait for requests (use grpcurl, client SDKs, or Postman to interact)

### Configuration

Create or edit `config/config.yaml`:

```yaml
proxy_connection_string: "" # Optional HTTP proxy URL
super_subtitle_domain: "https://feliratok.eu"
client_timeout: "30s"
user_agent: "Sub-Zero/2"
server:
  address: "localhost"
  port: 8080
log_level: "info" # debug, info, warn, error
```

Environment variables (with `APP_` prefix) override YAML settings:

```bash
APP_CLIENT_TIMEOUT=60s APP_LOG_LEVEL=debug ./super-subtitles
```

## Architecture & Design

For detailed information about the architecture, module organization, design decisions, and data flow, see **[docs/architecture.md](docs/architecture.md)**.

## Development

### Generate Proto Code

If you modify the proto definitions, regenerate the Go code using `go generate`:

```bash
# Generate from proto definitions
go generate ./api/proto/v1

# Or from within the proto directory
cd api/proto/v1 && go generate
```

**Note:** Required tools (`protoc-gen-go`, `protoc-gen-go-grpc`) are automatically installed if not present.

### Run Tests

```bash
# All unit tests
go test ./...

# With race detector (recommended before commits)
go test -race ./...

# Unit tests only (skip integration tests)
CI=true go test ./...

# With coverage
go test -race -coverprofile=coverage.txt -covermode=atomic ./...
```

### Code Quality

```bash
# Format code
gofmt -s -w .

# Vet for potential issues
go vet ./...

# Full linting
golangci-lint run

# All checks together
go mod verify && go vet ./... && gofmt -s -l . && golangci-lint run && go test -race ./...
```

### Project Structure

See [docs/project_structure.md](docs/project_structure.md) for the complete directory layout, module organization, file relationships, and naming conventions.

## Commit Messages

Use conventional commits (required for semantic-release):

```
feat(client): add HTTP compression support
fix(parser): handle missing year headers gracefully
refactor(services): simplify language conversion
test(downloader): add ZIP bomb detection tests
docs: update architecture guide
chore(deps): update goquery to v1.11
```

## API Overview

SuperSubtitles exposes a gRPC API with 6 main operations. For complete API documentation, see **[docs/grpc-api.md](docs/grpc-api.md)**.

### gRPC Service

```protobuf
service SuperSubtitlesService {
  rpc GetShowList(GetShowListRequest) returns (GetShowListResponse);
  rpc GetSubtitles(GetSubtitlesRequest) returns (GetSubtitlesResponse);
  rpc GetShowSubtitles(GetShowSubtitlesRequest) returns (GetShowSubtitlesResponse);
  rpc CheckForUpdates(CheckForUpdatesRequest) returns (CheckForUpdatesResponse);
  rpc DownloadSubtitle(DownloadSubtitleRequest) returns (DownloadSubtitleResponse);
  rpc GetRecentSubtitles(GetRecentSubtitlesRequest) returns (GetRecentSubtitlesResponse);
}
```

### Example: Using grpcurl

```bash
# List all available services
grpcurl -plaintext localhost:8080 list

# Get all shows
grpcurl -plaintext localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetShowList

# Get subtitles for a specific show
grpcurl -plaintext -d '{"show_id": 1234}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetSubtitles

# Download a subtitle
grpcurl -plaintext -d '{"download_url": "http://...", "subtitle_id": "101", "episode": 3}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/DownloadSubtitle
```

### Internal Client Interface

The gRPC server wraps the internal `Client` interface:

```go
type Client interface {
    GetShowList(ctx context.Context) ([]Show, error)
    GetSubtitles(ctx context.Context, showID int) (*SubtitleCollection, error)
    GetShowSubtitles(ctx context.Context, shows []Show) ([]ShowSubtitles, error)
    CheckForUpdates(ctx context.Context, contentID string) (*UpdateCheckResult, error)
    DownloadSubtitle(ctx context.Context, downloadURL string, req DownloadRequest) (*DownloadResult, error)
    GetRecentSubtitles(ctx context.Context, sinceID int) ([]ShowSubtitles, error)
}
```

## Dependencies

| Package                                     | Purpose                          |
| ------------------------------------------- | -------------------------------- |
| `google.golang.org/grpc` v1.78.0            | gRPC framework                   |
| `google.golang.org/protobuf` v1.36.11       | Protocol Buffers runtime         |
| `github.com/PuerkitoBio/goquery` v1.11.0    | jQuery-like HTML parsing         |
| `github.com/rs/zerolog` v1.34.0             | Structured console/JSON logging  |
| `github.com/spf13/viper` v1.21.0            | Configuration management         |
| `github.com/hashicorp/golang-lru/v2` v2.0.7 | LRU cache for subtitle downloads |
| `github.com/andybalholm/brotli` v1.2.0      | Brotli compression support       |
| `github.com/klauspost/compress` v1.18.3     | Zstd compression support         |

## CI/CD

### Continuous Integration (`.github/workflows/ci.yml`)

Runs on every push and pull request to `main`:

- **Lint:** `go mod verify`, `go vet`, `gofmt`, `golangci-lint`
- **Test:** `go test -race` with coverage upload to Codecov
- **Build:** Cross-platform binary builds (linux/amd64, linux/arm64)

### Release (`.github/workflows/release.yml`)

Runs on push to `main`:

1. **Versioning:** `semantic-release` analyzes conventional commits
2. **Build:** `GoReleaser` builds binaries and Docker images
3. **Publish:** Creates GitHub release with changelog, SBOMs, and attestations
4. **Docker:** Pushes multi-platform images to `ghcr.io/belphemur/supersubtitles`

## Testing

The project includes:

- **Unit tests** with mocked HTTP servers (`httptest`)
- **Integration tests** that call the real feliratok.eu (skipped in CI)
- **Benchmark tests** for performance-critical code (subtitle conversion, ZIP extraction)
- **No external test frameworks** — uses only Go's standard `testing` package

## Docker

Build and run in Docker:

```bash
# Build image locally
docker build -t supersubtitles:latest .

# Run container
docker run --rm -e APP_LOG_LEVEL=debug supersubtitles:latest
```

Multi-platform images are automatically published to the GitHub Container Registry.

## Logging

Logs are structured JSON/console output via `zerolog`. Set log level via config or environment:

```bash
LOG_LEVEL=debug ./super-subtitles
```

Example debug output:

```
7:21PM INF Fetching show list from multiple endpoints in parallel baseURL=https://feliratok.eu
7:21PM INF Starting HTML parsing for shows
7:21PM INF Completed HTML parsing for shows total_shows=47
```

## Contributing

1. Follow the **project structure** and **code style** conventions (enforced by linters)
2. Write **unit tests** for new features (aim for >80% coverage)
3. Use **conventional commits** for proper semantic versioning
4. Run the full **test and lint suite** before submitting PRs
5. See [.github/copilot-instructions.md](.github/copilot-instructions.md) for detailed developer guidelines

## License

See LICENSE file for details.

## References

- **Architecture:** [docs/architecture.md](docs/architecture.md)
- **feliratok.eu:** https://feliratok.eu
- **Go:** https://golang.org
- **GoReleaser:** https://goreleaser.com
- **Semantic Release:** https://semantic-release.gitbook.io
