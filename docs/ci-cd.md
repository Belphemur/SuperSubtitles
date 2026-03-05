# SuperSubtitles — CI/CD

## Workflows

### CI (`.github/workflows/ci.yml`) — every push/PR to `main`

Five jobs:

1. **Lint** — `go mod verify`, `go vet`, `gofmt`, `golangci-lint run`
2. **Test** — 3-group matrix (`parser-models-errors`, `client`, `services-grpc-metrics`) without service containers. Runs with `-race` flag
3. **Test-cache** — separate job for `cache` package with a Valkey service container and `REDIS_ADDRESS` set. Runs with `-race` flag
4. **Report** — collects all test artifacts and uploads coverage + test results to Codecov
5. **Build** — `CGO_ENABLED=0 go build` producing a static binary

### Release (`.github/workflows/release.yml`) — push to `main`

1. **semantic-release** analyzes conventional commits → determines version
2. **GoReleaser** builds cross-platform binaries (linux/amd64, linux/arm64)
3. Multi-platform **Docker images** pushed to `ghcr.io/belphemur/supersubtitles`
4. **GitHub Release** with changelog, SBOMs, and attestation

### Copilot Setup (`.github/workflows/copilot-setup-steps.yml`)

Prepares Copilot agent environment: Go 1.26, gopls, golangci-lint, dependency download.

## Local Development

```bash
go mod download          # Install dependencies
go test -race ./...      # Run tests with race detector
go build ./cmd/proxy     # Build binary
golangci-lint run        # Lint
```

Prerequisites: Go 1.26+, golangci-lint. Optional: gotestsum (prettier test output), Valkey/Redis for cache tests (`REDIS_ADDRESS=localhost:6379`).
