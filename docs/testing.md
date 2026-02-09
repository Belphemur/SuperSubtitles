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
- **Parallel page fetching**: Tests verify pagination with 2-page parallel batches
- **Integration test guards**: `client_integration_test.go` checks for `CI` / `SKIP_INTEGRATION_TESTS` env vars to skip live requests
- **Benchmark coverage**: Performance tests for critical paths (ZIP extraction)

## Test Coverage

### Parser Tests

- `internal/parser/show_parser_test.go` - Show listing parser tests
- `internal/parser/subtitle_parser_test.go` - 24+ comprehensive tests covering:
  - Quality detection (360p-2160p)
  - Release group parsing
  - Season pack detection
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

## Running Tests

```bash
# All tests
go test ./...

# With race detector (required before commits)
go test -race ./...

# Specific package
go test ./internal/parser/...

# With coverage
go test -coverprofile=coverage.txt -covermode=atomic ./...

# Integration tests (not run in CI)
SKIP_INTEGRATION_TESTS=false go test ./internal/client/...
```
