# Design Decisions — Logging

## Automatic Sentry Breadcrumbs and Structured Logs via Zerolog Writer

**Decision**: Forward all zerolog output to Sentry as breadcrumbs and structured log entries through a custom `LevelWriter`, rather than adding Sentry calls at individual log sites.

**Rationale**:

- A single writer adapter captures every log event without requiring manual Sentry calls in application code
- Breadcrumbs give captured Sentry exceptions a chronological trail of recent activity, making errors easier to diagnose
- Sentry's structured Logger API provides searchable logs alongside error events
- Following zerolog's own `ConsoleWriter` pattern (parse JSON, re-emit in a different format) keeps the integration idiomatic
- Error capture remains explicit via `CaptureException` so only actionable failures produce Sentry issues, while the writer provides passive context

**Implementation**: `internal/sentryio/writer.go` defines `SentryWriter`, a `zerolog.LevelWriter` that parses the JSON-serialized log event, records a `sentry.Breadcrumb` on the reporter hub, and emits a structured log via `sentry.NewLogger`. `internal/config/config.go` wraps the base output writer (console or JSON) with `zerolog.MultiLevelWriter` to include the `SentryWriter` when Sentry is enabled. `EnableLogs` in `sentryio.Config` (populated from `sentry.enable_logs` in `config.yaml`, defaulting to `true`) controls whether structured logs are forwarded to Sentry alongside breadcrumbs — set it to `false` to send only breadcrumbs and error events.

## Shared Build Metadata for Startup Logs and Sentry Release Tracking

**Decision**: Keep application build metadata in a shared internal package and use that single source for both startup logging and Sentry release configuration, rather than defining version variables only in `main`.

**Rationale**:

- Sentry is initialized from the config package before `main()` runs, so `main`-local ldflag variables cannot reliably supply release metadata there
- A shared package gives startup logs, Sentry events, and future diagnostics the same version, commit, and build date values
- Default `dev` and `unknown` values keep local development builds readable without requiring GoReleaser or manual linker flags
- Centralizing the linker targets reduces drift between release packaging and runtime observability

**Implementation**: `internal/buildinfo/buildinfo.go` defines the shared `Version`, `Commit`, and `Date` variables. `.goreleaser.yml` populates those symbols via `-X github.com/Belphemur/SuperSubtitles/v2/internal/buildinfo.*`. `cmd/proxy/main.go` logs the build metadata at startup, while `internal/config/config.go` passes `buildinfo.Version` into `sentryio.Config`. `internal/sentryio/reporter.go` maps that value to `sentry.ClientOptions.Release`, and `internal/sentryio/reporter_test.go` verifies the captured event includes the expected release.
