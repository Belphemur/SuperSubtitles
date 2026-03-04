# Design Decisions — Infrastructure

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
- Docker HEALTHCHECK uses `grpc_health_probe` binary (downloaded in separate build stage with SHA256 verification)
- Multi-stage Dockerfile: download stage fetches `grpc_health_probe` and verifies checksum against official release checksums
- Health check runs every 30s with 10s timeout, 5s start period, 3 retries
- Final image excludes build tools (wget) for minimal size

## Error Handling Strategy

**Decision**: Use custom error types with `Is()` method support, wrap errors with `fmt.Errorf`, and prefer partial success over complete failure.

**Rationale**:

- `errors.Is()` support enables proper error checking
- Wrapped errors preserve context
- Partial success maximizes data availability
- Logged warnings enable monitoring

**Implementation**:

- `ErrNotFound`, `ErrSubtitleNotFoundInZip`, and `ErrSubtitleResourceNotFound` custom error types in `internal/apperrors/errors.go`, each with `Is()` support
- All errors wrapped with context
- Parallel operations collect errors but return successful results
