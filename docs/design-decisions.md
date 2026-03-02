# SuperSubtitles — Design Decisions

This document explains key architectural and design decisions made in the SuperSubtitles project.

## Partial Failure Resilience

**Decision**: The client returns whatever data it successfully fetched, logging warnings for failed endpoints rather than failing the entire operation.

**Rationale**:

- feliratok.eu endpoints may be temporarily unavailable
- Users benefit from partial data rather than complete failure
- Warnings in logs allow monitoring of endpoint health

**Implementation**: All parallel fetching operations collect errors but still return successful results if any endpoints succeed.

## Generic Parser Interfaces

**Decision**: Use `Parser[T]` and `SingleResultParser[T]` generic interfaces for type-safe HTML parsing.

**Rationale**:

- Type safety prevents runtime errors
- Reusable pattern for different model types
- Clear contract for parser implementations

**Implementation**: `internal/parser/interfaces.go` defines generic interfaces implemented by ShowParser, SubtitleParser, and ThirdPartyIdParser.

## Batch Processing

**Decision**: Show subtitle fetching is batched (20 concurrent requests) to avoid overwhelming the upstream server.

**Rationale**:

- Prevents overloading feliratok.eu infrastructure
- Balances speed with responsible resource usage
- Allows rate limiting if needed in the future

**Implementation**: `GetShowSubtitles` processes shows in batches of 20 with parallel processing within each batch.

## Programmatic Test Fixtures

**Decision**: HTML fixtures generated via reusable utility functions instead of hardcoded strings.

**Rationale**:

- HTML structure changes require updating only one generator
- Tests express intent through configuration structs
- Consistency across all tests
- Easy to add edge cases
- Prevents brittle hardcoded HTML in tests

**Implementation**: `internal/testutil/html_fixtures.go` provides generators for all HTML table types:

- `GenerateSubtitleTableHTML` - Basic subtitle listings
- `GenerateSubtitleTableHTMLWithPagination` - Subtitles with pagination
- `GenerateShowTableHTML` - Single-column show listings
- `GenerateShowTableHTMLMultiColumn` - Multi-column grid layouts (e.g., 2 shows per row)
- `GenerateThirdPartyIDHTML` - Third-party ID detail pages
- `GeneratePaginationHTML` - Pagination elements

**Evolution**: Added `GenerateShowTableHTMLMultiColumn` to support the actual website's grid layout for special show listing pages, ensuring tests match production HTML structure.

## Parser Reusability

**Decision**: The `SubtitleParser` is used for both individual show subtitle pages and the main page.

**Rationale**:

- Both pages share the same HTML table structure
- Reduces code duplication
- Single source of truth for subtitle parsing logic
- Only requires one additional method (`extractShowIDFromCategory`) for main page support

**Implementation**:

- `SubtitleParser.ParseHtml` works for both page types
- `extractShowIDFromCategory` method added to extract show ID from category column when available

## Parser Handles All Data Normalization

**Decision**: The SubtitleParser directly handles language conversion, quality extraction, season/episode parsing, and data normalization rather than using a separate converter service.

**Rationale**:

- Parsing and normalization are tightly coupled operations
- Eliminates unnecessary intermediate data structures
- Reduces code complexity and maintenance burden
- Parser has all HTML context needed for normalization
- Single responsibility: transform HTML → normalized models

**Implementation**:

- `SubtitleParser` includes:
  - `convertLanguageToISO` for Hungarian → ISO 639-1 conversion
  - `parseReleaseInfo` for quality and release group extraction
  - `parseDescription` for season/episode/show name extraction
  - `detectQuality` for quality enum conversion
- All normalization happens during HTML parsing in one pass

## Client Architecture

**Decision**: Keep unified client interface rather than splitting into multiple specialized clients.

**Current State**:

- ~600 lines in `client.go` handling all operations
- Single `Client` interface provides unified API
- Clear method separation with comprehensive tests

**Rationale**:

- Current structure is manageable and well-tested
- Single interface is convenient for consumers
- Premature splitting adds unnecessary complexity

