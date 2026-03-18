# Design Decisions — Infrastructure

## Standard gRPC Health Checking Protocol

**Decision**: Implement the standard gRPC health checking protocol rather than a custom health endpoint.

**Rationale**:

- Industry-standard protocol widely supported by infrastructure tools
- Native support in Kubernetes, service meshes, and load balancers
- Works with standard tooling like grpc_health_probe
- No custom client code needed for health checks
- Enables both overall server health and per-service health reporting

**Implementation**: `cmd/proxy/main.go` registers the `grpc.health.v1.Health` service and reports `SERVING` for both the overall server and the SuperSubtitles service. Docker HEALTHCHECK uses `grpc_health_probe` binary downloaded in a separate Dockerfile build stage with SHA256 verification.

## Error Handling Strategy

**Decision**: Use custom error types with error-chain support, wrap errors with context, and prefer partial success over complete failure.

**Rationale**:

- Error-chain support enables proper error checking with standard Go patterns
- Wrapped errors preserve context
- Partial success maximizes data availability
- Logged warnings enable monitoring

**Implementation**: `ErrNotFound`, `ErrSubtitleNotFoundInArchive`, `ErrSubtitleResourceNotFound`, and `ArchiveError` in `internal/apperrors/errors.go`, each with `Is()` support and gRPC/HTTP binding metadata via `GRPCBindableError`. `internal/grpc/error_mapping.go` performs centralized translation from application errors to gRPC statuses, including `ErrorInfo` metadata for equivalent HTTP statuses (for example, archive-processing failures map to `codes.FailedPrecondition` with `http_status=422`).

## Archive Handling For Season Packs

**Decision**: Always normalize RAR archives to ZIP before any processing — both whole-archive downloads and episode extraction operate exclusively on ZIP data.

**Rationale**:

- A single archive format simplifies all downstream logic: one extraction path, one bomb-detection path, one caching representation
- feliratok.eu may serve season packs as either ZIP or RAR for the same download workflow; normalizing early eliminates format-specific branching
- ZIP is the safer target format: Go's `archive/zip` is well-tested and supports random access, while RAR handling requires a third-party library with less ecosystem maturity
- Converting at download time means cached data is always ZIP, so subsequent cache hits never need RAR handling

**Implementation**: `internal/services/subtitle_downloader_impl.go` detects archive type via `archive.DetectFormat()` (magic bytes + MIME). Both whole-download and episode-download paths convert RAR→ZIP through `archive.ConvertRarToZip()` before caching. Episode extraction then calls `archive.ExtractEpisodeFromZip()`. All archive logic lives in the `internal/archive` package — see [archive decisions](archive.md) for details.

## Optional Sentry Error Reporting

**Decision**: Keep Sentry reporting optional and only capture top-level server exceptions, excluding expected “episode not found in archive” failures.

**Rationale**:

- Sentry should provide stack-bearing exception events without becoming a hard runtime dependency
- Top-level capture avoids duplicating every internal log line while still surfacing request failures and fatal server errors
- Expected archive miss cases are part of normal subtitle lookup behavior and would create noise in error reporting

**Implementation**: `internal/config/config.go` maps optional `sentry.*` settings and initializes the official `github.com/getsentry/sentry-go` SDK when a DSN is configured. `internal/sentryio/reporter.go` owns filtering and flushing. `internal/grpc/server.go` reports request-level failures with gRPC method/request context, while `cmd/proxy/main.go` reports fatal startup and serve errors before process exit. Log-level Sentry integration (breadcrumbs and structured logs) is covered in the [logging design decisions](logging.md).
