# SuperSubtitles — Data Flow

This document describes the data flow for all major operations in SuperSubtitles.

## Show List Fetching

1. `StreamShowList` fires 3 parallel HTTP requests to different feliratok.eu endpoints
2. Each response is parsed by `ShowParser.ParseHtml` using goquery to extract show ID, name, year, and image URL from HTML tables
3. Results are merged and deduplicated by show ID, preserving first-occurrence order
4. Each deduplicated show is sent to a `StreamResult[Show]` channel as it becomes available
5. Partial failures are tolerated — if at least one endpoint succeeds, results are streamed
6. The gRPC server consumes from the channel and streams `Show` messages to the client
7. `GetShowList` wraps `StreamShowList`, collecting all channel results into a slice

**Implementation:**

- `internal/client/show_list.go` - `StreamShowList` and `GetShowList` methods
- `internal/parser/show_parser.go` - ShowParser implementation

## Subtitle Fetching

`StreamSubtitles` streams subtitles from HTML pages with automatic parallel pagination support.

**Process:**

1. Fetch first page: `GET /index.php?sid=<showID>`
2. Parse HTML using `SubtitleParser.ParseHtmlWithPagination`:
   - Extracts subtitles from 6-column table (Category | Language | Description | Uploader | Date | Download)
   - Parses description for season/episode/release info
   - Detects all qualities from release string
   - Splits comma-separated release groups
   - Detects season packs by looking for special naming patterns
   - Extracts pagination info from `oldal=<page>` parameters
3. Parsed subtitles are sent to a `StreamResult[Subtitle]` channel as they become available
4. If totalPages > 1, fetch remaining pages in parallel **2 pages at a time**:
   - Pages 2–3 fetched in parallel
   - Pages 4–5 fetched in parallel
   - And so on...
5. Subtitles from each page are streamed to the channel as pages complete
6. The gRPC server consumes from the channel and streams `Subtitle` messages to the client
7. `GetSubtitles` wraps `StreamSubtitles`, collecting all channel results into a `SubtitleCollection`

**Example:**