**Future Consideration**: If the client grows significantly (>1000 lines) or new features require substantial complexity, consider splitting into:

1. Show Client (GetShowList, GetShowSubtitles)
2. Subtitle Client (GetSubtitles, GetRecentSubtitles)
3. Metadata Client (third-party IDs, update checking)
4. Download Client (already delegates to SubtitleDownloader service)

## Parallel Pagination

**Decision**: Fetch paginated content in parallel batches rather than sequentially. Batch sizes differ by context:

- **Subtitle pages**: Batch size of 2 (~3x faster for shows with many subtitle pages)
- **Show list pages**: Batch size of 10 (show list endpoints like `nem-all-forditas-alatt` can have 40+ pages)

**Rationale**:

- Dramatically faster for endpoints with many pages (42 pages in ~6 rounds instead of 42 sequential requests)
- Balances speed with server load
- First page always fetched alone to discover total pages via `ExtractLastPage` (show lists) or `extractPaginationInfo` (subtitles)
- Show list uses a larger batch size because individual show pages are lightweight HTML responses

**Implementation**:

- **Subtitles**: Pages 2-3 fetched together, then 4-5, etc. (`internal/client/subtitles.go`)
- **Show lists**: Pages 2-11 fetched together, then 12-21, etc. (`internal/client/show_list.go` with `pageBatchSize = 10`). `ShowParser.ExtractLastPage` parses `div.pagination` links to determine the highest page number from `oldal=` URL parameters.

## Error Handling Strategy

**Decision**: Use custom error types with `Is()` method support, wrap errors with `fmt.Errorf`, and prefer partial success over complete failure.

**Rationale**:

- `errors.Is()` support enables proper error checking
- Wrapped errors preserve context
- Partial success maximizes data availability
- Logged warnings enable monitoring

**Implementation**:

- `ErrNotFound` custom error in `internal/client/errors.go`
- `ErrSubtitleNotFoundInZip` custom error in `internal/services/subtitle_downloader.go` — returned when a requested episode is not found inside a ZIP season pack; checked in the gRPC server to return `codes.NotFound` instead of `codes.Internal`
- All errors wrapped with context
- Parallel operations collect errors but return successful results

## Server-Side Streaming RPCs

**Decision**: Use server-side streaming for 4 of 6 gRPC RPCs (`GetShowList`, `GetSubtitles`, `GetShowSubtitles`, `GetRecentSubtitles`). Only `CheckForUpdates` and `DownloadSubtitle` remain unary.

**Rationale**:

- Improves time-to-first-result by sending items as they become available
- Reduces server memory usage — no need to buffer entire response collections
- Natural fit for list/collection endpoints that aggregate data from multiple sources
- Enables progressive rendering on the client side
- Unary RPCs are kept for single-value responses (update check, file download)

**Implementation**:

- Proto definitions use `returns (stream T)` for streaming RPCs
- `ShowSubtitlesCollection` bundles show metadata (with third-party IDs) and all subtitles per show into a single streamed message
- gRPC server methods consume from client streaming channels and call `stream.Send()` per item
- Removed `GetShowListResponse`, `GetSubtitlesResponse`, `GetShowSubtitlesResponse`, `GetRecentSubtitlesResponse` wrapper messages

## Streaming-First Client Architecture

**Decision**: The client exposes **only** streaming methods for list/collection operations. Non-streaming `GetX` methods have been removed from the client interface. Test helpers in `internal/testutil` provide collection utilities for tests.

**Rationale**:

- Channels are Go's natural primitive for streaming data
- Eliminates dual API surface (streaming + non-streaming)
- Forces consumers to handle streaming properly from the start
- Reduces memory usage — no intermediate buffering of full collections
- Simplifies client interface and reduces code duplication
- Tests use `testutil` helpers (`CollectShows`, `CollectSubtitles`, `CollectShowSubtitles`) to consume streams when needed
- Production gRPC server consumes streams directly without buffering

