# Design Decisions — Parsing

## Generic Parser Interfaces

**Decision**: Use `Parser[T]` and `SingleResultParser[T]` generic interfaces for type-safe HTML parsing.

**Rationale**:

- Type safety prevents runtime errors
- Reusable pattern for different model types
- Clear contract for parser implementations

**Implementation**: `internal/parser/interfaces.go` defines generic interfaces implemented by ShowParser, SubtitleParser, and ThirdPartyIdParser.

## Batch Processing

**Decision**: Show subtitle fetching is batched (20 concurrent requests) to avoid overwhelming the upstream server.

**Rationale**:

- Prevents overloading feliratok.eu infrastructure
- Balances speed with responsible resource usage
- Allows rate limiting if needed in the future

**Implementation**: `GetShowSubtitles` processes shows in batches of 20 with parallel processing within each batch.

## Parser Reusability

**Decision**: The `SubtitleParser` is used for both individual show subtitle pages and the main page.

**Rationale**:

- Both pages share the same HTML table structure
- Reduces code duplication
- Single source of truth for subtitle parsing logic
- Only requires one additional method (`extractShowIDFromCategory`) for main page support

**Implementation**:

- `SubtitleParser.ParseHtml` works for both page types
- `extractShowIDFromCategory` method added to extract show ID from category column when available

## Parser Handles All Data Normalization

**Decision**: The SubtitleParser directly handles language conversion, quality extraction, season/episode parsing, and data normalization rather than using a separate converter service.

**Rationale**:

- Parsing and normalization are tightly coupled operations
- Eliminates unnecessary intermediate data structures
- Reduces code complexity and maintenance burden
- Parser has all HTML context needed for normalization
- Single responsibility: transform HTML → normalized models

**Implementation**:

- `SubtitleParser` includes:
  - `convertLanguageToISO` for Hungarian → ISO 639-1 conversion
  - `parseReleaseInfo` for quality and release group extraction
  - `parseDescription` for season/episode/show name extraction
  - `detectQuality` for quality enum conversion
- All normalization happens during HTML parsing in one pass

## Show Name Extraction via DOM Traversal

**Decision**: Use direct DOM sibling traversal (`.Next()`) to find show names instead of iterating through all table cells with string matching.

**Problem**: The original implementation used a complex flag-based iteration pattern, searching for the link's href in all cells, then looking for subsequent `td.sangol` elements. This was fragile and prone to incorrect matches in multi-column layouts.

**Solution**: Simplified to:

1. Get the parent `<td>` of the show link (the image cell)
2. Use `.Next()` to get the immediately following sibling `<td>`
3. Verify it has class "sangol" (the name cell)
4. Extract the show name from the first `<div>` in that cell

**Rationale**:

- Mirrors the actual HTML structure where name cells always follow image cells
- Eliminates string matching and iteration complexity
- Works correctly with multi-column layouts (2+ shows per row)
- More maintainable and easier to understand
- Preserves show names with parenthetical alternate titles (e.g., "Cash Queens (Les Lionnes)")

**Implementation**: `internal/parser/show_parser.go` - `extractShowNameFromGoquery` method uses goquery's `Closest()` and `Next()` for reliable sibling navigation.

## UTF-8 Safety for Scraped Content

**Decision**: Apply multi-layer UTF-8 sanitization across the entire data pipeline — HTML parsing, subtitle file content, ZIP filenames, and gRPC serialization.

**Rationale**:

- feliratok.eu serves HTML in various encodings (ISO-8859-1, Windows-1252, UTF-8) and may not always declare charset correctly
- ZIP archives contain filenames encoded in the creator's local encoding (e.g., CP437, ISO-8859-1), which are not valid UTF-8
- Protocol Buffers requires all `string` fields to be valid UTF-8; invalid sequences cause marshaling errors at the gRPC transport layer
- Subtitle files downloaded from the site may be in non-UTF-8 encodings

**Implementation**:

- `internal/parser/charset.go` - `NewUTF8Reader` wraps `io.Reader` with `golang.org/x/net/html/charset` for automatic encoding detection and conversion to UTF-8, used by all HTML parsers
- `internal/grpc/converters.go` - `sanitizeUTF8` / `sanitizeUTF8Slice` use `strings.ToValidUTF8` to replace invalid sequences with U+FFFD as a defense-in-depth safety net before protobuf marshaling
- `internal/services/subtitle_downloader_impl.go` - `convertToUTF8` uses `golang.org/x/text/transform` with `charset.DetermineEncoding` for subtitle file content; `strings.ToValidUTF8` for ZIP entry filenames
