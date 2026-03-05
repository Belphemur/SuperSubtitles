# Design Decisions — HTTP Client

## HTTP Request Resilience with failsafe-go

**Decision**: All HTTP requests made by the client are wrapped with a retry policy using [failsafe-go](https://failsafe-go.dev/). The retry logic is implemented at the transport layer via `failsafehttp.NewRoundTripper`, making it transparent to all call sites.

**Rationale**:

- feliratok.eu is an external dependency that may experience transient outages, rate limiting, or temporary server errors
- Retrying at the transport layer is the least invasive approach — no changes to individual request sites are required
- failsafe-go provides a well-tested, configurable resilience library that handles subtle edge cases (body buffering for re-tries, context cancellation, Retry-After headers, etc.)
- Exponential back-off with a configurable cap prevents thundering-herd scenarios

**Retry behaviour**:

- Retries on connection errors and most 5xx responses (except 501 Not Implemented)
- Retries on 429 Too Many Requests, honouring the `Retry-After` response header when present
- Does **not** retry on 404, 4xx client errors, certificate errors, or unsupported scheme errors
- Context cancellation immediately aborts any pending retry
- A `WARN` log entry is emitted for every retry attempt, including the attempt number and the last HTTP status code

**Configuration** (see `retry.*` config fields):

- `retry.max_attempts` — total attempts including the initial try (default 3, i.e. up to 2 retries)
- `retry.initial_delay` — base delay for exponential back-off (default `"1s"`; set empty to disable back-off)
- `retry.max_delay` — maximum back-off delay cap (default `"10s"`)

**Implementation**: `internal/client/client.go` — `NewClient` builds the retry policy and wraps the compression transport with `failsafehttp.NewRoundTripper` before creating the `http.Client`.

**Partial failure**: The existing partial-failure resilience (returning successful results even when some endpoints fail) is preserved and complementary to retry: retries reduce individual request failures, while partial-failure handling copes with endpoints that remain unavailable after all retries are exhausted.

## Partial Failure Resilience

**Decision**: The client returns whatever data it successfully fetched, logging warnings for failed endpoints rather than failing the entire operation.

**Rationale**:

- feliratok.eu endpoints may be temporarily unavailable
- Users benefit from partial data rather than complete failure
- Warnings in logs allow monitoring of endpoint health

**Implementation**: All parallel fetching operations collect errors but still return successful results if any endpoints succeed.

## Client Architecture

**Decision**: Keep a unified `Client` interface with the implementation split into per-feature files within the `internal/client` package.

**Current State**:

- Single `Client` interface in `client.go` (~140 lines) provides unified API
- Implementation split by feature: `show_list.go`, `subtitles.go`, `show_subtitles.go`, `recent_subtitles.go`, `updates.go`, `download.go`
- Clear method separation with comprehensive per-file tests

**Rationale**:

- Single interface is convenient for consumers
- Per-feature files keep each file focused and testable
- No need for separate client types — the package-level split is sufficient

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