For a show with 5 subtitle pages (like https://feliratok.eu/index.php?sid=3217):

- Request 1: Page 1 (3 subtitles)
- Request 2–3: Pages 2–3 in parallel (3 subtitles each)
- Request 4–5: Pages 4–5 in parallel (3 subtitles each)
- **Total:** 5 requests instead of 5 sequential requests, **~3x faster**

**Implementation Files:**

- `internal/parser/subtitle_parser.go` - HTML table parser with pagination support
- `internal/parser/subtitle_parser_test.go` - 23 comprehensive tests covering quality detection, release groups, season packs, pagination
- `internal/client/subtitles.go` - `StreamSubtitles` and `GetSubtitles` methods with parallel page fetching and pagination
- `internal/client/subtitles_test.go` - Unit tests for pagination (3 tests)

## Third-Party ID Extraction

1. `StreamShowSubtitles` processes shows in batches of 20
2. For each show, it fetches **all subtitles** (all pages), then loads the detail page HTML using the first valid (non-zero) subtitle ID
3. `ThirdPartyIdParser` extracts IDs from `div.adatlapRow a` links using regex and URL parsing
4. For each show, a `ShowSubtitleItem` with `ShowInfo` (show + third-party IDs) is sent to the channel first, followed by individual `ShowSubtitleItem` entries for each subtitle
5. The gRPC server consumes from the channel and streams `ShowSubtitleItem` messages to the client

**Implementation:**

- `internal/parser/third_party_parser.go` - ThirdPartyIdParser implementation
- `internal/client/show_subtitles.go` - `StreamShowSubtitles` and `GetShowSubtitles` methods with batching
- `internal/models/show_subtitles.go` - `ShowInfo` and `ShowSubtitleItem` models

## Recent Subtitles Fetching

`StreamRecentSubtitles` streams the latest subtitles from the main show page with optional ID filtering.

**Process:**

1. Fetch main page: `GET /index.php?tab=sorozat`
2. Parse HTML using `SubtitleParser.ParseHtml`:
   - Same 6-column table structure as individual show subtitle pages
   - **Extracts show ID** from the category column's link (`index.php?sid=<showID>`)
   - Parses subtitle details (language, season/episode, uploader, date, download URL)
3. Filter subtitles by ID:
   - If `sinceID` is provided, only returns subtitles with `ID > sinceID` (numeric integer comparison on `Subtitle.ID`)
   - Useful for incremental updates and polling for new content
4. Stream each filtered `Subtitle` directly to the channel as it's processed
5. The gRPC server consumes from the channel and streams `Subtitle` messages to the client
6. `GetRecentSubtitles` wraps `StreamRecentSubtitles`, collecting channel results into `[]Subtitle`

**Key Features:**

- **Efficient filtering**: Only processes subtitles newer than a given ID (numeric comparison)
- **Direct subtitle streaming**: Each subtitle is streamed individually with `show_id` and `show_name` for client-side grouping
- **Reuses existing parsers**: Same `SubtitleParser` used for both individual show pages and main page
- **Lightweight**: No additional HTTP requests needed — all data comes from the main page parse

**Implementation Files:**

- `internal/parser/subtitle_parser.go` - `extractShowIDFromCategory` method extracts show ID from HTML
- `internal/client/recent_subtitles.go` - `StreamRecentSubtitles` and `GetRecentSubtitles` methods with filtering
- `internal/client/recent_subtitles_test.go` - 4 comprehensive tests covering filtering, empty results, and errors
- `internal/models/subtitle.go` - `ShowID` field on Subtitle model

**Example Use Cases:**

- Polling for new subtitles every N minutes
- Building a feed of recent uploads
- Change detection and notifications
- Incremental data synchronization

## Subtitle Download with Episode Extraction

1. `DownloadSubtitle` method in the Client interface accepts a `DownloadRequest` with the subtitle ID
2. Client builds the download URL as `index.php?action=letolt&felirat=<subtitleID>` against the configured base domain
3. **`SubtitleDownloader` service is the primary download handler for all subtitle files**:
   - **Regular subtitle files** (SRT, ASS, VTT, SUB): Downloaded and returned with correct content-type and extension
   - **ZIP files without episode number**: Entire ZIP returned (for manual extraction)
   - **ZIP files with episode number**:
     - ZIP file is downloaded (or retrieved from cache)
     - Episode pattern matching using regex with word boundaries: `S03E01`, `s03e01`, `3x01`, `E01` (with guards against false positives like E01 matching E010)
     - Specific episode subtitle extracted from the ZIP archive
     - Only the requested episode file is returned with correct content-type based on file extension
4. **Multi-Format Support**:
   - SRT (SubRip) - `application/x-subrip`
   - ASS (Advanced SubStation Alpha) - `application/x-ass`, `text/ass`
   - VTT (WebVTT) - `text/vtt`, `text/webvtt`
   - SUB (MicroDVD) - `application/x-sub`
   - ZIP archives - `application/zip`
   - Unknown formats default to `application/octet-stream`
5. **Caching Strategy**:
   - LRU cache with 100-entry capacity and 1-hour TTL
   - Only ZIP files are cached (regular subtitle files are small and not cached)
   - Cache key is the download URL
   - Multiple episode requests from same season pack use cached ZIP
6. **File Structure Support**:
   - Flat ZIP structure (all files in root)
   - Nested folders (e.g., `ShowName.S03/ShowName.S03E01.srt`)
   - Various naming patterns (uppercase, lowercase, different separators)

**Implementation Files:**

- `internal/services/subtitle_downloader.go` - Interface definition
- `internal/services/subtitle_downloader_impl.go` - Implementation with caching, ZIP extraction, and format detection
- `internal/services/subtitle_downloader_test.go` - Comprehensive unit tests and benchmarks covering ZIP detection/extraction, caching, and edge cases
- `internal/models/download_request.go` - Request/response models
- `internal/client/client.go` - Client integration
