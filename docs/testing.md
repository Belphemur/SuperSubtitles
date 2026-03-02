# SuperSubtitles — Testing Infrastructure

## HTML Fixture Generator (`internal/testutil/html_fixtures.go`)

The project uses a **programmatic HTML generation system** for tests instead of hardcoded HTML strings. This provides maintainable, flexible test fixtures that match the actual structure of feliratok.eu pages.

**Key Features:**

- **Type-safe configuration**: `SubtitleRowOptions` and `ShowRowOptions` structs define table rows with sensible defaults
- **Automatic styling**: Background colors alternate automatically, flag images map to languages
- **Pagination support**: `GenerateSubtitleTableHTMLWithPagination` includes page navigation elements
- **Realistic structure**: Generated HTML matches the actual feliratok.eu DOM structure (table classes, div nesting, onclick handlers)

**Available Generators:**

| Function                                  | Purpose                                         | Key Parameters                                         |
| ----------------------------------------- | ----------------------------------------------- | ------------------------------------------------------ |
| `GenerateSubtitleTableHTML`               | Basic subtitle listing table                    | `[]SubtitleRowOptions` with language, titles, uploader |
| `GenerateSubtitleTableHTMLWithPagination` | Subtitle table with page navigation             | Rows + `currentPage`, `totalPages`, `useOldalParam`    |
| `GenerateShowTableHTML`                   | TV show listing with year headers               | `[]ShowRowOptions` with show ID, name, year            |
| `GenerateShowTableHTMLMultiColumn`        | TV show listing with multiple shows per row     | `[]ShowRowOptions` + `columnsPerRow` (default 2)       |
| `GenerateThirdPartyIDHTML`                | Episode detail page with IMDB/TVDB/TVMaze/Trakt | Individual ID parameters                               |
| `GeneratePaginationHTML`                  | Standalone pagination elements                  | `currentPage`, `totalPages`, `useOldalParam`           |

**Example Usage:**

```go
html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
    {
        ShowID:           2967,
        Language:         "Magyar",
        MagyarTitle:      "Test Show S01E01",
        EredetiTitle:     "Test Show S01E01",
        Uploader:         "TestUser",
        UploadDate:       "2024-01-15",
        DownloadAction:   "letolt",
        DownloadFilename: "test.srt",
        SubtitleID:       1737439811,
    },
})
```

**Benefits:**

- **Maintainability**: Changing HTML structure requires updating one generator, not dozens of test strings
- **Readability**: Tests clearly express intent through configuration structs
- **Flexibility**: Easy to add edge cases (missing fields, different languages, status flags)
- **Consistency**: All tests use the same HTML structure, reducing false negatives

## Test Strategy

- **No external test frameworks**: All tests use the Go standard library `testing` package with `httptest` for HTTP mocking
- **Programmatic fixtures**: HTML fixtures generated via `testutil` package instead of hardcoded strings
- **Stream collection helpers**: `testutil` provides helpers to consume streams in tests (`CollectShows`, `CollectSubtitles`, `CollectShowSubtitles`)
- **Streaming-first testing**: Tests consume from client streaming methods and use testutil helpers to collect results
- **Parallel page fetching**: Tests verify pagination with 2-page parallel batches
- **`t.Parallel()` throughout**: Most test functions call `t.Parallel()` so Go runs them concurrently within each package. Tests that assert on shared global Prometheus counters are intentionally left sequential to avoid count interference.
- **Integration test guards**: `client_integration_test.go` checks for `CI` / `SKIP_INTEGRATION_TESTS` env vars to skip live requests
- **Benchmark coverage**: Performance tests for critical paths (ZIP extraction)

## Stream Collection Helpers (`internal/testutil/stream_helpers.go`)

Since the client exposes only streaming methods, tests need helpers to consume streams and collect results. These helpers are **test-only** and should never be used in production code.

**Available Helpers:**

| Function                    | Purpose                                                          | Returns                               |
| --------------------------- | ---------------------------------------------------------------- | ------------------------------------- |
| `CollectShows`              | Consumes a `Show` stream and returns a slice                     | `([]models.Show, error)`              |
| `CollectSubtitles`          | Consumes a `Subtitle` stream and returns a `SubtitleCollection`  | `(*models.SubtitleCollection, error)` |
| `CollectShowSubtitles`      | Consumes a `ShowSubtitles` stream and returns a slice            | `([]models.ShowSubtitles, error)`     |

