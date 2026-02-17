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

**Decision**: Fetch subtitle pages in parallel batches of 2 rather than sequentially.

**Rationale**:

- ~3x faster for shows with many subtitle pages
- Balances speed with server load
- Batch size of 2 is conservative and respectful

**Implementation**: Pages 2-3 fetched together, then 4-5, etc. First page always fetched alone to determine total pages.

## Error Handling Strategy

**Decision**: Use custom error types with `Is()` method support, wrap errors with `fmt.Errorf`, and prefer partial success over complete failure.

**Rationale**:

- `errors.Is()` support enables proper error checking
- Wrapped errors preserve context
- Partial success maximizes data availability
- Logged warnings enable monitoring

**Implementation**:

- `ErrNotFound` custom error in `internal/client/errors.go`
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
- `ShowSubtitleItem` uses protobuf `oneof` to interleave show metadata and subtitles in a single stream
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

## ShowSubtitleItem Streaming Model

**Decision**: Use a `oneof`-based `ShowSubtitleItem` message to interleave show info and subtitles in `GetShowSubtitles` and `GetRecentSubtitles` streams. ShowInfo is sent once per unique `show_id` within a single call, followed by individual subtitles for that show.

**Rationale**:

- Allows streaming show metadata and subtitles through a single stream without separate RPCs
- `ShowInfo` (show + third-party IDs) is sent once per show, followed by its subtitles
- Consumers link subtitles to shows via the `show_id` field on each `Subtitle`
- More efficient than sending complete `ShowSubtitles` objects that bundle all subtitles together
- For `GetRecentSubtitles`, show info includes third-party IDs fetched from detail pages, enabling clients to discover new shows they haven't seen before
- Internal `ShowInfo` and `ShowSubtitleItem` models in `internal/models/show_subtitles.go` mirror the proto structure

**Implementation**:

- Proto `ShowSubtitleItem.oneof item` contains either `ShowInfo` or `Subtitle`
- Internal `ShowSubtitleItem` struct uses pointer fields (`*ShowInfo`, `*Subtitle`) — exactly one is non-nil
- `convertShowSubtitleItemToProto` converter handles the oneof mapping

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
- Docker HEALTHCHECK uses `grpc_health_probe` binary (downloaded in separate build stage)
- Multi-stage Dockerfile: download stage fetches `grpc_health_probe`, runtime stage only includes the binary
- Health check runs every 30s with 10s timeout, 5s start period, 3 retries
- Final image excludes build tools (wget) for minimal size
