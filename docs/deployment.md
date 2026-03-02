# SuperSubtitles — Configuration & Deployment

## Configuration

Configuration is loaded from `config/config.yaml` using Viper. Environment variables are supported with `APP_` prefix, with nested keys mapped by replacing `.` with `_` (for example, `server.address` → `APP_SERVER_ADDRESS`).

### Configuration Fields

| Field                     | Description                           | Default                                                                            | Env Var                        |
| ------------------------- | ------------------------------------- | ---------------------------------------------------------------------------------- | ------------------------------ |
| `proxy_connection_string` | HTTP proxy URL (optional)             | `""`                                                                               | `APP_PROXY_CONNECTION_STRING`  |
| `super_subtitle_domain`   | Base URL for feliratok.eu             | `https://feliratok.eu`                                                             | `APP_SUPER_SUBTITLE_DOMAIN`    |
| `client_timeout`          | HTTP client timeout (Go duration)     | `30s`                                                                              | `APP_CLIENT_TIMEOUT`           |
| `user_agent`              | User-Agent header for HTTP requests   | `Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0` | `APP_USER_AGENT`               |
| `server.port`             | Server listening port                 | `8080`                                                                             | `APP_SERVER_PORT`              |
| `server.address`          | Server listening address              | `localhost`                                                                        | `APP_SERVER_ADDRESS`           |
| `log_level`               | Zerolog level (debug/info/warn/error) | `info`                                                                             | `APP_LOG_LEVEL` or `LOG_LEVEL` |
| `log_format`              | Log output format (console/json); defaults to console for unrecognized values | `console`                                                                          | `APP_LOG_FORMAT` or `LOG_FORMAT` |
| `cache.size`              | Maximum entries in LRU ZIP cache      | `2000`                                                                             | `APP_CACHE_SIZE`               |
| `cache.ttl`               | LRU cache TTL (Go duration)           | `24h`                                                                              | `APP_CACHE_TTL`                |
| `cache.type`              | Cache backend (`memory` or `redis`)   | `memory`                                                                           | `APP_CACHE_TYPE`               |
| `cache.redis.address`     | Redis/Valkey server address           | `localhost:6379`                                                                   | `APP_CACHE_REDIS_ADDRESS`      |
| `cache.redis.password`    | Redis/Valkey password (optional)      | `""`                                                                               | `APP_CACHE_REDIS_PASSWORD`     |
| `cache.redis.db`          | Redis/Valkey database number          | `0`                                                                                | `APP_CACHE_REDIS_DB`           |
| `metrics.enabled`         | Enable Prometheus metrics endpoint    | `true`                                                                             | `APP_METRICS_ENABLED`          |
| `metrics.port`            | Port for the metrics HTTP server      | `9090`                                                                             | `APP_METRICS_PORT`             |

### Example Configuration

```yaml
proxy_connection_string: ""
super_subtitle_domain: "https://feliratok.eu"
client_timeout: "30s"
user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0"
log_level: "info"
log_format: "console"

server:
  port: 8080
  address: "localhost"

cache:
  type: "memory"  # "memory" (in-process LRU) or "redis" (Redis/Valkey-backed LRU)
  size: 2000
  ttl: "24h"
  redis:
    address: "localhost:6379"
    password: ""
    db: 0

metrics:
  enabled: true
  port: 9090
```

### Environment Variables

```bash
# Override log level
export LOG_LEVEL=debug

# Enable JSON logging
export LOG_FORMAT=json

# Override server address
export APP_SERVER_ADDRESS=0.0.0.0

# Override domain
export APP_SUPER_SUBTITLE_DOMAIN=https://feliratok.eu

# Override timeout
export APP_CLIENT_TIMEOUT=60s

# Override metrics port
export APP_METRICS_PORT=9091

# Disable metrics
export APP_METRICS_ENABLED=false
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

The test job spins up a `valkey/valkey:latest` service container (Redis-compatible, exposed on `localhost:6379`) and sets `REDIS_ADDRESS=localhost:6379` so the Redis cache tests in `internal/cache/redis_test.go` run against a real Valkey instance instead of being skipped.

```bash
# Service: valkey/valkey:latest on localhost:6379
REDIS_ADDRESS=localhost:6379 gotestsum --format testname -- -race -coverprofile=coverage.txt -covermode=atomic ./...
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
| `github.com/hashicorp/golang-lru/v2` | In-memory LRU cache (memory backend)           | v2                 |
| `github.com/redis/go-redis/v9`      | Redis/Valkey client (redis cache backend)       | v9                 |
| `github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus` | gRPC Prometheus interceptors | Latest |
| `github.com/prometheus/client_golang` | Prometheus client library               | Latest             |
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

**Key features:**

- Multi-stage build for minimal image size (download stage separate from runtime)
- SHA256 checksum verification for downloaded `grpc_health_probe` binary (supply chain security)
- Non-root user for security
- Standard gRPC health check support via `grpc_health_probe`
- Health check runs every 30 seconds with 10-second timeout
- Only essential runtime dependencies in final image (ca-certificates)

