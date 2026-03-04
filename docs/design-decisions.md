# SuperSubtitles — Design Decisions

This is the index of all architectural and design decisions made in the SuperSubtitles project. Each decision is documented in a focused sub-document below.

## Decision Index

| Document | Decisions Covered |
| --- | --- |
| [Cache](./design-decisions/cache.md) | Cache-layer Prometheus metrics with group label; pluggable cache with factory pattern (memory + Redis/Valkey) |
| [Streaming](./design-decisions/streaming.md) | Server-side streaming RPCs; streaming-first client architecture; `StreamResult[T]` in models package; `ShowSubtitlesCollection` model |
| [HTTP Client](./design-decisions/http-client.md) | HTTP resilience with failsafe-go; partial failure resilience; client architecture; parallel pagination |
| [Parsing](./design-decisions/parsing.md) | Generic parser interfaces; batch processing; parser reusability; data normalization in parser; show name DOM traversal; UTF-8 safety |
| [Infrastructure](./design-decisions/infrastructure.md) | Standard gRPC health checking protocol; error handling strategy |
| [Testing](./design-decisions/testing.md) | Programmatic HTML test fixtures |
