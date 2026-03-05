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

**Implementation**: `ErrNotFound`, `ErrSubtitleNotFoundInZip`, and `ErrSubtitleResourceNotFound` in `internal/apperrors/errors.go`, each with `Is()` support. gRPC server in `internal/grpc/server.go` maps all three to `codes.NotFound`.
