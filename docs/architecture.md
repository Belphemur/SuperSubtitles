# SuperSubtitles — Architecture

SuperSubtitles is a Go gRPC proxy for [feliratok.eu](https://feliratok.eu) (Hungarian subtitle site). It scrapes show listings and subtitles from HTML pages, normalizes the data (language codes, qualities, timestamps), extracts third-party IDs (IMDB, TVDB, TVMaze, Trakt), and serves it all through a streaming gRPC API. Subtitle files can be downloaded with episode-level extraction from season pack ZIPs, backed by a pluggable LRU cache.

## Domain Structure

```
cmd/proxy/          → Application entry point, gRPC server startup, graceful shutdown
internal/
  grpc/             → gRPC server (4 streaming + 2 unary RPCs), proto ↔ model converters
  client/           → Streaming-first HTTP client for feliratok.eu (parallel fetching, pagination)
  parser/           → HTML parsers using goquery (shows, subtitles, third-party IDs, charset)
  services/         → SubtitleDownloader (ZIP extraction, format detection, caching, metrics)
  models/           → Core domain types (Show, Subtitle, Quality, StreamResult[T], etc.)
  cache/            → Pluggable LRU cache (memory + Redis/Valkey), factory + provider registry
  metrics/          → Prometheus metric definitions and /metrics HTTP server
  config/           → Viper config loader + zerolog singleton logger
  apperrors/        → Custom error types (ErrNotFound, ErrSubtitleNotFoundInZip, etc.)
  testutil/         → HTML fixture generators + stream collection helpers (test-only)
api/proto/v1/       → Proto definitions and generated gRPC code
config/             → Default config.yaml
```

## Component Flow

```
gRPC Clients
     │
     ▼
internal/grpc        ← Server: consumes streaming channels, sends proto messages
     │
     ├──► internal/client     ← Streaming HTTP client (parallel fetch, pagination, partial failure)
     │        │
     │        ├──► internal/parser    ← HTML → normalized models (goquery, charset detection)
     │        └──► internal/services  ← SubtitleDownloader (ZIP, caching, format detection)
     │                  │
     │                  └──► internal/cache  ← Pluggable LRU (memory / Redis)
     │
     ├──► internal/models     ← Shared domain types
     ├──► internal/config     ← Configuration + logging
     └──► internal/metrics    ← Prometheus counters/gauges + HTTP server
```

## Key Design Decisions

| Domain | Decision | Why | Details |
| --- | --- | --- | --- |
| Streaming | Server-side streaming for list RPCs; streaming-first client with channels | Faster time-to-first-result, lower memory | [streaming](./design-decisions/streaming.md) |
| HTTP | failsafe-go retry at transport layer; partial failure resilience | Transparent retries, graceful degradation | [http-client](./design-decisions/http-client.md) |
| Parsing | Generic `Parser[T]` interfaces; normalization in parser; UTF-8 safety | Type safety, single-pass transform, encoding resilience | [parsing](./design-decisions/parsing.md) |
| Cache | Pluggable factory (memory/Redis); metrics with group label | Backend flexibility, transparent instrumentation | [cache](./design-decisions/cache.md) |
| Infrastructure | Standard gRPC health protocol; custom error types with `Is()` | Industry-standard tooling, proper error propagation | [infrastructure](./design-decisions/infrastructure.md) |
| Testing | Programmatic HTML fixtures via `testutil` | One generator to update, not dozens of HTML strings | [testing](./design-decisions/testing.md) |

## Documentation

| Document | Description |
| --- | --- |
| [gRPC API](./grpc-api.md) | RPC endpoints, proto service definition, usage examples |
| [Data Flow](./data-flow.md) | Operation flows for show list, subtitles, downloads, recent |
| [Testing](./testing.md) | Test strategy, coverage goals, fixture generators |
| [Configuration](./configuration.md) | All config fields and environment variables |
| [CI/CD](./ci-cd.md) | What each workflow does |
| [Deployment](./deployment.md) | Docker, Kubernetes, monitoring |
| [Design Decisions](./design-decisions.md) | Index of architectural decision records |