### Health Checks

The Docker image includes built-in health checking using the standard gRPC health checking protocol:

```dockerfile
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ["/bin/grpc_health_probe", "-addr=:8080"]
```

**Health check parameters:**

- **Interval**: 30 seconds between checks
- **Timeout**: 10 seconds per check
- **Start period**: 5 seconds grace period on container startup
- **Retries**: 3 consecutive failures before marking unhealthy

**Manual health check:**

```bash
docker exec <container-id> /bin/grpc_health_probe -addr=:8080
```

**View health status:**

```bash
docker ps --format "table {{.Names}}\t{{.Status}}"
```

### Running with Docker

```bash
# Pull the latest image
docker pull ghcr.io/belphemur/supersubtitles:latest

# Run with default configuration (expose gRPC and metrics ports)
docker run -p 8080:8080 -p 9090:9090 ghcr.io/belphemur/supersubtitles:latest

# Run with custom configuration
docker run -p 8080:8080 -p 9090:9090 \
  -e APP_SERVER_ADDRESS=0.0.0.0 \
  -e LOG_LEVEL=debug \
  ghcr.io/belphemur/supersubtitles:latest

# Run with volume-mounted config
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config/config.yaml \
  ghcr.io/belphemur/supersubtitles:latest

# Run with Redis/Valkey cache backend
docker run -p 8080:8080 -p 9090:9090 \
  -e APP_SERVER_ADDRESS=0.0.0.0 \
  -e APP_CACHE_TYPE=redis \
  -e APP_CACHE_REDIS_ADDRESS=redis:6379 \
  ghcr.io/belphemur/supersubtitles:latest
```

### Kubernetes Deployment

Example deployment with liveness and readiness probes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: supersubtitles
spec:
  replicas: 3
  selector:
    matchLabels:
      app: supersubtitles
  template:
    metadata:
      labels:
        app: supersubtitles
    spec:
      containers:
        - name: supersubtitles
          image: ghcr.io/belphemur/supersubtitles:latest
          ports:
            - containerPort: 8080
              name: grpc
          env:
            - name: APP_SERVER_ADDRESS
              value: "0.0.0.0"
            - name: LOG_LEVEL
              value: "info"
          livenessProbe:
            exec:
              command: ["/bin/grpc_health_probe", "-addr=:8080"]
            initialDelaySeconds: 5
            periodSeconds: 30
            timeoutSeconds: 10
          readinessProbe:
            exec:
              command: ["/bin/grpc_health_probe", "-addr=:8080"]
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: supersubtitles
spec:
  selector:
    app: supersubtitles
  ports:
    - port: 8080
      targetPort: 8080
      name: grpc
  type: ClusterIP
```

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

## Monitoring

### Logging

The application uses structured logging with zerolog. Control log level and format via config or env vars:

```bash
LOG_LEVEL=debug LOG_FORMAT=json ./super-subtitles
```

Log levels: `debug`, `info`, `warn`, `error`

### Metrics

When `metrics.enabled: true` (the default), an HTTP server exposes Prometheus metrics at `/metrics` on the configured metrics port (default `9090`):

```bash
curl http://localhost:9090/metrics
```

**gRPC server metrics** (via `go-grpc-middleware/providers/prometheus`):

| Metric                           | Type      | Labels                      | Description              |
| -------------------------------- | --------- | --------------------------- | ------------------------ |
| `grpc_server_started_total`      | Counter   | type, service, method       | RPCs started             |
| `grpc_server_handled_total`      | Counter   | type, service, method, code | RPCs completed           |
| `grpc_server_handling_seconds`   | Histogram | type, service, method       | RPC latency              |
| `grpc_server_msg_received_total` | Counter   | type, service, method       | Stream messages received |
| `grpc_server_msg_sent_total`     | Counter   | type, service, method       | Stream messages sent     |

**Application metrics** (custom, registered in `internal/metrics/metrics.go`):

| Metric                           | Type    | Labels                 | Description                |
| -------------------------------- | ------- | ---------------------- | -------------------------- |
| `subtitle_downloads_total`       | Counter | status (success/error) | Subtitle download attempts |
| `subtitle_cache_hits_total`      | Counter | —                      | ZIP cache hits             |
| `subtitle_cache_misses_total`    | Counter | —                      | ZIP cache misses           |
| `subtitle_cache_evictions_total` | Counter | —                      | ZIP cache evictions        |
| `subtitle_cache_entries`         | Gauge   | —                      | Current ZIP cache size     |

Go runtime metrics (goroutines, memory, GC) are included automatically by the default Prometheus registry.

A ready-to-import Grafana dashboard is available at [`grafana/dashboard.json`](../grafana/dashboard.json). Import it via Grafana → Dashboards → Import, then select your Prometheus datasource.
