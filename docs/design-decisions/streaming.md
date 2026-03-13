# Design Decisions — Streaming

## Server-Side Streaming RPCs

**Decision**: Use server-side streaming for 4 of 6 gRPC RPCs (show list, subtitles, show subtitles, recent subtitles). Only update checks and subtitle downloads remain unary.

**Rationale**:

- Improves time-to-first-result by sending items as they become available
- Reduces server memory usage — no need to buffer entire response collections
- Natural fit for list/collection endpoints that aggregate data from multiple sources
- Enables progressive rendering on the client side
- Unary RPCs are kept for single-value responses (update check, file download)

**Implementation**: Proto definitions use `returns (stream T)` for streaming RPCs. gRPC server methods in `internal/grpc/server.go` consume from client streaming channels and call `stream.Send()` per item.

## Streaming-First Client Architecture

**Decision**: The client exposes **only** streaming methods for list/collection operations. Non-streaming methods have been removed. Test helpers provide collection utilities for tests.

**Rationale**:

- Channels are Go's natural primitive for streaming data
- Eliminates dual API surface (streaming + non-streaming)
- Forces consumers to handle streaming properly from the start
- Reduces memory usage — no intermediate buffering of full collections
- Simplifies client interface and reduces code duplication
- Production gRPC server consumes streams directly without buffering

**Implementation**: `StreamResult[T]` generic struct in `internal/models/stream_result.go`. All streaming methods return read-only `<-chan models.StreamResult[T]` channels. `internal/testutil/stream_helpers.go` provides test-only collection helpers (`CollectShows`, `CollectSubtitles`, `CollectShowSubtitles`).

## Stream Result in Models Package

**Decision**: The generic stream result type is defined in the models package rather than the client package.

**Rationale**:

- Avoids circular dependency: test utilities need to reference both stream results and models
- Makes it a core domain type alongside other model types
- Allows other packages to use it without depending on the client

## Show+Subtitles Streaming Bundle

**Decision**: Stream complete show data (show info + all subtitles) as a single message per show, rather than interleaving individual items.

**Rationale**:

- Simplifies client consumption — each message is self-contained
- No need for clients to reconstruct show-subtitle groupings from interleaved messages
- Show info includes third-party IDs fetched from detail pages, enabling clients to discover new shows
- Client accumulates subtitles per show before sending, trading slightly more memory for simpler semantics

**Implementation**: Proto `ShowSubtitlesCollection` contains `ShowInfo show_info` and `repeated Subtitle subtitles`. Internal `ShowSubtitles` model in `internal/models/show_subtitles.go` maps directly to the proto structure. `convertShowSubtitlesToProto` in `internal/grpc/converters.go` handles the conversion.

## Sequential Pagination for Recent Subtitles

**Decision**: When `sinceID > 0`, `StreamRecentSubtitles` fetches pages sequentially using the `oldal=` parameter until a subtitle with ID ≤ sinceID is found. When `sinceID == 0`, only the first page is fetched.

**Rationale**:

- The homepage lists subtitles in reverse chronological order across 1000+ pages — fetching only page 1 misses subtitles uploaded since the last poll if more than one page of new content appeared
- Sequential fetching (not parallel) is correct here because we don't know the total number of needed pages upfront — we stop as soon as we hit the boundary
- `sinceID == 0` remains single-page to avoid accidentally crawling the entire site on the first call
- Reuses `ParseHtmlWithPagination` already used by `StreamSubtitles`, keeping the parser surface consistent

**Implementation**: `StreamRecentSubtitles` in `internal/client/recent_subtitles.go` loops page-by-page, calling `SubtitleParser.ParseHtmlWithPagination` on each response. An `addSubtitle` closure accumulates results and signals when the sinceID boundary is reached. The loop also respects `HasNextPage` from the parser to stop at the last page.
