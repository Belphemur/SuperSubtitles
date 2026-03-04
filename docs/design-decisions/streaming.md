# Design Decisions — Streaming

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
