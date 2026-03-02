# SuperSubtitles — gRPC API

## Overview

SuperSubtitles exposes all client functionality through a gRPC API. The API is defined in Protocol Buffers and provides type-safe, efficient communication for all subtitle operations.

## Proto Definition

The API is defined in [`api/proto/v1/supersubtitles.proto`](../api/proto/v1/supersubtitles.proto).

### Service: SuperSubtitlesService

```protobuf
service SuperSubtitlesService {
  rpc GetShowList(GetShowListRequest) returns (stream Show);
  rpc GetSubtitles(GetSubtitlesRequest) returns (stream Subtitle);
  rpc GetShowSubtitles(GetShowSubtitlesRequest) returns (stream ShowSubtitlesCollection);
  rpc CheckForUpdates(CheckForUpdatesRequest) returns (CheckForUpdatesResponse);
  rpc DownloadSubtitle(DownloadSubtitleRequest) returns (DownloadSubtitleResponse);
  rpc GetRecentSubtitles(GetRecentSubtitlesRequest) returns (stream ShowSubtitlesCollection);
}
```

Four of six RPCs use **server-side streaming**, sending items as they become available rather than buffering entire responses. This improves time-to-first-result and reduces memory usage. `CheckForUpdates` and `DownloadSubtitle` remain unary RPCs.

### Health Check Service

The server also implements the standard gRPC health checking protocol (`grpc.health.v1.Health`). This enables:

- Docker container health checks using `grpc_health_probe`
- Kubernetes liveness/readiness probes
- Load balancer health monitoring
- Service mesh integration

**Check overall server health:**

```bash
grpc_health_probe -addr=localhost:8080
```

**Check specific service health:**

```bash
grpc_health_probe -addr=localhost:8080 -service=supersubtitles.v1.SuperSubtitlesService
```

The health service reports `SERVING` for both the overall server (`""` service) and the `supersubtitles.v1.SuperSubtitlesService` specifically.

## Code Generation

Proto files are compiled to Go code using `go generate`:

```bash
# Generate proto code
cd api/proto/v1
go generate

# Or from project root
go generate ./api/proto/v1
```

**Required tools:**

- `protoc` (Protocol Buffer compiler)
- `protoc-gen-go` (Go plugin for protoc)
- `protoc-gen-go-grpc` (gRPC plugin for protoc)

These are automatically installed by the generate script if not present.

## Server Implementation

The gRPC server is implemented in [`internal/grpc/server.go`](../internal/grpc/server.go):

- Implements `SuperSubtitlesServiceServer` interface
- Wraps the HTTP client (`internal/client.Client`)
- Converts between proto messages and internal models
- Consumes from channel-based streaming client methods, sending items as they arrive
- Provides structured logging via zerolog
- Returns gRPC status codes for errors

### Starting the Server

The server is created via `NewGRPCServer()` in [`internal/grpc/setup.go`](../internal/grpc/setup.go), which configures Prometheus interceptors, health checking, and reflection. The entry point in [`cmd/proxy/main.go`](../cmd/proxy/main.go) orchestrates startup:

```go
// Create gRPC server with Prometheus interceptors, health check, and reflection
grpcServer := grpcserver.NewGRPCServer(httpClient)

// Optionally start Prometheus metrics HTTP server
if cfg.Metrics.Enabled {
    metricsServer := metrics.NewHTTPServer(cfg.Server.Address, cfg.Metrics.Port)
    go metricsServer.ListenAndServe()
}

// Listen and serve
listener, _ := net.Listen("tcp", address)
grpcServer.Serve(listener)
```

**Features:**

- Prometheus gRPC interceptors for request counts, latencies, and stream message counts
- Graceful shutdown on SIGTERM/SIGINT
- gRPC reflection enabled (for grpcurl, Postman, etc.)
- Standard gRPC health checking protocol
- Configurable address and port via config

## API Endpoints

### 1. GetShowList (server-side streaming)

Streams all available TV shows.

**Request:** Empty
**Response:** Stream of `Show` messages (name, ID, year, image URL)

**Example with grpcurl:**

```bash
grpcurl -plaintext localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetShowList
```

### 2. GetSubtitles (server-side streaming)

Streams all subtitles for a specific show (with automatic pagination).

**Request:**

- `show_id` (int64): Show ID

**Response:** Stream of `Subtitle` messages

**Example:**

```bash
grpcurl -plaintext -d '{"show_id": 1234}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetSubtitles
```

### 3. GetShowSubtitles (server-side streaming)

Streams complete show subtitle collections for multiple shows. Each streamed message contains a show's full information and all its subtitles.

**Request:**

- `shows` (repeated Show): List of shows to fetch

**Response:** Stream of `ShowSubtitlesCollection` messages. Each message contains `show_info` (with show metadata and third-party IDs) and a `subtitles` list with all subtitles for that show.

