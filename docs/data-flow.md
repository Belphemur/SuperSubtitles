# SuperSubtitles — Data Flow

## Show List (`StreamShowList`)

1. Fires 3 parallel HTTP requests to different feliratok.eu endpoints
2. Fetches page 1 of each endpoint, parses HTML via `ShowParser` to extract shows and discover total pages
3. Remaining pages fetched in **parallel batches of 10**
4. Results deduplicated by show ID using `sync.Map`
5. Each show streamed to channel → gRPC server sends `Show` messages as they arrive
6. Partial failures tolerated: individual endpoint/page failures log warnings but don't fail the operation

**Files:** `internal/client/show_list.go`, `internal/parser/show_parser.go`

## Subtitles (`StreamSubtitles`)

1. Fetches first page: `GET /index.php?sid=<showID>`
2. `SubtitleParser` extracts subtitles from 6-column HTML table (language, description, uploader, date, download) with normalization (ISO language codes, qualities, season/episode, release groups, season pack detection)
3. If multiple pages exist, remaining fetched in **parallel pairs** (pages 2–3, then 4–5, etc.)
4. Subtitles streamed to channel as pages complete

**Files:** `internal/client/subtitles.go`, `internal/parser/subtitle_parser.go`

## Show Subtitles with Third-Party IDs (`StreamShowSubtitles`)

1. Processes shows in **batches of 20**
2. For each show: collects all subtitles via `StreamSubtitles`, then loads the detail page HTML
3. `ThirdPartyIdParser` extracts IMDB/TVDB/TVMaze/Trakt IDs from detail page links
4. Streams a complete `ShowSubtitles` (show info + IDs + all subtitles) per show

**Files:** `internal/client/show_subtitles.go`, `internal/parser/third_party_parser.go`

## Recent Subtitles (`StreamRecentSubtitles`)

1. Fetches main page: `GET /index.php?tab=sorozat`
2. Parses subtitles (same parser as individual show pages)
3. Filters by `sinceID` — only subtitles with `ID > sinceID`
4. Groups by show, fetches detail pages for third-party IDs
5. Streams `ShowSubtitles` per show (with partial failure resilience for detail pages)

**Files:** `internal/client/recent_subtitles.go`

## Subtitle Download (`DownloadSubtitle`)

1. Client builds download URL and delegates to `SubtitleDownloader`
2. **Regular files** (SRT, ASS, VTT, SUB): downloaded, converted to UTF-8, returned with correct MIME type
3. **ZIP without episode**: returned as-is
4. **ZIP with episode number**: ZIP downloaded (or from cache), episode extracted by pattern matching (`S03E01`, `3x01`, `E01`), returned with correct type
5. **Cache**: pluggable LRU (memory or Redis/Valkey) caches ZIP files only. Multiple episode requests from the same season pack reuse the cached ZIP

**Files:** `internal/services/subtitle_downloader_impl.go`, `internal/client/download.go`
