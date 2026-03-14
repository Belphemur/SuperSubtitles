# Design Decisions — Logging

## Automatic Sentry Breadcrumbs and Structured Logs via Zerolog Writer

**Decision**: Forward all zerolog output to Sentry as breadcrumbs and structured log entries through a custom `LevelWriter`, rather than adding Sentry calls at individual log sites.

**Rationale**:

- A single writer adapter captures every log event without requiring manual Sentry calls in application code
- Breadcrumbs give captured Sentry exceptions a chronological trail of recent activity, making errors easier to diagnose
- Sentry's structured Logger API provides searchable logs alongside error events
- Following zerolog's own `ConsoleWriter` pattern (parse JSON, re-emit in a different format) keeps the integration idiomatic
- Error capture remains explicit via `CaptureException` so only actionable failures produce Sentry issues, while the writer provides passive context

**Implementation**: `internal/sentryio/writer.go` defines `SentryWriter`, a `zerolog.LevelWriter` that parses the JSON-serialized log event, records a `sentry.Breadcrumb` on the reporter hub, and emits a structured log via `sentry.NewLogger`. `internal/config/config.go` wraps the base output writer (console or JSON) with `zerolog.MultiLevelWriter` to include the `SentryWriter` when Sentry is enabled. `EnableLogs: true` is set in the Sentry client options so structured logs are forwarded alongside breadcrumbs.
