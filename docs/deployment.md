# SuperSubtitles — Deployment

> **See also:** [Configuration](./configuration.md) for all config fields and environment variables, and [CI/CD](./ci-cd.md) for the build pipeline and dependencies.

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

**Application metrics** (custom):

| Metric                     | Type    | Labels                 | Source                    | Description                |
| -------------------------- | ------- | ---------------------- | ------------------------- | -------------------------- |
| `subtitle_downloads_total` | Counter | status (success/error) | `internal/metrics`        | Subtitle download attempts |
| `cache_hits_total`         | Counter | cache                  | `internal/cache`          | Cache hits per group       |
| `cache_misses_total`       | Counter | cache                  | `internal/cache`          | Cache misses per group     |
| `cache_evictions_total`    | Counter | cache                  | `internal/cache`          | Evictions per group        |
| `cache_entries`            | Gauge   | cache                  | `internal/cache`          | Current entries per group (lazily evaluated at scrape time) |

The `cache` label value is the **group** set via `ProviderConfig.Group` when the cache is created (e.g., `"zip"` for the subtitle ZIP cache). Using a label instead of a metric-name prefix allows the same cache infrastructure to be reused for other purposes without renaming metrics.

Go runtime metrics (goroutines, memory, GC) are included automatically by the default Prometheus registry.

A ready-to-import Grafana dashboard is available at [`grafana/dashboard.json`](../grafana/dashboard.json). Import it via Grafana → Dashboards → Import, then select your Prometheus datasource.
