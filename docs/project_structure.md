# Project Structure

## Directory Layout

```
SuperSubtitles/
├── cmd/proxy/
│   └── main.go                    # Application entry point
├── internal/
│   ├── apperrors/
│   │   └── errors.go              # ErrNotFound, ErrSubtitleNotFoundInZip, ErrSubtitleResourceNotFound
│   ├── cache/
│   │   ├── cache.go               # Cache interface: Get, Set, Contains, Len, Close
│   │   ├── factory.go             # Provider registry: Register, New, RegisteredProviders
│   │   ├── memory.go              # In-memory LRU provider (hashicorp/golang-lru)
│   │   ├── redis.go               # Redis/Valkey provider (hash + sorted set + HPEXPIRE)
│   │   ├── memory_test.go
│   │   ├── factory_test.go
│   │   └── redis_test.go
│   ├── client/
│   │   ├── client.go              # Client interface & constructor
│   │   ├── show_list.go           # Show list fetching from parallel endpoints
│   │   ├── subtitles.go           # Subtitle fetching with pagination
│   │   ├── show_subtitles.go      # Show subtitles with third-party IDs
│   │   ├── updates.go             # Update checking
│   │   ├── download.go            # Subtitle download delegation
│   │   ├── recent_subtitles.go    # Recent subtitles fetching
│   │   ├── compression_transport.go       # GZIP/Brotli/Zstd transport
│   │   ├── show_list_test.go
│   │   ├── subtitles_test.go
│   │   ├── show_subtitles_test.go
│   │   ├── updates_test.go
│   │   ├── download_test.go
│   │   ├── recent_subtitles_test.go
│   │   ├── client_compression_test.go
│   │   ├── compression_transport_test.go
│   │   └── client_integration_test.go     # Live integration tests (skipped in CI)
│   ├── config/
│   │   └── config.go              # Viper configuration & zerolog singleton logger
│   ├── grpc/
│   │   ├── server.go              # SuperSubtitlesServiceServer implementation (6 RPCs)
│   │   ├── setup.go               # NewGRPCServer factory (Prometheus interceptors, health, reflection)
│   │   ├── converters.go          # Proto ↔ model conversion functions
│   │   ├── converters_test.go
│   │   └── server_test.go
│   ├── metrics/
│   │   ├── metrics.go             # Custom Prometheus metric definitions
│   │   ├── server.go              # HTTP server for /metrics endpoint
│   │   └── metrics_test.go
│   ├── models/
│   │   ├── show.go                # Show struct
│   │   ├── subtitle.go            # Subtitle & SubtitleCollection
│   │   ├── show_subtitles.go      # ShowSubtitles composite model
│   │   ├── third_party_ids.go     # IMDB/TVDB/TVMaze/Trakt IDs
│   │   ├── quality.go             # Quality enum (360p–2160p)
│   │   ├── stream_result.go       # StreamResult[T] generic streaming type
│   │   ├── download_request.go    # Download request/response models
│   │   └── update_check.go        # Update check models
│   ├── parser/
│   │   ├── interfaces.go          # Generic Parser[T] and SingleResultParser[T] interfaces
│   │   ├── charset.go             # NewUTF8Reader for automatic charset detection
│   │   ├── show_parser.go         # HTML parser for show listings
│   │   ├── subtitle_parser.go     # HTML parser for subtitles + pagination
│   │   ├── third_party_parser.go  # HTML parser for third-party IDs
│   │   ├── show_parser_test.go
│   │   ├── subtitle_parser_test.go
│   │   └── third_party_parser_test.go
│   ├── services/
│   │   ├── subtitle_downloader.go          # SubtitleDownloader interface
│   │   └── subtitle_downloader_impl.go     # ZIP extraction, caching, format detection, metrics
│   └── testutil/
│       ├── html_fixtures.go       # Programmatic HTML test fixture generators
│       └── stream_helpers.go      # CollectShows / CollectSubtitles / CollectShowSubtitles helpers
├── api/proto/v1/
│   ├── supersubtitles.proto
│   ├── supersubtitles.pb.go
│   ├── supersubtitles_grpc.pb.go
│   └── generate.go
├── config/
│   └── config.yaml                # Default configuration
├── grafana/
│   └── dashboard.json             # Grafana dashboard for gRPC + subtitle metrics
├── build/
│   └── Dockerfile                 # Multi-platform Docker image
├── go.mod / go.sum
├── .golangci.yml
├── .goreleaser.yml
├── .releaserc.yml
├── renovate.json
└── .github/workflows/
    ├── ci.yml                     # Lint, test, build
    ├── release.yml                # Semantic-release + GoReleaser
    └── copilot-setup-steps.yml
```

