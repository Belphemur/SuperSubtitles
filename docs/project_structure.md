# Project Structure

## Directory Layout

```
SuperSubtitles/
├── cmd/proxy/
│   └── main.go                    # CLI entry point
├── internal/
│   ├── client/
│   │   ├── client.go              # HTTP client interface & constructor
│   │   ├── show_list.go           # Show list fetching implementation
│   │   ├── subtitles.go           # Subtitle fetching with pagination
│   │   ├── show_subtitles.go      # Show subtitles with third-party IDs
│   │   ├── updates.go             # Update checking implementation
│   │   ├── download.go            # Subtitle download delegation
│   │   ├── recent_subtitles.go    # Recent subtitles fetching
│   │   ├── show_list_test.go      # Show list tests
│   │   ├── subtitles_test.go      # Subtitle fetching tests
│   │   ├── show_subtitles_test.go # Show subtitles tests
│   │   ├── updates_test.go        # Update check tests
│   │   ├── download_test.go       # Download tests
│   │   ├── recent_subtitles_test.go # Recent subtitles tests
│   │   ├── client_integration_test.go  # Integration tests (skipped in CI)
│   │   ├── client_compression_test.go  # Compression support tests
│   │   ├── compression_transport.go    # GZIP/Brotli/Zstd support
│   │   └── errors.go              # Custom error types
│   ├── config/
│   │   └── config.go              # Viper configuration & zerolog logger
│   ├── models/
│   │   ├── show.go                # Show struct
│   │   ├── subtitle.go            # Subtitle & SubtitleCollection
│   │   ├── show_subtitles.go      # ShowSubtitles composite
│   │   ├── third_party_ids.go     # IMDB/TVDB/TVMaze/Trakt IDs
│   │   ├── quality.go             # Quality enum (360p–2160p)
│   │   ├── download_request.go    # Download request/response models
│   │   └── update_check.go        # Update check models
│   ├── parser/
│   │   ├── interfaces.go          # Generic Parser[T] interfaces
│   │   ├── show_parser.go         # HTML parser for shows
│   │   ├── show_parser_test.go    # Show parser tests
│   │   ├── subtitle_parser.go     # HTML parser for subtitles + pagination
│   │   ├── subtitle_parser_test.go # Subtitle parser tests
│   │   ├── third_party_parser.go  # HTML parser for third-party IDs
│   │   └── third_party_parser_test.go
│   └── services/
│       ├── subtitle_converter.go              # Conversion interface
│       ├── subtitle_converter_impl.go         # Language/quality/URL normalization
│       ├── subtitle_converter_test.go         # Converter tests
│       ├── subtitle_downloader.go             # Download interface
│       ├── subtitle_downloader_impl.go        # Download with ZIP extraction & caching
│       └── subtitle_downloader_test.go        # Download tests with benchmarks
├── config/
│   └── config.yaml                # Default configuration
├── docs/
│   ├── architecture.md            # Detailed architecture & design decisions
│   └── project_structure.md       # This file
├── build/
│   └── Dockerfile                 # Multi-platform Docker build support
├── go.mod / go.sum                # Go dependencies
├── .golangci.yml                  # Linter configuration
├── .goreleaser.yml                # Release binary configuration
├── .releaserc.yml                 # Semantic versioning configuration
├── renovate.json                  # Dependency update automation
└── .github/
    ├── copilot-instructions.md    # Copilot agent guidelines
    ├── dependabot.yml             # GitHub dependency updates
    └── workflows/
        ├── ci.yml                 # CI: lint, test, build
        ├── release.yml            # Release: semantic-release + GoReleaser
        └── copilot-setup-steps.yml # Copilot environment setup
```

## Module Organization

### `cmd/proxy/`

Application entry point. Currently a CLI tool that demonstrates fetching and logging show data. Designed to be extended into a full HTTP proxy server.

### `internal/client/`

**HTTP client with feliratok.eu integration**

The client package is organized by feature with each file containing related functionality:

- `client.go` — `Client` interface definition and constructor (`NewClient`)
- `show_list.go` — Show list fetching from multiple endpoints in parallel
- `subtitles.go` — Subtitle fetching via HTML parsing with pagination support
- `show_subtitles.go` — Fetching show subtitles with third-party ID extraction
- `updates.go` — Update checking implementation
- `download.go` — Subtitle download delegation to SubtitleDownloader service
- `recent_subtitles.go` — Recent subtitles fetching and filtering
- `compression_transport.go` — HTTP transport supporting GZIP, Brotli, and Zstd compression
- `*_test.go` — Unit tests for each feature (one test file per implementation file)
- `client_integration_test.go` — Real API tests (skipped in CI)
- `client_compression_test.go` — Compression support tests
- `errors.go` — Custom error types (e.g., `ErrNotFound`)

