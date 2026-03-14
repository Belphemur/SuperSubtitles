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

**Decision**: Normalize RAR archives to ZIP only for whole-archive downloads, while episode extraction reads ZIP and RAR season packs in their original format.

**Rationale**:

- Whole-archive downloads benefit from a single returned archive format when the upstream file is RAR
- feliratok.eu may serve season packs as either ZIP or RAR for the same download workflow
- Episode extraction should not depend on a prior whole-archive download choosing a converted representation
- Separate cache entries keep no-episode normalization from altering later episode lookups on the same upstream URL

**Implementation**: `internal/services/subtitle_downloader_impl.go` detects archive type from both MIME metadata and file signatures. Whole-download requests use a normalized-download cache entry that stores ZIP files as-is and stores RAR files after conversion through `github.com/nwaples/rardecode/v2`. Episode requests use a separate cache entry for the original archive bytes, then extract from ZIP via the existing ZIP flow or from RAR via direct RAR traversal and filename matching.

## Optional Sentry Error Reporting

**Decision**: Keep Sentry reporting optional and only capture top-level server exceptions, excluding expected “episode not found in archive” failures.

**Rationale**:

- Sentry should provide stack-bearing exception events without becoming a hard runtime dependency
- Top-level capture avoids duplicating every internal log line while still surfacing request failures and fatal server errors
- Expected archive miss cases are part of normal subtitle lookup behavior and would create noise in error reporting

**Implementation**: `internal/config/config.go` maps optional `sentry.*` settings and initializes the official `github.com/getsentry/sentry-go` SDK when a DSN is configured. `internal/sentryio/reporter.go` owns filtering and flushing. `internal/grpc/server.go` reports request-level failures with gRPC method/request context, while `cmd/proxy/main.go` reports fatal startup and serve errors before process exit. Log-level Sentry integration (breadcrumbs and structured logs) is covered in the [logging design decisions](logging.md).
