# SuperSubtitles — Overview

## What the App Does

SuperSubtitles is a Go proxy service that interfaces with [feliratok.eu](https://feliratok.eu), a Hungarian subtitle repository. It:

1. **Scrapes TV show listings** from multiple HTML endpoints (pending, in-progress, and not-fully-translated shows) in parallel, deduplicating results by show ID.
2. **Fetches subtitle data** for individual shows by scraping HTML pages with automatic pagination support (2 pages fetched in parallel), returning language, qualities, season/episode info, uploader, and download URLs.
3. **Fetches recent subtitles** from the main show page, filtered by subtitle ID, returning full show details with third-party IDs for each unique show.
4. **Extracts third-party IDs** (IMDB, TVDB, TVMaze, Trakt) by scraping show detail pages.
5. **Normalizes all data** — converting Hungarian language names to ISO codes, parsing quality strings (360p–2160p), building download URLs, and converting timestamps.
6. **Checks for updates** since a given content ID via the recheck endpoint.
7. **Downloads subtitles with episode extraction** — downloads subtitle files with support for extracting specific episodes from season pack ZIP files, using an LRU cache (1-hour TTL) to optimize repeated requests.

The application runs a **gRPC server** (`cmd/proxy/main.go`) that exposes all client functionality through a clean gRPC API with **server-side streaming** for list/collection endpoints. The server listens on the configured address and port (`server.address` and `server.port` in config), supports graceful shutdown, and includes gRPC reflection for tools like grpcurl.

## High-Level Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                        cmd/proxy/main.go                      │
│                      (Application Entry Point)                │
│                                                               │
│  • Creates gRPC server via internal/grpc.NewGRPCServer        │
│  • Starts Prometheus metrics HTTP server (port 9090)          │
│  • Handles graceful shutdown on SIGTERM/SIGINT                │
└──────────────────────────┬────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────────┐
│                    internal/grpc                              │
│                                                               │
│  setup.go:                                                    │
│    • NewGRPCServer: creates server with Prometheus            │
│      interceptors, health check, and reflection               │
│                                                               │
│  server.go — Implements SuperSubtitlesServiceServer:          │
│    • GetShowList        (server-side streaming)                │
│    • GetSubtitles       (server-side streaming)                │
│    • GetShowSubtitles   (server-side streaming)                │
│    • CheckForUpdates    (unary)                                │
│    • DownloadSubtitle   (unary)                                │
│    • GetRecentSubtitles (server-side streaming)                │
│                                                               │
│  Consumes channel-based streaming from client,                │
│  converts and sends proto messages as they arrive              │
└────────┬──────────────────────────────────────────────────────┘
         │
         ▼
┌───────────────────────────────────────────────────────────────┐
│                  api/proto/v1/supersubtitles.proto            │
│                                                               │
│  Proto definitions for:                                       │
│    • Service: SuperSubtitlesService (4 streaming, 2 unary)    │
│    • Messages: Show, Subtitle, ShowInfo, ShowSubtitlesCollection│
│    • Enums: Quality (360p-2160p)                              │
│                                                               │
│  Generated via: go generate ./api/proto/v1                    │
└───────────────────────────────────────────────────────────────┘
         │
         ▼
┌───────────────────────────────────────────────────────────────┐
│                     internal/client                           │
│                                                               │
│  Client interface (streaming-first):                          │
│    • CheckForUpdates(ctx, contentID) → *UpdateCheckResult     │
│    • DownloadSubtitle(ctx, url, req) → *DownloadResult        │
│                                                               │
│    • StreamShowList(ctx) → <-chan StreamResult[Show]           │
│    • StreamSubtitles(ctx, showID) → <-chan StreamResult[Sub]  │
│    • StreamShowSubtitles(ctx, shows)                          │
│        → <-chan StreamResult[ShowSubtitles]                   │
│    • StreamRecentSubtitles(ctx, sinceID)                      │
│        → <-chan StreamResult[ShowSubtitles]                   │
│                                                               │
│  Streaming methods accumulate subtitles per show and send     │
│  complete ShowSubtitles items. Handles HTTP requests, proxy   │
│  config, parallel fetching, error aggregation, and            │
│  partial-failure resilience.                                  │
└────────┬──────────────────────────┬───────────────────────────┘
         │                          │
         ▼                          ▼
┌─────────────────────┐  ┌──────────────────────────────┐
│   internal/parser   │  │     internal/services        │
│                     │  │                              │
│  Parser[T]          │  │  SubtitleDownloader          │
│  SingleResultParser │  │   • ZIP download & caching   │
│                     │  │   • Episode extraction       │
│  ShowParser         │  │   • LRU cache (1h TTL)       │
│   (HTML → []Show)   │  │                              │
│                     │  │                              │
│  ThirdPartyIdParser │  │                              │
│   (HTML → IDs)      │  │                              │
│                     │  │                              │
│  SubtitleParser [*] │  │                              │
│   (HTML → []Sub)    │  │                              │
│                     │  │                              │
│  [*] Parses HTML    │  │                              │
│      subtitle       │  │                              │
│      tables with    │  │                              │
│      pagination,    │  │                              │
│      converts lang  │  │                              │
│      to ISO codes,  │  │                              │
│      extracts       │  │                              │
│      qualities &    │  │                              │
│      release info   │  │                              │
└─────────────────────┘  └──────────────────────────────┘
         │                          │
         └──────────┬───────────────┘
                    ▼
┌───────────────────────────────────────────────────────────────┐
│                     internal/models                           │
│                                                               │
│  Show, Subtitle, SubtitleCollection, SuperSubtitleResponse,   │
│  ShowSubtitles, ThirdPartyIds,                               │
│  Quality (enum), UpdateCheckResponse, UpdateCheckResult,      │
│  DownloadRequest, DownloadResult, StreamResult[T]             │
│                                                               │
│  StreamResult[T] is the generic streaming result type used    │
│  by all client streaming methods and gRPC streaming handlers. │
└───────────────────────────────────────────────────────────────┘
                    │
                    ▼
┌───────────────────────────────────────────────────────────────┐
│                     internal/config                           │
│                                                               │
│  Viper-based configuration loaded from config/config.yaml     │
│  Zerolog singleton logger with console output                 │
│  Env var support (prefix: APP_, also LOG_LEVEL)               │
└───────────────────────────────────────────────────────────────┘
                    │
                    ▼
┌───────────────────────────────────────────────────────────────┐
│                     internal/metrics                          │
│                                                               │
│  metrics.go: Custom Prometheus metrics (subtitle downloads,   │
│    cache hits/misses/evictions/entries)                        │
│  server.go: HTTP server for /metrics endpoint                 │
│  Metrics registered via init() with default Prometheus        │
│  registry. Go runtime metrics included automatically.         │
└───────────────────────────────────────────────────────────────┘
```

## Related Documentation

- [Data Flow](./data-flow.md) - Detailed data flow for all operations
- [Testing](./testing.md) - Testing infrastructure and strategy
- [Design Decisions](./design-decisions.md) - Key architectural decisions
- [Deployment](./deployment.md) - Configuration, CI/CD, and dependencies
