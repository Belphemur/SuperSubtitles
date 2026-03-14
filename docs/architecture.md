# SuperSubtitles — Architecture

SuperSubtitles is a Go gRPC proxy for [feliratok.eu](https://feliratok.eu) (Hungarian subtitle site). It scrapes show listings and subtitles from HTML pages, normalizes the data (language codes, qualities, timestamps), extracts third-party IDs (IMDB, TVDB, TVMaze, Trakt), and serves it all through a streaming gRPC API. Subtitle files can be downloaded with episode-level extraction from season pack ZIPs, backed by a pluggable LRU cache.

## Domain Structure

```
cmd/proxy/          → Application entry point
internal/
  grpc/             → gRPC API layer
  client/           → HTTP scraping client for feliratok.eu
  parser/           → HTML parsing and data normalization
  services/         → Subtitle download and file processing
  models/           → Shared domain types
  cache/            → Pluggable caching abstraction
  metrics/          → Prometheus instrumentation
  config/           → Configuration and logging
  sentryio/         → Sentry integration (error reporting, log breadcrumbs)
  apperrors/        → Application error types
  testutil/         → Test utilities (fixtures, helpers)
api/proto/v1/       → Proto definitions and generated code
config/             → Default configuration file
```

## Component Flow

```
gRPC Clients
     │
     ▼
grpc/            ← API layer: streams results to clients
     │
     ├──► client/        ← Scrapes feliratok.eu (parallel fetch, pagination, partial failure)
     │       │
     │       ├──► parser/    ← Transforms HTML into normalized domain models
     │       └──► services/  ← Downloads and processes subtitle files
     │                │
     │                └──► cache/  ← Caches expensive resources (ZIP files)
     │
     ├──► models/    ← Shared domain types
     ├──► config/    ← Configuration + logging
     └──► metrics/   ← Observability
```

## Key Design Decisions

| Domain | Decision | Why | Details |
| --- | --- | --- | --- |
| Streaming | Server-side streaming for list RPCs; channel-based client | Faster time-to-first-result, lower memory | [streaming](./design-decisions/streaming.md) |
| HTTP | Retry at transport layer; partial failure resilience | Transparent retries, graceful degradation | [http-client](./design-decisions/http-client.md) |
| Parsing | Generic parser interfaces; normalization in parser; UTF-8 safety | Type safety, single-pass transform, encoding resilience | [parsing](./design-decisions/parsing.md) |
| Cache | Pluggable factory (memory/Redis); metrics with group label | Backend flexibility, transparent instrumentation | [cache](./design-decisions/cache.md) |
| Infrastructure | Standard gRPC health protocol; custom error types | Industry-standard tooling, proper error propagation | [infrastructure](./design-decisions/infrastructure.md) |
| Logging | Zerolog writer forwards breadcrumbs and structured logs to Sentry | Passive context without per-site Sentry calls | [logging](./design-decisions/logging.md) |
| Testing | Programmatic HTML fixtures | One generator to update, not dozens of HTML strings | [testing](./design-decisions/testing.md) |

## Documentation

| Document | Description |
| --- | --- |
| [gRPC API](./grpc-api.md) | RPC endpoints and usage examples |
| [Data Flow](./data-flow.md) | How each operation works |
| [Testing](./testing.md) | Test strategy, coverage goals, fixture generators |
| [Configuration](./configuration.md) | All config fields and environment variables |
| [CI/CD](./ci-cd.md) | What each workflow does |
| [Deployment](./deployment.md) | Docker, Kubernetes, monitoring |
| [Design Decisions](./design-decisions.md) | Index of architectural decision records |