## Package Descriptions

### `cmd/proxy/`

Application entry point. Creates the HTTP client, starts the gRPC server (via `grpc.NewGRPCServer()`), starts the Prometheus metrics HTTP server (if enabled), and handles graceful shutdown on SIGTERM/SIGINT.

### `internal/apperrors/`

Custom error types with `Is()` method support:

- `ErrNotFound` — resource or show not found
- `ErrSubtitleNotFoundInZip` — requested episode missing from a season pack ZIP
- `ErrSubtitleResourceNotFound` — subtitle download URL returned HTTP 404

All three are mapped to `codes.NotFound` by the gRPC server.

### `internal/cache/`

Pluggable LRU cache abstraction using a factory + provider registry pattern:

- `cache.go` — `Cache` interface: `Get`, `Set`, `Contains`, `Len`, `Close`
- `factory.go` — Provider registry: `Register`, `New`, `RegisteredProviders`
- `memory.go` — In-memory LRU provider (wraps `hashicorp/golang-lru/v2/expirable`)
- `redis.go` — Redis/Valkey provider using a hash + sorted set + `HPEXPIRE` with atomic Lua scripts

### `internal/client/`

HTTP client for feliratok.eu, organized by feature:

- `client.go` — `Client` interface and `NewClient` constructor
- `show_list.go` — Parallel fetch from 3 endpoints with batch pagination
- `subtitles.go` — Subtitle fetching with 2-page parallel pagination
- `show_subtitles.go` — Subtitle fetch + third-party ID extraction, batched in groups of 20
- `updates.go` — Update checking via recheck endpoint
- `download.go` — Delegates to `SubtitleDownloader` service
- `recent_subtitles.go` — Main page scraping with ID-based filtering and per-show grouping
- `compression_transport.go` — HTTP transport with GZIP, Brotli, Zstd support

### `internal/config/`

Viper-based configuration from `config/config.yaml`. Supports `APP_*` env var overrides. Provides a zerolog singleton logger via `config.GetLogger()`.

### `internal/grpc/`

gRPC server layer:

- `setup.go` — `NewGRPCServer()`: Prometheus interceptors, health check, gRPC reflection
- `server.go` — 6 RPC implementations (4 streaming, 2 unary)
- `converters.go` — Proto ↔ internal model conversion functions

### `internal/metrics/`

Prometheus metrics:

- `metrics.go` — Custom counters/gauges registered via `init()`
- `server.go` — `NewHTTPServer()` serving `promhttp.Handler()` at `/metrics`

### `internal/models/`

Core data structures used across all packages:

- `Show`, `Subtitle`, `SubtitleCollection`, `ShowSubtitles`, `ThirdPartyIds`
- `Quality` enum (360p–2160p) with JSON marshaling
- `StreamResult[T]` — generic streaming result carrier with `Value` and `Err` fields
- `DownloadRequest`/`DownloadResult`, `UpdateCheckResult`

### `internal/parser/`

HTML parsers using `goquery`:

- `interfaces.go` — `Parser[T]` (returns `[]T`) and `SingleResultParser[T]` (returns single `T`)
- `charset.go` — `NewUTF8Reader` wraps any `io.Reader` with automatic encoding detection
- `show_parser.go` — Show listings: extracts ID, name, year, image URL; handles multi-column grid layout
- `subtitle_parser.go` — Subtitle tables: language conversion, quality/season/episode parsing, pagination
- `third_party_parser.go` — Detail pages: extracts IMDB, TVDB, TVMaze, Trakt IDs from links

### `internal/services/`

- `subtitle_downloader.go` — `SubtitleDownloader` interface
- `subtitle_downloader_impl.go` — Downloads subtitle files; detects content type; extracts episodes from ZIP season packs using regex; pluggable LRU cache via `cache.Cache`; emits Prometheus metrics

### `internal/testutil/`

Test-only utilities (never imported by production code):

- `html_fixtures.go` — Programmatic HTML generators: `GenerateSubtitleTableHTML`, `GenerateShowTableHTML`, `GenerateThirdPartyIDHTML`, etc.
- `stream_helpers.go` — `CollectShows`, `CollectSubtitles`, `CollectShowSubtitles` for consuming client streams in tests
