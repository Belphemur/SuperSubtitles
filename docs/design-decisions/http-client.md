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

## Circuit Breaker for HTTP Calls

**Decision**: All HTTP requests to feliratok.eu are additionally protected by a [failsafe-go](https://failsafe-go.dev/) circuit breaker, composed around the retry policy at the transport level. The circuit breaker opens after a run of consecutive transient failures (5xx responses, 429, or connection errors — the same class of failures the retry policy handles), short-circuiting further requests for a configurable delay before allowing trial requests through again (half-open) and eventually closing once those trials succeed.

**Rationale**:

- Retries alone can amplify load on an already-struggling upstream (every caller keeps retrying, multiplying request volume during an outage)
- A circuit breaker gives feliratok.eu a chance to recover by failing fast instead of continuing to hammer it once failures are persistent
- Failing fast also improves latency for callers during an outage — they get an immediate, clear error instead of waiting through a full retry sequence on every call
- Composing the circuit breaker *around* the retry policy (rather than *inside* it) means the breaker counts failures per logical request (after retries are exhausted), not per individual HTTP attempt — this avoids opening the circuit prematurely due to a single request's retries

**Behaviour**:

- Opens after `circuit_breaker.failure_threshold` consecutive failures (same failure classification as the retry policy: 5xx except 501, 429, connection errors)
- Stays open for `circuit_breaker.open_duration`, rejecting all requests immediately with an error wrapping `circuitbreaker.ErrOpen`
- Transitions to half-open afterward, allowing `circuit_breaker.success_threshold` consecutive trial requests through; success closes the circuit again, failure re-opens it
- State transitions (open/half-open/closed) are logged and exposed via the `http_client_circuit_breaker_state` Prometheus gauge (`0`=closed, `1`=half-open, `2`=open)
- The gRPC layer maps a circuit-breaker-open error to `codes.Unavailable` (HTTP 503 equivalent) with a clear, human-readable message via `apperrors.ErrCircuitOpen`, instead of a generic `Internal` error — see [gRPC error codes](../grpc-api.md#error-codes)

**Configuration**: See `circuit_breaker.*` fields in [configuration](../configuration.md).

**Implementation**: `newCircuitBreakerPolicy` in `internal/client/circuit_breaker.go` builds the breaker via `circuitbreaker.NewBuilder()`; `NewClient` composes it with the retry policy via `failsafehttp.NewRoundTripper(transport, circuitBreakerPolicy, retryPolicy)`. Error translation lives in `internal/grpc/error_mapping.go` (`toStatusError`) and `internal/apperrors/errors.go` (`ErrCircuitOpen`).

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
