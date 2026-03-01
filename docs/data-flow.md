# SuperSubtitles — Data Flow

This document describes the data flow for all major operations in SuperSubtitles.

## Show List Fetching

1. `StreamShowList` fires 3 parallel HTTP requests to different feliratok.eu endpoints
2. Each endpoint is handled by `fetchEndpointPages`, which fetches page 1 first
3. Page 1 HTML is converted to UTF-8 using `NewUTF8Reader` (automatic charset detection), then parsed by `ShowParser.ParseHtml` using goquery to extract show ID, name, year, and image URL from HTML tables
4. `ShowParser.ExtractLastPage` inspects the pagination HTML (`div.pagination` links with `oldal=N` parameters) to discover the total page count
5. If more than 1 page exists, remaining pages (2..lastPage) are fetched in **parallel batches of 10** (`pageBatchSize`). Each batch waits for completion before starting the next
6. Results from all pages are merged and deduplicated by show ID using a `sync.Map`, preserving first-occurrence order
7. Each deduplicated show is sent to a `models.StreamResult[models.Show]` channel as it becomes available
8. Partial failures are tolerated — if at least one endpoint succeeds, results are streamed; individual page failures within an endpoint log warnings but don't fail the endpoint
9. The gRPC server consumes from the channel and streams `Show` messages to the client
10. Tests use `testutil.CollectShows` helper to collect stream results into a slice

**Pagination Example:**

For the `nem-all-forditas-alatt` endpoint with 42 pages:

- Request 1: Page 1 (discovers 42 total pages)
- Request 2–11: Pages 2–11 in parallel (batch 1)
- Request 12–21: Pages 12–21 in parallel (batch 2)
- Request 22–31: Pages 22–31 in parallel (batch 3)
- Request 32–41: Pages 32–41 in parallel (batch 4)
- Request 42: Page 42 (batch 5)
- **Total:** 42 requests completed in ~6 rounds instead of 42 sequential requests

**Implementation:**

- `internal/client/show_list.go` - `StreamShowList`, `fetchEndpointPages`, `fetchPage`, `streamShowsFromBody` methods
- `internal/parser/show_parser.go` - `ShowParser` with `ParseHtml` and `ExtractLastPage` methods
- `internal/testutil/stream_helpers.go` - `CollectShows` test helper

## Subtitle Fetching

`StreamSubtitles` streams subtitles from HTML pages with automatic parallel pagination support.

**Process:**

1. Fetch first page: `GET /index.php?sid=<showID>`
2. Parse HTML using `SubtitleParser.ParseHtmlWithPagination` (content is first converted to UTF-8 via `NewUTF8Reader`):
   - Extracts subtitles from 6-column table (Category | Language | Description | Uploader | Date | Download)
   - Parses description for season/episode/release info
   - Detects all qualities from release string
   - Splits comma-separated release groups
   - Detects season packs by looking for special naming patterns
   - **Extracts episode title**: For regular episodes (with SxEE pattern), extracts only the episode title portion from the description; for season packs, returns empty string
   - Extracts pagination info from `oldal=<page>` parameters
3. Parsed subtitles are sent to a `models.StreamResult[models.Subtitle]` channel as they become available
4. If totalPages > 1, fetch remaining pages in parallel **2 pages at a time**:
   - Pages 2–3 fetched in parallel
   - Pages 4–5 fetched in parallel
   - And so on...
5. Subtitles from each page are streamed to the channel as pages complete
6. The gRPC server consumes from the channel and streams `Subtitle` messages to the client
7. Tests use `testutil.CollectSubtitles` helper to collect stream results into a `SubtitleCollection`

**Example:**