**Implementation**:

- `StreamResult[T]` generic struct with `Value T` and `Err error` fields in `internal/models/stream_result.go`
- Moved from client package to models package to avoid circular dependencies
- Client interface exposes only: `StreamShowList`, `StreamSubtitles`, `StreamShowSubtitles`, `StreamRecentSubtitles`, `CheckForUpdates`, `DownloadSubtitle`
- All streaming methods return read-only `<-chan models.StreamResult[T]` channels
- Channels are closed when all data has been sent or on error
- `internal/testutil/stream_helpers.go` provides test-only collection helpers

## StreamResult in Models Package

**Decision**: `StreamResult[T]` is defined in `internal/models` rather than `internal/client`.

**Rationale**:

- Avoids circular dependency: `testutil` needs to reference `StreamResult`, but also needs `models` (which `client` depends on)
- Makes `StreamResult` a core domain type alongside other model types
- Allows other packages to use `StreamResult` without depending on `client`
- Clean separation: models define data structures, client implements streaming

**Implementation**:

- `models.StreamResult[T]` in `internal/models/stream_result.go`
- Client methods return `<-chan models.StreamResult[T]`
- Testutil helpers accept `<-chan models.StreamResult[T]`
- gRPC server mocks use `models.StreamResult[T]`

## ShowSubtitlesCollection Streaming Model

**Decision**: Use a `ShowSubtitlesCollection` message to stream complete show data (show info + all subtitles) in `GetShowSubtitles` and `GetRecentSubtitles`. Each streamed message contains one show's metadata (with third-party IDs) and all its subtitles.

**Rationale**:

- Simplifies client consumption — each message is self-contained with a show and all its subtitles
- No need for clients to reconstruct show-subtitle groupings from interleaved messages
- For `GetRecentSubtitles`, show info includes third-party IDs fetched from detail pages, enabling clients to discover new shows
- Client accumulates subtitles per show before sending, trading slightly more memory for simpler semantics
- Internal `ShowSubtitles` model in `internal/models/show_subtitles.go` maps directly to the proto structure

**Implementation**:

- Proto `ShowSubtitlesCollection` contains `ShowInfo show_info` and `repeated Subtitle subtitles`
- Client `StreamShowSubtitles` and `StreamRecentSubtitles` both return `<-chan models.StreamResult[models.ShowSubtitles]`
- `convertShowSubtitlesToProto` converter maps `models.ShowSubtitles` to proto `ShowSubtitlesCollection`

## Show Name Extraction via DOM Traversal

**Decision**: Use direct DOM sibling traversal (`.Next()`) to find show names instead of iterating through all table cells with string matching.

**Problem**: The original implementation used a complex flag-based iteration pattern, searching for the link's href in all cells, then looking for subsequent `td.sangol` elements. This was fragile and prone to incorrect matches in multi-column layouts.

**Solution**: Simplified to:

1. Get the parent `<td>` of the show link (the image cell)
2. Use `.Next()` to get the immediately following sibling `<td>`
3. Verify it has class "sangol" (the name cell)
4. Extract the show name from the first `<div>` in that cell

**Rationale**:

- Mirrors the actual HTML structure where name cells always follow image cells
- Eliminates string matching and iteration complexity
- Works correctly with multi-column layouts (2+ shows per row)
- More maintainable and easier to understand
- Preserves show names with parenthetical alternate titles (e.g., "Cash Queens (Les Lionnes)")

**Implementation**: `internal/parser/show_parser.go` - `extractShowNameFromGoquery` method uses goquery's `Closest()` and `Next()` for reliable sibling navigation.

## UTF-8 Safety for Scraped Content

**Decision**: Apply multi-layer UTF-8 sanitization across the entire data pipeline — HTML parsing, subtitle file content, ZIP filenames, and gRPC serialization.

**Rationale**:

