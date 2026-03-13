# SuperSubtitles — gRPC API

Proto source: [`api/proto/v1/supersubtitles.proto`](../api/proto/v1/supersubtitles.proto). Regenerate with `go generate ./api/proto/v1`.

## Endpoints

| RPC | Type | Request | Response | Description |
| --- | --- | --- | --- | --- |
| GetShowList | streaming | empty | stream of shows | All available TV shows from 3 parallel endpoints |
| GetSubtitles | streaming | show ID | stream of subtitles | Subtitles for a show (auto-paginated) |
| GetShowSubtitles | streaming | list of shows | stream of show+subtitles bundles | Shows with subtitles and third-party IDs |
| GetRecentSubtitles | streaming | since ID | stream of show+subtitles bundles | Recent uploads since a subtitle ID |
| CheckForUpdates | unary | content ID | update counts | New subtitle counts since content ID |
| DownloadSubtitle | unary | subtitle ID, episode | file content + MIME type | Download file, optionally extract episode from ZIP |

Four of six RPCs use **server-side streaming** (see [streaming decisions](./design-decisions/streaming.md)). The server also implements the standard gRPC health checking protocol.

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
| NOT_FOUND | Episode missing from ZIP, subtitle URL 404, show ID not found |
| INVALID_ARGUMENT | No valid shows provided |
| FAILED_PRECONDITION | Archive validation/conversion/extraction failures; includes `ErrorInfo` metadata `http_status=422` (`UNPROCESSABLE_ENTITY`) |
| INTERNAL | HTTP failures, parsing errors |
