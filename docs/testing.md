# SuperSubtitles — Testing

## Strategy

- **Standard library only** — Go `testing` package + `httptest` for HTTP mocking. No external frameworks
- **Coverage target: 90%+** — all new code must include comprehensive tests
- **Parallel by default** — most tests call `t.Parallel()`. Exceptions: tests asserting on global Prometheus counters or shared Redis DB 15
- **Integration tests skip in CI** — `client_integration_test.go` checks `CI` / `SKIP_INTEGRATION_TESTS` env vars; live requests never run in CI

## HTML Fixture Generators

All HTML test data is generated via `internal/testutil/html_fixtures.go` — **never hardcode HTML in tests**. If a test needs HTML that no generator supports, add a new generator to `html_fixtures.go`.

| Generator | Purpose |
| --- | --- |
| `GenerateSubtitleTableHTML` | Subtitle listing tables |
| `GenerateSubtitleTableHTMLWithPagination` | Subtitle tables with pagination |
| `GenerateShowTableHTML` | Show listings |
| `GenerateShowTableHTMLMultiColumn` | Multi-column show grid layouts |
| `GenerateThirdPartyIDHTML` | Detail pages with IMDB/TVDB/TVMaze/Trakt |
| `GeneratePaginationHTML` | Standalone pagination elements |

Fixtures use option structs (`SubtitleRowOptions`, `ShowRowOptions`) for readable, intent-expressing configuration. Changing the HTML structure requires updating one generator instead of dozens of test strings.

## Stream Collection Helpers

Since the client exposes only streaming methods, `internal/testutil/stream_helpers.go` provides test-only helpers:

- `CollectShows` — consumes a Show stream → slice
- `CollectSubtitles` — consumes a Subtitle stream → SubtitleCollection
- `CollectShowSubtitles` — consumes a ShowSubtitles stream → slice

These must **never** be used in production code — the gRPC server consumes streams directly.

## Running Tests

```bash
go test -race ./...                                     # All tests (race detector required before commits)
go test ./internal/parser/...                           # Specific package
REDIS_ADDRESS=localhost:6379 go test ./internal/cache/...  # With Valkey/Redis (enables cache tests)
go test -coverprofile=coverage.txt -covermode=atomic ./... # With coverage
```

Redis/Valkey cache tests are skipped unless `REDIS_ADDRESS` is set. Requires Valkey 8+ / Redis 7.4+ for `HPEXPIRE`.
