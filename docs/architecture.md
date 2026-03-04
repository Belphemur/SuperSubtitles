# SuperSubtitles — Architecture Documentation

## Documentation

| Document | Description |
| --- | --- |
| [Overview](./overview.md) | High-level architecture diagram and component relationships |
| [gRPC API](./grpc-api.md) | All RPC endpoints, proto definitions, data models, and usage examples |
| [Data Flow](./data-flow.md) | Detailed operation flows (show list, subtitles, recent, download) |
| [Testing](./testing.md) | Test infrastructure, fixture generators, and coverage |
| [Configuration](./configuration.md) | Configuration reference and environment variables |
| [CI/CD](./ci-cd.md) | CI/CD pipeline, dependencies, and local development setup |
| [Deployment](./deployment.md) | Docker, Kubernetes deployment, and monitoring |
| [Project Structure](./project_structure.md) | Directory layout and file relationships |

## Design Decisions

| Document | Decisions Covered |
| --- | --- |
| [Design Decisions Index](./design-decisions.md) | Overview and index of all architectural decisions |
| [Cache](./design-decisions/cache.md) | Cache-layer Prometheus metrics; pluggable cache with factory pattern |
| [Streaming](./design-decisions/streaming.md) | Server-side streaming RPCs; streaming-first client; `StreamResult[T]`; `ShowSubtitlesCollection` |
| [HTTP Client](./design-decisions/http-client.md) | HTTP resilience; partial failure; client architecture; parallel pagination |
| [Parsing](./design-decisions/parsing.md) | Generic parser interfaces; batch processing; data normalization; DOM traversal; UTF-8 safety |
| [Infrastructure](./design-decisions/infrastructure.md) | Standard gRPC health checking; error handling strategy |
| [Testing](./design-decisions/testing.md) | Programmatic HTML test fixtures |
