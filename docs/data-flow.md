# SuperSubtitles — Data Flow

## Show List

1. Fires 3 parallel HTTP requests to different feliratok.eu endpoints
2. Fetches page 1 of each endpoint, parses HTML to extract shows and discover total pages
3. Remaining pages fetched in **parallel batches of 10**
4. Results deduplicated by show ID
5. Each show streamed to gRPC clients as it arrives
6. Partial failures tolerated: individual endpoint/page failures log warnings but don't fail the operation

## Subtitles

1. Fetches first subtitle page for a show
2. Parses 6-column HTML table with normalization (ISO language codes, qualities, season/episode, release groups, season pack detection)
3. If multiple pages exist, remaining fetched in **parallel pairs** (2 at a time)
4. Subtitles streamed as pages complete

## Show Subtitles with Third-Party IDs

1. Processes shows in **batches of 20**
2. For each show: collects all subtitles, then loads the detail page
3. Extracts IMDB/TVDB/TVMaze/Trakt IDs from detail page links
4. Streams a complete bundle (show info + IDs + all subtitles) per show

## Recent Subtitles

1. Fetches main page (same HTML table structure as individual show pages)
2. Filters by since-ID — only subtitles newer than the given ID
3. Groups by show, fetches detail pages for third-party IDs
4. Streams a bundle per show (with partial failure resilience for detail pages)

## Subtitle Download

1. Client builds download URL and delegates to the download service
2. **Regular files** (SRT, ASS, VTT, SUB): downloaded, converted to UTF-8, returned with correct MIME type
3. **ZIP without episode**: returned as-is
4. **RAR without episode**: normalized to ZIP, then returned with ZIP MIME metadata
5. **Season pack with episode number**: ZIP archives are searched directly; RAR archives are searched directly without conversion, using pattern matching (S03E01, 3x01, E01 and case variants)
6. **Cache**: pluggable LRU (memory or Redis/Valkey) keeps whole-download ZIP normalizations separate from episode-extraction archive bytes so converting a RAR download does not change later episode extraction behavior