For a show with 5 subtitle pages (like https://feliratok.eu/index.php?sid=3217):

- Request 1: Page 1 (3 subtitles)
- Request 2–3: Pages 2–3 in parallel (3 subtitles each)
- Request 4–5: Pages 4–5 in parallel (3 subtitles each)
- **Total:** 5 requests instead of 5 sequential requests, **~3x faster**

**Implementation Files:**

- `internal/parser/subtitle_parser.go` - HTML table parser with pagination support
- `internal/parser/subtitle_parser_test.go` - 23 comprehensive tests covering quality detection, release groups, season packs, pagination
- `internal/client/subtitles.go` - `StreamSubtitles` method with parallel page fetching and pagination
- `internal/client/subtitles_test.go` - Unit tests for pagination (3 tests)
- `internal/testutil/stream_helpers.go` - `CollectSubtitles` test helper

## Third-Party ID Extraction

1. `StreamShowSubtitles` processes shows in batches of 20
2. For each show, it accumulates all subtitles from `StreamSubtitles`, then loads the detail page HTML using the first valid (non-zero) subtitle ID
3. `ThirdPartyIdParser` converts HTML to UTF-8 via `NewUTF8Reader`, then extracts IDs from `div.adatlapRow a` links using regex and URL parsing
4. For each show, a complete `ShowSubtitles` (containing show info, third-party IDs, and all subtitles) is sent to the channel
5. The gRPC server converts each `ShowSubtitles` to a `ShowSubtitlesCollection` proto message and streams it to the client
6. Tests use `testutil.CollectShowSubtitles` helper to collect stream results into a slice

**Implementation:**

- `internal/parser/third_party_parser.go` - ThirdPartyIdParser implementation
- `internal/client/show_subtitles.go` - `StreamShowSubtitles` method with batching and per-show accumulation
- `internal/models/show_subtitles.go` - `ShowSubtitles` model
- `internal/testutil/stream_helpers.go` - `CollectShowSubtitles` test helper

## Recent Subtitles Fetching

`StreamRecentSubtitles` streams the latest subtitles from the main show page with optional ID filtering, grouped by show.

**Process:**

1. Fetch main page: `GET /index.php?tab=sorozat`
2. Parse HTML using `SubtitleParser.ParseHtml`:
   - Same 6-column table structure as individual show subtitle pages
   - **Extracts show ID** from the category column's link (`index.php?sid=<showID>`)
   - Parses subtitle details (language, season/episode, uploader, date, download URL)
3. Filter subtitles by ID:
   - If `sinceID` is provided, only returns subtitles with `ID > sinceID` (numeric integer comparison on `Subtitle.ID`)
   - Useful for incremental updates and polling for new content
4. Group filtered subtitles by show, preserving encounter order
5. For each show, fetch the detail page to get third-party IDs using the first valid subtitle ID
6. Stream a complete `ShowSubtitles` (show info + all subtitles) for each show
7. The gRPC server converts each `ShowSubtitles` to a `ShowSubtitlesCollection` proto message and streams it to the client
8. Tests use `testutil.CollectShowSubtitles` helper to collect stream results into a slice

**Key Features:**

- **Efficient filtering**: Only processes subtitles newer than a given ID (numeric comparison)
- **Per-show grouping**: Subtitles are grouped by show and sent as complete `ShowSubtitles` items
- **Third-party ID enrichment**: Fetches detail pages to include IMDB, TVDB, TVMaze, Trakt IDs with show info
- **Reuses existing parsers**: Same `SubtitleParser` used for both individual show pages and main page
- **Partial failure resilience**: If a detail page fetch fails, show is still sent with empty third-party IDs

**Implementation Files:**

- `internal/parser/subtitle_parser.go` - `extractShowIDFromCategory` method extracts show ID from HTML
- `internal/client/recent_subtitles.go` - `StreamRecentSubtitles` method with filtering, per-show grouping, and detail page fetching
- `internal/client/recent_subtitles_test.go` - 5 comprehensive tests covering filtering, empty results, errors, and show info deduplication
- `internal/models/subtitle.go` - `ShowID` field on Subtitle model
- `internal/testutil/stream_helpers.go` - `CollectShowSubtitles` test helper

**Example Use Cases:**

- Polling for new subtitles every N minutes
- Building a feed of recent uploads
- Change detection and notifications
- Incremental data synchronization

## Subtitle Download with Episode Extraction

1. `DownloadSubtitle` method in the Client interface accepts a `DownloadRequest` with the subtitle ID
2. Client builds the download URL as `index.php?action=letolt&felirat=<subtitleID>` against the configured base domain
3. **`SubtitleDownloader` service is the primary download handler for all subtitle files**:
   - **Regular subtitle files** (SRT, ASS, VTT, SUB): Downloaded, converted to UTF-8 (via `convertToUTF8` charset detection), and returned with correct content-type and extension
   - **ZIP files without episode number**: Entire ZIP returned (for manual extraction)
   - **ZIP files with episode number**:
     - ZIP file is downloaded (or retrieved from cache)
     - Episode pattern matching using regex with word boundaries: `S03E01`, `s03e01`, `3x01`, `E01` (with guards against false positives like E01 matching E010)
     - Specific episode subtitle extracted from the ZIP archive
     - ZIP entry filenames are sanitized to valid UTF-8 (replacing invalid bytes with U+FFFD)
     - Extracted text subtitle content is converted to UTF-8 via charset detection
     - Only the requested episode file is returned with correct content-type based on file extension
4. **Multi-Format Support**:
   - SRT (SubRip) - `application/x-subrip`
   - ASS (Advanced SubStation Alpha) - `application/x-ass`, `text/ass`
   - VTT (WebVTT) - `text/vtt`, `text/webvtt`
   - SUB (MicroDVD) - `application/x-sub`
   - ZIP archives - `application/zip`
   - Unknown formats default to `application/octet-stream`
5. **Caching Strategy**:
   - Pluggable LRU cache backend selected via `cache.type` config: `memory` (default) or `redis`
   - **Memory backend**: In-process LRU cache using hashicorp/golang-lru with configurable size and TTL
   - **Redis/Valkey backend**: Uses a single Redis hash for values with per-field TTL (`HPEXPIRE`, requires Redis 7.4+ / Valkey 8+) and a sorted set for LRU access-time tracking — only 2 Redis keys regardless of cache size. Atomic Lua scripts ensure get-and-touch / set-and-evict consistency
   - Only ZIP files are cached (regular subtitle files are small and not cached)
   - Cache key is the download URL
   - Multiple episode requests from same season pack use cached ZIP
   - Falls back to memory if the configured backend fails to initialize
6. **File Structure Support**:
   - Flat ZIP structure (all files in root)
   - Nested folders (e.g., `ShowName.S03/ShowName.S03E01.srt`)
   - Various naming patterns (uppercase, lowercase, different separators)

**Implementation Files:**

- `internal/services/subtitle_downloader.go` - Interface definition
- `internal/services/subtitle_downloader_impl.go` - Implementation with caching, ZIP extraction, format detection, and UTF-8 conversion
- `internal/services/subtitle_downloader_test.go` - Comprehensive unit tests and benchmarks covering ZIP detection/extraction, caching, and edge cases
- `internal/models/download_request.go` - Request/response models
- `internal/client/client.go` - Client integration
