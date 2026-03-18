# SuperSubtitles — Testing

## Strategy

- **Standard library only** — Go `testing` package + `httptest` for HTTP mocking. No external frameworks
- **Coverage target: 90%+** — all new code must include comprehensive tests
- **Parallel by default** — most tests run concurrently. Exceptions: tests asserting on global Prometheus counters or shared Redis state
- **Integration tests skip in CI** — live requests against feliratok.eu never run in CI

## HTML Fixture Generators

All HTML test data is generated via centralized generators in the testutil package — **never hardcode HTML in tests**. Generators exist for:

- Subtitle tables (with and without pagination)
- Show listings (single-column and multi-column grid)
- Third-party ID detail pages (IMDB/TVDB/TVMaze/Trakt)
- Standalone pagination elements

They use option structs for readable, intent-expressing configuration. If a test needs HTML that no generator supports, add a new generator rather than embedding HTML.

## Binary Archive Fixtures

Real archive compatibility tests may use checked-in binary fixtures when synthetic helpers cannot reproduce the upstream format accurately. This is currently used for RAR season-pack coverage in `.tests-files/`, including RAR→ZIP conversion and episode extraction from converted archives, while ZIP behavior continues to use generated in-memory archives inside tests.

## Stream Collection Helpers

Since the client exposes only streaming methods, the testutil package provides helpers to consume streams into slices for test assertions. These must **never** be used in production code — the gRPC server consumes streams directly.

## Running Tests

```bash
go test -race ./...                                        # All tests (race detector required before commits)
REDIS_ADDRESS=localhost:6379 go test ./internal/cache/...  # With Valkey/Redis (enables cache tests)
go test -coverprofile=coverage.txt -covermode=atomic ./... # With coverage
```

Redis/Valkey cache tests are skipped unless `REDIS_ADDRESS` is set. Requires Valkey 8+ / Redis 7.4+.

## CI Test Matrix

Tests run in parallel groups defined in `.github/workflows/ci.yml`. When adding a new `internal/` package, **add it to an existing CI test group** (or create a new one) so it is included in the matrix. Packages not listed in any group will not run in CI.
