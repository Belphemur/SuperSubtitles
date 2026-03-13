# Design Decisions — Parsing

## Generic Parser Interfaces

**Decision**: Use generic interfaces for type-safe HTML parsing.

**Rationale**:

- Type safety prevents runtime errors
- Reusable pattern for different model types
- Clear contract for parser implementations

**Implementation**: `internal/parser/interfaces.go` defines `Parser[T]` and `SingleResultParser[T]` generic interfaces implemented by `ShowParser`, `SubtitleParser`, and `ThirdPartyIdParser`.

## Batch Processing

**Decision**: Show subtitle fetching is batched (20 concurrent requests) to avoid overwhelming the upstream server.

**Rationale**:

- Prevents overloading feliratok.eu infrastructure
- Balances speed with responsible resource usage
- Allows rate limiting if needed in the future

**Implementation**: `StreamShowSubtitles` in `internal/client/show_subtitles.go` processes shows in batches of 20 with parallel processing within each batch.

## Parser Reusability

**Decision**: The same subtitle parser is used for both individual show pages and the main page.

**Rationale**:

- Both pages share the same HTML table structure
- Reduces code duplication
- Single source of truth for subtitle parsing logic
- Only requires one additional method for main page support (extracting show ID from the category column)

**Implementation**: `SubtitleParser.ParseHtml` in `internal/parser/subtitle_parser.go` works for both page types. `extractShowIDFromCategory` method extracts show ID from the category column when available.

## Parser Handles All Data Normalization

**Decision**: The subtitle parser directly handles language conversion, quality extraction, season/episode parsing, and data normalization rather than using a separate converter service.

**Rationale**:

- Parsing and normalization are tightly coupled operations
- Eliminates unnecessary intermediate data structures
- Reduces code complexity and maintenance burden
- Parser has all HTML context needed for normalization
- Single responsibility: transform HTML → normalized models

**Implementation**: `SubtitleParser` in `internal/parser/subtitle_parser.go` includes `convertLanguageToISO` (Hungarian → ISO 639-1), `parseReleaseInfo` (quality and release groups), `parseDescription` (season/episode/show name), and `detectQuality` (quality enum). Season-pack detection relies exclusively on archive-type download filenames (`.zip`/`.rar`). Title parsing still extracts season-level metadata such as `(Season 2)` or ranged notation like `1x01-09`, but those patterns do not classify an entry as a season pack unless the download file is an archive. When valid archive-backed ranged notation is detected, range bounds are normalized and stored as optional subtitle metadata exposed through gRPC fields. All normalization happens during HTML parsing in one pass.

## Show Name Extraction via DOM Traversal

**Decision**: Use direct DOM sibling traversal to find show names instead of iterating through all table cells with string matching.

**Problem**: The original implementation used a complex flag-based iteration pattern, searching for link hrefs across all cells. This was fragile and prone to incorrect matches in multi-column layouts.

**Solution**: Navigate to the parent cell, then get the immediately following sibling cell (which always contains the name in the HTML structure).

**Rationale**:

- Mirrors the actual HTML structure where name cells always follow image cells
- Eliminates string matching and iteration complexity
- Works correctly with multi-column layouts
- More maintainable and easier to understand

**Implementation**: `extractShowNameFromGoquery` in `internal/parser/show_parser.go` uses goquery's `Closest()` and `Next()` for reliable sibling navigation.

## UTF-8 Safety for Scraped Content

**Decision**: Apply multi-layer UTF-8 sanitization across the entire data pipeline — HTML parsing, subtitle file content, ZIP filenames, and gRPC serialization.

**Rationale**:

- feliratok.eu serves HTML in various encodings (ISO-8859-1, Windows-1252, UTF-8) and may not always declare charset correctly
- ZIP archives contain filenames encoded in the creator's local encoding, which are not valid UTF-8
- Protocol Buffers requires all string fields to be valid UTF-8; invalid sequences cause marshaling errors
- Subtitle files downloaded from the site may be in non-UTF-8 encodings

**Implementation**:

- `internal/parser/charset.go` — `NewUTF8Reader` wraps `io.Reader` with automatic encoding detection and conversion to UTF-8, used by all HTML parsers
- `internal/grpc/converters.go` — `sanitizeUTF8` / `sanitizeUTF8Slice` replace invalid sequences with U+FFFD as defense-in-depth before protobuf marshaling
- `internal/services/subtitle_downloader_impl.go` — `convertToUTF8` uses `golang.org/x/text/transform` with charset detection for subtitle file content; `strings.ToValidUTF8` for ZIP entry filenames