- feliratok.eu serves HTML in various encodings (ISO-8859-1, Windows-1252, UTF-8) and may not always declare charset correctly
- ZIP archives contain filenames encoded in the creator's local encoding (e.g., CP437, ISO-8859-1), which are not valid UTF-8
- Protocol Buffers requires all `string` fields to be valid UTF-8; invalid sequences cause marshaling errors at the gRPC transport layer
- Subtitle files downloaded from the site may be in non-UTF-8 encodings

**Implementation**:

- `internal/parser/charset.go` - `NewUTF8Reader` wraps `io.Reader` with `golang.org/x/net/html/charset` for automatic encoding detection and conversion to UTF-8, used by all HTML parsers
- `internal/grpc/converters.go` - `sanitizeUTF8` / `sanitizeUTF8Slice` use `strings.ToValidUTF8` to replace invalid sequences with U+FFFD as a defense-in-depth safety net before protobuf marshaling
- `internal/services/subtitle_downloader_impl.go` - `convertToUTF8` uses `golang.org/x/text/transform` with `charset.DetermineEncoding` for subtitle file content; `strings.ToValidUTF8` for ZIP entry filenames

## Standard gRPC Health Checking Protocol

**Decision**: Implement the standard gRPC health checking protocol (`grpc.health.v1.Health`) rather than a custom health endpoint.

**Rationale**:

- Industry-standard protocol widely supported by infrastructure tools
- Native support in Kubernetes, service meshes, and load balancers
- Works with standard tooling like `grpc_health_probe`
- No custom client code needed for health checks
- Enables both overall server health and per-service health reporting

**Implementation**:

- `cmd/proxy/main.go` registers the `grpc.health.v1.Health` service
- Reports `SERVING` status for both overall server (`""`) and `supersubtitles.v1.SuperSubtitlesService`
- Docker HEALTHCHECK uses `grpc_health_probe` binary (downloaded in separate build stage with SHA256 verification)
- Multi-stage Dockerfile: download stage fetches `grpc_health_probe` and verifies checksum against official release checksums
- Health check runs every 30s with 10s timeout, 5s start period, 3 retries
- Final image excludes build tools (wget) for minimal size

## Pluggable Cache with Factory Pattern

**Decision**: Abstract the ZIP file cache behind an interface (`cache.Cache`) with a provider registry, allowing the cache backend to be selected via configuration (`cache.type`). Ship two built-in providers: `memory` (in-process LRU) and `redis` (Redis/Valkey-backed LRU).

**Rationale**:

- Decouples the `SubtitleDownloader` from a specific cache implementation
- Enables sharing cache across multiple application instances via Redis/Valkey
- Factory + provider registry pattern makes it easy to add new backends without modifying existing code
- Falls back gracefully to memory if the configured backend fails to initialize

**Redis/Valkey LRU Architecture**:

- Only **2 Redis keys** are used regardless of cache size (not N+1 individual keys):
  - A **Hash** (`sscache:data`) stores all cached values as fields, with per-field TTL via `HPEXPIRE` (Redis 7.4+ / Valkey 8+)
  - A **Sorted Set** (`sscache:lru`) tracks LRU ordering (member = key, score = last-access microsecond timestamp)
- **Atomic Lua scripts** ensure consistency:
  - `getAndTouch`: retrieves a value and refreshes the LRU score in one atomic operation
  - `setAndEvict`: stores a value, sets per-field TTL, updates LRU, and evicts the oldest entries when over capacity
- Expired hash fields are automatically removed by Redis; stale sorted-set members are lazily cleaned during eviction

**Implementation**:

- `internal/cache/cache.go` — `Cache` interface with `Get`, `Set`, `Contains`, `Len`, `Close`
- `internal/cache/factory.go` — Provider registry with `Register`, `New`, `RegisteredProviders`
- `internal/cache/memory.go` — In-memory provider wrapping `hashicorp/golang-lru/v2/expirable`
- `internal/cache/redis.go` — Redis/Valkey provider with Lua scripts for atomic LRU operations
- `internal/services/subtitle_downloader_impl.go` — Uses `cache.Cache` interface; selects backend via `cache.New(cacheType, ...)`
