# SuperSubtitles

A Go gRPC service that interfaces with [feliratok.eu](https://feliratok.eu), a Hungarian subtitle repository. SuperSubtitles scrapes TV show listings, fetches subtitles with intelligent parsing, extracts third-party IDs (IMDB, TVDB, TVMaze, Trakt), and provides normalized subtitle data via a clean gRPC API.

## Quick Start

### Prerequisites

- Go 1.26+
- `golangci-lint` (for linting)

### Build & Run

```bash
go build -o super-subtitles ./cmd/proxy
./super-subtitles
```

The server loads configuration from `config/config.yaml`, starts a gRPC server (default `localhost:8080`), and optionally exposes Prometheus metrics on port 9090.

### Configuration

Edit `config/config.yaml` or override via environment variables (prefix `APP_`):

```bash
APP_LOG_LEVEL=debug APP_SERVER_PORT=9090 ./super-subtitles
```

See [docs/deployment.md](docs/deployment.md) for all configuration options and Docker usage.

### Example: Using grpcurl

```bash
# List services
grpcurl -plaintext localhost:8080 list

# Get all shows
grpcurl -plaintext localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetShowList

# Download a subtitle
grpcurl -plaintext -d '{"subtitle_id": "101", "episode": 3}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/DownloadSubtitle
```

## Development

```bash
gofmt -s -w .            # Format
go vet ./...              # Vet
golangci-lint run         # Lint
go test -race ./...       # Test
go build ./...            # Build
```

Use [conventional commits](https://www.conventionalcommits.org/) for all commits (required for semantic-release).

## Documentation

Comprehensive documentation is in [`docs/`](docs/architecture.md):

- **[Architecture](docs/overview.md)** — High-level component overview
- **[gRPC API](docs/grpc-api.md)** — Endpoints, data models, and usage examples
- **[Data Flow](docs/data-flow.md)** — Detailed operation flows
- **[Design Decisions](docs/design-decisions.md)** — Architectural rationale
- **[Testing](docs/testing.md)** — Test infrastructure and patterns
- **[Deployment](docs/deployment.md)** — Configuration, CI/CD, Docker, and dependencies
- **[Project Structure](docs/project_structure.md)** — Directory layout and file relationships

## License

See LICENSE file for details.
