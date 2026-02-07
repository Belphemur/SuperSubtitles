# SuperSubtitles — Architecture

## What the App Does

SuperSubtitles is a Go proxy service that interfaces with [feliratok.eu](https://feliratok.eu), a Hungarian subtitle repository. It:

1. **Scrapes TV show listings** from multiple HTML endpoints (pending, in-progress, and not-fully-translated shows) in parallel, deduplicating results by show ID.
2. **Fetches subtitle data** for individual shows via a JSON API (`?action=xbmc&sid=<id>`), returning language, quality, season/episode info, uploader, and download URLs.
3. **Extracts third-party IDs** (IMDB, TVDB, TVMaze, Trakt) by scraping show detail pages.
4. **Normalizes all data** — converting Hungarian language names to ISO codes, parsing quality strings (360p–2160p), building download URLs, and converting timestamps.
5. **Checks for updates** since a given content ID via the recheck endpoint.

The application is currently a CLI tool (`cmd/proxy/main.go`) that demonstrates fetching and logging show data. It is designed to be extended into a full proxy server (the config already supports `server.port` and `server.address`).

## High-Level Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                        cmd/proxy/main.go                      │
│                      (Application Entry Point)                │
└──────────────────────────┬────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────────┐
│                     internal/client                           │
│                                                               │
│  Client interface:                                            │
│    • GetShowList(ctx) → []Show                                │
│    • GetSubtitles(ctx, showID) → *SubtitleCollection          │
│    • GetShowSubtitles(ctx, shows) → []ShowSubtitles           │
│    • CheckForUpdates(ctx, contentID) → *UpdateCheckResult     │
│                                                               │
│  Handles HTTP requests, proxy config, parallel fetching,      │
│  error aggregation, and partial-failure resilience.            │
└────────┬──────────────────────────┬───────────────────────────┘
         │                          │
         ▼                          ▼
┌─────────────────────┐  ┌──────────────────────────────┐
│   internal/parser   │  │     internal/services        │
│                     │  │                              │
│  Parser[T]          │  │  SubtitleConverter           │
│  SingleResultParser │  │   • Language → ISO code      │
│                     │  │   • Quality extraction       │
│  ShowParser         │  │   • Season/episode parsing   │
│   (HTML → []Show)   │  │   • Download URL building    │
│                     │  │   • Timestamp conversion     │
│  ThirdPartyIdParser │  │                              │
│   (HTML → IDs)      │  │                              │
└─────────────────────┘  └──────────────────────────────┘
         │                          │
         └──────────┬───────────────┘
                    ▼
┌───────────────────────────────────────────────────────────────┐
│                     internal/models                           │
│                                                               │
│  Show, Subtitle, SubtitleCollection, SuperSubtitleResponse,   │
│  ShowSubtitles, ThirdPartyIds, Quality (enum),                │
│  UpdateCheckResponse, UpdateCheckResult                       │
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

## Data Flow

### Show List Fetching
1. `GetShowList` fires 3 parallel HTTP requests to different feliratok.eu endpoints
2. Each response is parsed by `ShowParser.ParseHtml` using goquery to extract show ID, name, year, and image URL from HTML tables
3. Results are merged and deduplicated by show ID, preserving first-occurrence order
4. Partial failures are tolerated — if at least one endpoint succeeds, results are returned

### Subtitle Fetching
1. `GetSubtitles` calls the JSON API endpoint (`?action=xbmc&sid=<id>`)
2. Response is a map of `SuperSubtitle` objects (Hungarian field names)
3. `SubtitleConverter.ConvertResponse` normalizes each entry:
   - Language names (Hungarian → ISO 639-1)
   - Quality detection from subtitle name string
   - Season/episode number parsing (with -1 for season packs)
   - Download URL construction
   - Upload timestamp conversion

### Third-Party ID Extraction
1. `GetShowSubtitles` processes shows in batches of 20
2. For each show, it fetches subtitles, then loads the detail page HTML
3. `ThirdPartyIdParser` extracts IDs from `div.adatlapRow a` links using regex and URL parsing

## Key Design Decisions

- **Partial failure resilience**: The client returns whatever data it successfully fetched, logging warnings for failed endpoints rather than failing the entire operation.
- **Generic parser interfaces**: `Parser[T]` and `SingleResultParser[T]` allow type-safe HTML parsing for different model types.
- **Batch processing**: Show subtitle fetching is batched (20 concurrent) to avoid overwhelming the upstream server.
- **No external test frameworks**: All tests use the Go standard library `testing` package with `httptest` for HTTP mocking.

## Configuration

Loaded from `config/config.yaml`:

| Field | Description | Default |
|-------|-------------|---------|
| `proxy_connection_string` | HTTP proxy URL (optional) | `""` |
| `super_subtitle_domain` | Base URL for feliratok.eu | `https://feliratok.eu` |
| `client_timeout` | HTTP client timeout (Go duration) | `30s` |
| `server.port` | Server listening port | `8080` |
| `server.address` | Server listening address | `localhost` |
| `log_level` | Zerolog level (debug/info/warn/error) | `info` |

Environment variables are also supported with `APP_` prefix (e.g., `APP_CLIENT_TIMEOUT`). `LOG_LEVEL` is bound directly.

## CI/CD Pipeline

### CI (`.github/workflows/ci.yml`)
Runs on every push and PR to `main`:
- **Lint job:** `go mod verify` → `go vet` → `gofmt` → `golangci-lint`
- **Test job:** `gotestsum` with race detector + coverage → Codecov upload
- **Build job:** `CGO_ENABLED=0 go build` → artifact upload

### Release (`.github/workflows/release.yml`)
Runs on push to `main`:
1. `semantic-release` analyzes conventional commits to determine the next version
2. `GoReleaser` builds cross-platform binaries (linux/amd64, linux/arm64)
3. Builds and pushes multi-platform Docker images to `ghcr.io/belphemur/supersubtitles`
4. Publishes a GitHub release with changelog, SBOMs, and build attestation

### Copilot Setup (`.github/workflows/copilot-setup-steps.yml`)
Prepares the Copilot agent environment: Go, gopls, golangci-lint, dependencies.

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/PuerkitoBio/goquery` | jQuery-like HTML parsing |
| `github.com/rs/zerolog` | Structured JSON/console logging |
| `github.com/spf13/viper` | Configuration management |