**Example Usage:**

```go
// Instead of: shows, err := client.GetShowList(ctx)
shows, err := testutil.CollectShows(ctx, client.StreamShowList(ctx))

// Instead of: subs, err := client.GetSubtitles(ctx, showID)
subs, err := testutil.CollectSubtitles(ctx, client.StreamSubtitles(ctx, showID))

// Instead of: results, err := client.GetShowSubtitles(ctx, shows)
results, err := testutil.CollectShowSubtitles(ctx, client.StreamShowSubtitles(ctx, shows))
```

**Why Test-Only?**

- Production code (gRPC server) consumes streams directly without buffering
- Collecting entire streams negates the benefits of streaming (memory usage, time-to-first-result)
- Tests need deterministic full results for assertions
- Keeps the distinction clear between test and production code

## Test Coverage

### Parser Tests

- `internal/parser/show_parser_test.go` - Show listing parser tests
- `internal/parser/subtitle_parser_test.go` - 28+ comprehensive tests covering:
  - Quality detection (360p-2160p)
  - Release group parsing
  - Season pack detection
  - Episode title extraction (with handling for regular episodes and season packs)
  - Pagination
  - Show ID extraction
  - Language conversion
- `internal/parser/third_party_parser_test.go` - Third-party ID extraction tests

### Client Tests

- `internal/client/show_list_test.go` - Show list fetching with parallel endpoints
- `internal/client/updates_test.go` - Update checking logic and edge cases
- `internal/client/subtitles_test.go` - Subtitle fetching with pagination (3 tests)
- `internal/client/recent_subtitles_test.go` - Recent subtitles with filtering (4 tests)
- `internal/client/show_subtitles_test.go` - Show subtitles with batching (1 test)
- `internal/client/download_test.go` - Download operations
- `internal/client/client_compression_test.go` - Compression support tests (gzip, brotli, zstd)
- `internal/client/compression_transport_test.go` - Compression transport unit tests
- `internal/client/client_integration_test.go` - Live integration tests (skipped in CI)

### Service Tests

- `internal/services/subtitle_downloader_test.go` - Download service tests with:
  - ZIP detection and extraction
  - Episode-specific extraction
  - Caching behavior
  - Multi-format support
  - Benchmark tests for performance
### Cache Tests

- `internal/cache/memory_test.go` - In-memory LRU cache tests:
  - Get/Set, Contains, Len operations
  - LRU eviction when exceeding capacity
  - Eviction callback invocation
  - Overwrite behavior
- `internal/cache/factory_test.go` - Factory and provider registry tests:
  - Creating caches via factory
  - Unknown provider error handling
  - Registered provider listing and sorting
  - Redis connection failure handling
- `internal/cache/redis_test.go` - Redis/Valkey cache tests (skipped without `REDIS_ADDRESS` env var):
  - Get/Set, Contains, Len operations
  - LRU eviction when exceeding capacity
  - Touch-promotes-entry (Get refreshes LRU ordering)
  - Eviction callback invocation

### Metrics Tests

- `internal/metrics/metrics_test.go` - Prometheus metrics tests:
  - Counter and gauge registration and increment verification
  - `SubtitleDownloadsTotal` counter with success/error labels
  - Cache metrics (hits, misses, evictions, entries)
  - `NewHTTPServer` creation and default port behavior

## Running Tests

> **Note:** Some cache tests require a running Valkey (or Redis-compatible) server. The
> `internal/cache/redis_test.go` tests are automatically skipped unless the `REDIS_ADDRESS`
> environment variable is set (e.g., `REDIS_ADDRESS=localhost:6379`). Valkey 8+ (or Redis 7.4+)
> is required for the HPEXPIRE command used by the Redis cache provider. Without Valkey running
> locally you can still run all other tests — only the Redis-specific tests will be skipped.

```bash
# All tests (Redis tests are skipped without REDIS_ADDRESS)
go test ./...

# With race detector (required before commits)
go test -race ./...

# With Valkey/Redis running locally (enables all cache tests)
REDIS_ADDRESS=localhost:6379 go test ./internal/cache/...

# Start Valkey with Docker for local development
# docker run -d -p 6379:6379 valkey/valkey:latest

# Specific package
go test ./internal/parser/...

# With coverage
go test -coverprofile=coverage.txt -covermode=atomic ./...

# Integration tests (not run in CI)
SKIP_INTEGRATION_TESTS=false go test ./internal/client/...
```
