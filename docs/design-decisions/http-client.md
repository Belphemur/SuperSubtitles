# Design Decisions — HTTP Client

## HTTP Request Resilience with failsafe-go

**Decision**: All HTTP requests are wrapped with a retry policy using [failsafe-go](https://failsafe-go.dev/). The retry logic is implemented at the transport layer, making it transparent to all call sites.

**Rationale**:

- feliratok.eu is an external dependency that may experience transient outages, rate limiting, or temporary server errors
- Retrying at the transport layer is the least invasive approach — no changes to individual request sites are required
- failsafe-go handles subtle edge cases (body buffering for retries, context cancellation, Retry-After headers, etc.)
- Exponential back-off with a configurable cap prevents thundering-herd scenarios

**Retry behaviour**:

- Retries on connection errors and most 5xx responses (except 501 Not Implemented)
- Retries on 429 Too Many Requests, honouring the Retry-After response header when present
- Does **not** retry on 404, 4xx client errors, certificate errors, or unsupported scheme errors
- Context cancellation immediately aborts any pending retry
- A warning log entry is emitted for every retry attempt

**Configuration**: See `retry.*` fields in [configuration](../configuration.md).

**Implementation**: `NewClient` in `internal/client/client.go` builds the retry policy via `failsafehttp.NewRetryPolicyBuilder()` and wraps the compression transport with `failsafehttp.NewRoundTripper`.

## Partial Failure Resilience

**Decision**: The client returns whatever data it successfully fetched, logging warnings for failed endpoints rather than failing the entire operation.

**Rationale**:

- feliratok.eu endpoints may be temporarily unavailable
- Users benefit from partial data rather than complete failure
- Warnings in logs allow monitoring of endpoint health

Retries and partial failure are complementary: retries reduce individual request failures, while partial failure handling copes with endpoints that remain unavailable after all retries are exhausted.

**Implementation**: All parallel fetching operations in `internal/client/` collect errors but still return successful results if any endpoints succeed.

## Client Architecture

**Decision**: Keep a unified Client interface with the implementation split into per-feature files within the client package.

**Rationale**:

- Single interface is convenient for consumers
- Per-feature files keep each file focused and testable
- No need for separate client types — the package-level split is sufficient

**Implementation**: `Client` interface in `internal/client/client.go`. Implementation split by feature: `show_list.go`, `subtitles.go`, `show_subtitles.go`, `recent_subtitles.go`, `updates.go`, `download.go`.

## Parallel Pagination

**Decision**: Fetch paginated content in parallel batches rather than sequentially. Batch sizes differ by context:

- **Subtitle pages**: Batch size of 2
- **Show list pages**: Batch size of 10 (show list endpoints can have 40+ pages)

**Rationale**:

- Dramatically faster for endpoints with many pages
- Balances speed with server load
- First page always fetched alone to discover total page count
- Show list uses a larger batch size because individual pages are lightweight

**Implementation**: Subtitles fetched in pairs via `internal/client/subtitles.go`. Show lists fetched in batches of 10 via `internal/client/show_list.go`; `ShowParser.ExtractLastPage` parses pagination links to discover the total page count.