**Example:**

```bash
grpcurl -plaintext -d '{"shows": [{"id": 1234, "name": "Breaking Bad"}]}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetShowSubtitles
```

### 4. CheckForUpdates

Checks if new subtitles are available since a given content ID.

**Request:**

- `content_id` (string): Content ID to check from

**Response:**

- `film_count` (int32): Number of new films
- `series_count` (int32): Number of new series episodes
- `has_updates` (bool): True if any updates available

**Example:**

```bash
grpcurl -plaintext -d '{"content_id": "12345"}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/CheckForUpdates
```

### 5. DownloadSubtitle

Downloads a subtitle file, optionally extracting a specific episode from a season pack.

**Request:**

- `subtitle_id` (string): Subtitle identifier
- `episode` (int32): Episode number to extract (0 = download entire file)

**Response:**

- `filename` (string): Subtitle filename
- `content` (bytes): File content
- `content_type` (string): MIME type

**Example:**

```bash
grpcurl -plaintext -d '{"subtitle_id": "101", "episode": 1}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/DownloadSubtitle
```

### 6. GetRecentSubtitles (server-side streaming)

Streams recently uploaded subtitles since a given subtitle ID.

**Request:**

- `since_id` (int32): Subtitle ID to fetch from

### 7. Health Check

Checks the health status of the server using the standard gRPC health checking protocol.

**Request:**

- `service` (string): Service name to check (empty string for overall server health)

**Response:**

- `status` (enum): `SERVING`, `NOT_SERVING`, `UNKNOWN`, or `SERVICE_UNKNOWN`

**Example:**

```bash
# Check overall server health
grpc_health_probe -addr=localhost:8080

# Check specific service health
grpc_health_probe -addr=localhost:8080 -service=supersubtitles.v1.SuperSubtitlesService

# Using grpcurl
grpcurl -plaintext localhost:8080 grpc.health.v1.Health/Check

# Check specific service with grpcurl
grpcurl -plaintext -d '{"service": "supersubtitles.v1.SuperSubtitlesService"}' \
  localhost:8080 grpc.health.v1.Health/Check
```

**Response:** Stream of `ShowSubtitlesCollection` messages. Each message contains a show's complete information (with third-party IDs) and all its recent subtitles.

**Example:**

```bash
grpcurl -plaintext -d '{"since_id": 1000}' \
  localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetRecentSubtitles
```

## Data Models

### Show

```protobuf
message Show {
  string name = 1;
  int32 id = 2;
  int32 year = 3;
  string image_url = 4;
}
```

### Subtitle

```protobuf
message Subtitle {
  int32 id = 1;
  int32 show_id = 2;
  string show_name = 3;
  string name = 4;
  string language = 5;
  int32 season = 6;
  int32 episode = 7;
  string filename = 8;
  string download_url = 9;
  string uploader = 10;
  google.protobuf.Timestamp uploaded_at = 11;
  repeated Quality qualities = 12;
  repeated string release_groups = 13;
  string release = 14;
  bool is_season_pack = 15;
}
```

### Quality Enum

```protobuf
enum Quality {
  QUALITY_UNSPECIFIED = 0;
  QUALITY_360P = 1;
  QUALITY_480P = 2;
  QUALITY_720P = 3;
  QUALITY_1080P = 4;
  QUALITY_2160P = 5;
}
```

### ThirdPartyIds

```protobuf
message ThirdPartyIds {
  string imdb_id = 1;
  int64 tvdb_id = 2;
  int64 tv_maze_id = 3;
  int64 trakt_id = 4;
}
```

### ShowInfo

```protobuf
message ShowInfo {
  Show show = 1;
  ThirdPartyIds third_party_ids = 2;
}
```

### ShowSubtitlesCollection

```protobuf
message ShowSubtitlesCollection {
  ShowInfo show_info = 1;
  repeated Subtitle subtitles = 2;
}
```

`ShowSubtitlesCollection` is used by `GetShowSubtitles` and `GetRecentSubtitles` to stream complete show data. Each message contains a show's metadata (with third-party IDs) and all its subtitles in a single message, simplifying client consumption.

## Error Handling

The gRPC server returns standard gRPC status codes:

- `OK` (0): Success
- `NOT_FOUND` (5): Resource not found — episode missing from subtitle ZIP archive (`DownloadSubtitle`), subtitle URL returns HTTP 404 (`DownloadSubtitle`), or show ID does not exist (`GetSubtitles`)
- `INVALID_ARGUMENT` (3): Invalid request — no valid shows provided (`GetShowSubtitles`)
- `INTERNAL` (13): Internal errors (HTTP failures, parsing errors, etc.)

All errors are logged with structured logging using zerolog.

## Testing

