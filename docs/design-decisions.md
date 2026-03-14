# SuperSubtitles — Design Decisions

Each decision follows the [Decision → Rationale → Implementation template](./design-decisions/TEMPLATE.md).

## Decision Index

| Document | Decisions Covered |
| --- | --- |
| [Cache](./design-decisions/cache.md) | Cache-layer metrics with group label; pluggable cache with factory pattern |
| [Streaming](./design-decisions/streaming.md) | Server-side streaming RPCs; streaming-first client; stream result in models; show+subtitles bundle |
| [HTTP Client](./design-decisions/http-client.md) | HTTP resilience with failsafe-go; partial failure; client architecture; parallel pagination |
| [Parsing](./design-decisions/parsing.md) | Generic parser interfaces; batch processing; parser reusability; normalization in parser; DOM traversal; UTF-8 safety |
| [Infrastructure](./design-decisions/infrastructure.md) | Standard gRPC health checking; error handling strategy |
| [Logging](./design-decisions/logging.md) | Automatic Sentry breadcrumbs and structured logs via zerolog writer |
| [Testing](./design-decisions/testing.md) | Programmatic HTML test fixtures |
