# SuperSubtitles — Design Decisions

This document explains key architectural and design decisions made in the SuperSubtitles project.

## Partial Failure Resilience

**Decision**: The client returns whatever data it successfully fetched, logging warnings for failed endpoints rather than failing the entire operation.

**Rationale**:

- feliratok.eu endpoints may be temporarily unavailable
- Users benefit from partial data rather than complete failure
- Warnings in logs allow monitoring of endpoint health

**Implementation**: All parallel fetching operations collect errors but still return successful results if any endpoints succeed.

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

## Programmatic Test Fixtures

**Decision**: HTML fixtures generated via reusable utility functions instead of hardcoded strings.

**Rationale**:

- HTML structure changes require updating only one generator
- Tests express intent through configuration structs
- Consistency across all tests
- Easy to add edge cases

**Implementation**: `internal/testutil/html_fixtures.go` provides generators for all HTML table types.

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

## Client Architecture

**Decision**: Keep unified client interface rather than splitting into multiple specialized clients.

**Current State**:

- ~600 lines in `client.go` handling all operations
- Single `Client` interface provides unified API
- Clear method separation with comprehensive tests

**Rationale**:

- Current structure is manageable and well-tested
- Single interface is convenient for consumers
- Premature splitting adds unnecessary complexity

**Future Consideration**: If the client grows significantly (>1000 lines) or new features require substantial complexity, consider splitting into:

1. Show Client (GetShowList, GetShowSubtitles)
2. Subtitle Client (GetSubtitles, GetRecentSubtitles)
3. Metadata Client (third-party IDs, update checking)
4. Download Client (already delegates to SubtitleDownloader service)

## Parallel Pagination

**Decision**: Fetch subtitle pages in parallel batches of 2 rather than sequentially.

**Rationale**:

- ~3x faster for shows with many subtitle pages
- Balances speed with server load
- Batch size of 2 is conservative and respectful

**Implementation**: Pages 2-3 fetched together, then 4-5, etc. First page always fetched alone to determine total pages.

## Error Handling Strategy

**Decision**: Use custom error types with `Is()` method support, wrap errors with `fmt.Errorf`, and prefer partial success over complete failure.

**Rationale**:

- `errors.Is()` support enables proper error checking
- Wrapped errors preserve context
- Partial success maximizes data availability
- Logged warnings enable monitoring

**Implementation**:

- `ErrNotFound` custom error in `internal/client/errors.go`
- All errors wrapped with context
- Parallel operations collect errors but return successful results
