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

The application runs a **gRPC server** (`cmd/proxy/main.go`) that exposes all client functionality through a clean gRPC API. The server listens on the configured address and port (`server.address` and `server.port` in config), supports graceful shutdown, and includes gRPC reflection for tools like grpcurl.

## High-Level Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                        cmd/proxy/main.go                      │
│                      (gRPC Server Entry Point)                │
│                                                               │
│  • Initializes gRPC server with reflection                    │
│  • Registers SuperSubtitlesService                            │
│  • Handles graceful shutdown on SIGTERM/SIGINT                │
└──────────────────────────┬────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────────┐
│                    internal/grpc/server                       │
│                                                               │
│  Implements SuperSubtitlesServiceServer:                      │
│    • GetShowList                                              │
│    • GetSubtitles                                             │
│    • GetShowSubtitles                                         │
│    • CheckForUpdates                                          │
│    • DownloadSubtitle                                         │
│    • GetRecentSubtitles                                       │
│                                                               │
│  Converts between proto messages and internal models          │
└────────┬──────────────────────────────────────────────────────┘
         │
         ▼
┌───────────────────────────────────────────────────────────────┐
│                  api/proto/v1/supersubtitles.proto            │
│                                                               │
│  Proto definitions for:                                       │
│    • Service: SuperSubtitlesService                           │
│    • Messages: Show, Subtitle, SubtitleCollection, etc.       │
│    • Enums: Quality (360p-2160p)                              │
│                                                               │
│  Generated via: go generate ./api/proto/v1                    │
└───────────────────────────────────────────────────────────────┘
         │
         ▼
┌───────────────────────────────────────────────────────────────┐
│                     internal/client                           │
│                                                               │
│  Client interface:                                            │
│    • GetShowList(ctx) → []Show                                │
│    • GetSubtitles(ctx, showID) → *SubtitleCollection [*]      │
│    • GetShowSubtitles(ctx, shows) → []ShowSubtitles           │
│    • GetRecentSubtitles(ctx, sinceID) → []ShowSubtitles [**]  │
│    • CheckForUpdates(ctx, contentID) → *UpdateCheckResult     │
│    • DownloadSubtitle(ctx, url, req) → *DownloadResult        │
│                                                               │
│  Handles HTTP requests, proxy config, parallel fetching,      │
│  error aggregation, and partial-failure resilience.            │
│  [*] HTML-based with parallel pagination (2 pages at a time)  │
│  [**] Fetches from main page, filters by ID, groups by show   │
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
│  ShowSubtitles, ThirdPartyIds, Quality (enum),                │
│  UpdateCheckResponse, UpdateCheckResult,                      │
│  DownloadRequest, DownloadResult                              │
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
```

## Related Documentation

- [Data Flow](./data-flow.md) - Detailed data flow for all operations
- [Testing](./testing.md) - Testing infrastructure and strategy
- [Design Decisions](./design-decisions.md) - Key architectural decisions
- [Deployment](./deployment.md) - Configuration, CI/CD, and dependencies