Comprehensive unit tests are in [`internal/grpc/server_test.go`](../internal/grpc/server_test.go):

- Mock client implementation for isolated testing
- Tests for all 6 RPC methods
- Error handling tests
- Model conversion tests (including Quality enum)
- No external dependencies (standard Go `testing` package)

Run tests:

```bash
go test ./internal/grpc/...
go test -race ./internal/grpc/...  # with race detector
```

## Configuration

Server configuration is in [`config/config.yaml`](../config/config.yaml):

```yaml
server:
  port: 8080
  address: "0.0.0.0"
```

Override via environment variables:

```bash
APP_SERVER_PORT=9090 APP_SERVER_ADDRESS=127.0.0.1 ./proxy
```

## Deployment

### Local Development

```bash
# Build
go build -o proxy ./cmd/proxy

# Run
./proxy
```

Server runs on configured address:port with gRPC reflection enabled.

### Docker

The Dockerfile in [`build/Dockerfile`](../build/Dockerfile) builds a multi-platform image:

```bash
docker build -f build/Dockerfile -t supersubtitles .
docker run -p 8080:8080 supersubtitles
```

### Testing with grpcurl

List services:

```bash
grpcurl -plaintext localhost:8080 list
```

Describe service:

```bash
grpcurl -plaintext localhost:8080 describe supersubtitles.v1.SuperSubtitlesService
```

Call method:

```bash
grpcurl -plaintext localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetShowList
```

## Design Decisions

### No TLS/SSL in Server

The gRPC server does **not** implement TLS/SSL. This is intentional:

- TLS termination should be handled by a reverse proxy (nginx, Envoy, cloud load balancer)
- Keeps the service focused on business logic
- Simplifies deployment in containerized environments
- Follows the single-responsibility principle

### Model Conversion Layer

The gRPC server includes explicit conversion functions between proto messages and internal models:

- `convertShowToProto` / `convertShowFromProto`
- `convertQualityToProto`
- `convertSubtitleToProto`
- `convertShowSubtitlesToProto`
- `convertThirdPartyIdsToProto`

**Rationale:**

- Decouples proto definitions from internal models
- Allows independent evolution of API and internal structures
- Makes conversions explicit and testable
- Centralizes conversion logic

### gRPC Reflection

The server enables gRPC reflection, allowing tools like grpcurl, Postman, and BloomRPC to introspect the API without access to proto files.

### Graceful Shutdown

The server handles SIGTERM and SIGINT signals, calling `GracefulStop()` to:

- Complete in-flight requests
- Refuse new requests
- Clean up resources properly

## Related Documentation

- [Overview](./overview.md) - High-level architecture
- [Data Flow](./data-flow.md) - Operation flows
- [Testing](./testing.md) - Testing infrastructure
- [Design Decisions](./design-decisions.md) - Architectural decisions
- [Deployment](./deployment.md) - CI/CD and deployment

## Prometheus Metrics

The gRPC server is instrumented with Prometheus metrics via `go-grpc-middleware/providers/prometheus` interceptors. Additionally, custom application metrics track subtitle downloads and cache performance.

### Metrics Endpoint

When enabled (`metrics.enabled: true` in config, default), an HTTP server exposes Prometheus metrics at `/metrics` on a separate port (default `9090`):

```bash
curl http://localhost:9090/metrics
```

Disable metrics or change the port via configuration:

```yaml
metrics:
  enabled: false # disable metrics endpoint
  port: 9091 # or change the port
```

### Available Metrics

**gRPC server metrics** (via interceptors):

| Metric                           | Type      | Labels                      | Description              |
| -------------------------------- | --------- | --------------------------- | ------------------------ |
| `grpc_server_started_total`      | Counter   | type, service, method       | RPCs started             |
| `grpc_server_handled_total`      | Counter   | type, service, method, code | RPCs completed           |
| `grpc_server_handling_seconds`   | Histogram | type, service, method       | RPC latency              |
| `grpc_server_msg_received_total` | Counter   | type, service, method       | Stream messages received |
| `grpc_server_msg_sent_total`     | Counter   | type, service, method       | Stream messages sent     |

**Application metrics** (custom):

| Metric                           | Type    | Labels                 | Description                |
| -------------------------------- | ------- | ---------------------- | -------------------------- |
| `subtitle_downloads_total`       | Counter | status (success/error) | Subtitle download attempts |
| `subtitle_cache_hits_total`      | Counter | —                      | ZIP cache hits             |
| `subtitle_cache_misses_total`    | Counter | —                      | ZIP cache misses           |
| `subtitle_cache_evictions_total` | Counter | —                      | ZIP cache evictions        |
| `subtitle_cache_entries`         | Gauge   | —                      | Current ZIP cache size     |

Go runtime metrics (goroutines, memory, GC) are included automatically by the default Prometheus registry.
