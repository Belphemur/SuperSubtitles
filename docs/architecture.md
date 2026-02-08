# SuperSubtitles — Architecture

## What the App Does

SuperSubtitles is a Go proxy service that interfaces with [feliratok.eu](https://feliratok.eu), a Hungarian subtitle repository. It:

1. **Scrapes TV show listings** from multiple HTML endpoints (pending, in-progress, and not-fully-translated shows) in parallel, deduplicating results by show ID.
2. **Fetches subtitle data** for individual shows via a JSON API (`?action=xbmc&sid=<id>`), returning language, quality, season/episode info, uploader, and download URLs.
3. **Extracts third-party IDs** (IMDB, TVDB, TVMaze, Trakt) by scraping show detail pages.
4. **Normalizes all data** — converting Hungarian language names to ISO codes, parsing quality strings (360p–2160p), building download URLs, and converting timestamps.
5. **Checks for updates** since a given content ID via the recheck endpoint.
6. **Downloads subtitles with episode extraction** — downloads subtitle files with support for extracting specific episodes from season pack ZIP files, using an LRU cache (1-hour TTL) to optimize repeated requests.

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
│    • DownloadSubtitle(ctx, url, req) → *DownloadResult        │
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
│   (HTML → IDs)      │  │  SubtitleDownloader          │
│                     │  │   • ZIP download & caching   │
│                     │  │   • Episode extraction       │
│                     │  │   • LRU cache (1h TTL)       │
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

### Subtitle Download with Episode Extraction
1. `DownloadSubtitle` method in the Client interface accepts a download URL and a `DownloadRequest`
2. **`SubtitleDownloader` service is the primary download handler for all subtitle files**:
   - **Regular subtitle files** (SRT, ASS, VTT, SUB): Downloaded and returned with correct content-type and extension
   - **ZIP files without episode number**: Entire ZIP returned (for manual extraction)
   - **ZIP files with episode number**: 
     - ZIP file is downloaded (or retrieved from cache)
     - Episode pattern matching using regex with word boundaries: `S03E01`, `s03e01`, `3x01`, `E01` (with guards against false positives like E01 matching E010)
     - Specific episode subtitle extracted from the ZIP archive
     - Only the requested episode file is returned with correct content-type based on file extension
3. **Multi-Format Support**:
   - SRT (SubRip) - `application/x-subrip`
   - ASS (Advanced SubStation Alpha) - `application/x-ass`, `text/ass`
   - VTT (WebVTT) - `text/vtt`, `text/webvtt`
   - SUB (MicroDVD) - `application/x-sub`
   - ZIP archives - `application/zip`
   - Unknown formats default to `application/octet-stream`
4. **Caching Strategy**:
   - LRU cache with 100-entry capacity and 1-hour TTL
   - Only ZIP files are cached (regular subtitle files are small and not cached)
   - Cache key is the download URL
   - Multiple episode requests from same season pack use cached ZIP
5. **File Structure Support**:
   - Flat ZIP structure (all files in root)
   - Nested folders (e.g., `ShowName.S03/ShowName.S03E01.srt`)
   - Various naming patterns (uppercase, lowercase, different separators)

**Implementation Files:**
- `internal/services/subtitle_downloader.go` - Interface definition
- `internal/services/subtitle_downloader_impl.go` - Implementation with caching, ZIP extraction, and format detection
- `internal/services/subtitle_downloader_test.go` - Comprehensive tests (15 test cases including edge cases)
- `internal/models/download_request.go` - Request/response models
- `internal/client/client.go` - Client integration

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
| `github.com/hashicorp/golang-lru/v2` | LRU cache for ZIP file caching (1h TTL) |
| `archive/zip` (stdlib) | ZIP file extraction for season packs |
