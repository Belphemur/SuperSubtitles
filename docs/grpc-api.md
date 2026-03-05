# SuperSubtitles — gRPC API

Proto source: [`api/proto/v1/supersubtitles.proto`](../api/proto/v1/supersubtitles.proto). Regenerate with `go generate ./api/proto/v1`.

## Service

```protobuf
service SuperSubtitlesService {
  rpc GetShowList(GetShowListRequest) returns (stream Show);
  rpc GetSubtitles(GetSubtitlesRequest) returns (stream Subtitle);
  rpc GetShowSubtitles(GetShowSubtitlesRequest) returns (stream ShowSubtitlesCollection);
  rpc CheckForUpdates(CheckForUpdatesRequest) returns (CheckForUpdatesResponse);
  rpc DownloadSubtitle(DownloadSubtitleRequest) returns (DownloadSubtitleResponse);
  rpc GetRecentSubtitles(GetRecentSubtitlesRequest) returns (stream ShowSubtitlesCollection);
}
```

Four of six RPCs use **server-side streaming** (see [streaming decisions](./design-decisions/streaming.md)). The server also implements `grpc.health.v1.Health` for standard health checking.

## Endpoints

| RPC | Type | Request | Response | Description |
| --- | --- | --- | --- | --- |
| `GetShowList` | streaming | empty | `stream Show` | All available TV shows from 3 parallel endpoints |
| `GetSubtitles` | streaming | `show_id` | `stream Subtitle` | Subtitles for a show (auto-paginated) |
| `GetShowSubtitles` | streaming | `repeated Show` | `stream ShowSubtitlesCollection` | Shows + subtitles + third-party IDs |
| `GetRecentSubtitles` | streaming | `since_id` | `stream ShowSubtitlesCollection` | Recent uploads since a subtitle ID |
| `CheckForUpdates` | unary | `content_id` | `CheckForUpdatesResponse` | New subtitle counts since content ID |
| `DownloadSubtitle` | unary | `subtitle_id`, `episode` | filename + content + MIME type | Download file, optionally extract episode from ZIP |

## grpcurl Examples

```bash
# List shows
grpcurl -plaintext localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetShowList

# Get subtitles for a show
grpcurl -plaintext -d '{"show_id": 1234}' localhost:8080 supersubtitles.v1.SuperSubtitlesService/GetSubtitles

# Download a specific episode from a season pack
grpcurl -plaintext -d '{"subtitle_id": "101", "episode": 1}' localhost:8080 supersubtitles.v1.SuperSubtitlesService/DownloadSubtitle

# Health check
grpc_health_probe -addr=localhost:8080
```

## Error Codes

| Code | When |
| --- | --- |
| `NOT_FOUND` | Episode missing from ZIP, subtitle URL 404, show ID not found |
| `INVALID_ARGUMENT` | No valid shows provided to `GetShowSubtitles` |
| `INTERNAL` | HTTP failures, parsing errors |

## Server Setup

`internal/grpc/setup.go` → `NewGRPCServer()` creates the server with Prometheus interceptors, health checking, and gRPC reflection. Entry point `cmd/proxy/main.go` starts the server and optional metrics HTTP endpoint, with graceful shutdown on SIGTERM/SIGINT.