### `internal/config/`

**Configuration & Logging**

- `config.go` — Viper-based configuration loader from `config/config.yaml`
  - Supports environment variable overrides (`APP_*` prefix)
  - Singleton `zerolog` logger instance (console output)
  - HTTP client timeout configuration
  - Server address and port settings
  - Log level control

### `internal/models/`

**Data structures** representing all entities:

- `show.go` — TV show with ID, name, year, image URL
- `subtitle.go` — Individual subtitle with language, quality, season/episode, download URL
- `subtitle.go` — `SubtitleCollection` grouping subtitles by show
- `show_subtitles.go` — Composite model combining show, subtitles, and third-party IDs
- `quality.go` — Quality enum (360p, 480p, 720p, 1080p, 2160p, Unknown) with JSON marshaling
- `third_party_ids.go` — IMDB, TVDB, TVMaze, Trakt IDs
- `download_request.go` — Request/response models for subtitle downloads
- `update_check.go` — Update check request/response models

### `internal/parser/`

**HTML & JSON Parsing**

- `interfaces.go` — Generic interfaces:
  - `Parser[T]` — Parses HTML and returns `[]T`
  - `SingleResultParser[T]` — Parses HTML and returns single `T` result
- `show_parser.go` — Parses show listings from HTML tables using `goquery`
  - Extracts show ID, name, year, image URL
  - Handles year header rows
- `third_party_parser.go` — Extracts third-party IDs (IMDB, TVDB, TVMaze, Trakt) from show detail page HTML
  - Uses regex and URL parsing to extract IDs from links
- `subtitle_parser.go` — Parses subtitle tables and pagination links from HTML
- Test files with inline HTML fixtures for comprehensive coverage

### `internal/services/`

**Data Transformation & Downloads**

- **SubtitleConverter** — Normalizes subtitle data
  - `subtitle_converter.go` — Interface definition
  - `subtitle_converter_impl.go` — Implementation:
    - Language conversion: Hungarian → ISO 639-1 codes
    - Quality extraction from subtitle name strings
    - Season/episode number parsing
    - Download URL construction
    - Upload timestamp conversion
  - `subtitle_converter_test.go` — Unit and benchmark tests

- **SubtitleDownloader** — Handles subtitle file downloads
  - `subtitle_downloader.go` — Interface definition
  - `subtitle_downloader_impl.go` — Implementation:
    - Downloads files with content-type detection (magic numbers + MIME types)
    - ZIP file detection and extraction
    - Episode extraction from season packs using regex pattern matching
    - LRU cache for ZIP files (100 entries, 1-hour TTL)
    - ZIP bomb detection to prevent malicious archives
    - Support for multiple subtitle formats (SRT, ASS, VTT, SUB)
  - `subtitle_downloader_test.go` — Comprehensive tests covering:
    - ZIP detection and extraction
    - Cache behavior
    - Episode matching with edge cases
    - ZIP bomb detection

## File Relationships

```
cmd/proxy/main.go
    ↓
client.NewClient()
    ├─→ ShowParser (parses HTML)
    ├─→ SubtitleConverter (normalizes data)
    ├─→ ThirdPartyIdParser (extracts IDs)
    └─→ SubtitleDownloader (downloads & caches files)

All depend on:
    • config.GetConfig() — Configuration
    • config.GetLogger() — Logging
    • models/* — Data structures
```

## Testing Strategy

- **Unit tests** — All packages except `models` have `*_test.go` files
- **No external frameworks** — Uses only Go's standard `testing` package
- **HTTP mocking** — Uses `httptest.Server` for API testing
- **Integration tests** — Real API calls in `client_integration_test.go` (skipped in CI with `CI=true`)
- **Benchmarks** — Performance tests in `subtitle_converter_test.go` and `subtitle_downloader_test.go`
- **Inline fixtures** — HTML and JSON test data embedded in test files

## Naming Conventions

- **Interfaces** — Named with `er` suffix (e.g., `Parser`, `Client`, `Converter`)
- **Implementations** — Concrete types or `Default` prefix (e.g., `DefaultSubtitleConverter`)
- **Constructors** — `New<TypeName>` (e.g., `NewClient`, `NewSubtitleConverter`)
- **Tests** — `Test<Receiver>_<Method>[_<Scenario>]` (e.g., `TestClient_GetShowList_WithProxy`)
