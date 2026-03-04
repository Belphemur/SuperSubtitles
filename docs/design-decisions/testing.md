# Design Decisions — Testing

## Programmatic Test Fixtures

**Decision**: HTML fixtures generated via reusable utility functions instead of hardcoded strings.

**Rationale**:

- HTML structure changes require updating only one generator
- Tests express intent through configuration structs
- Consistency across all tests
- Easy to add edge cases
- Prevents brittle hardcoded HTML in tests

**Implementation**: `internal/testutil/html_fixtures.go` provides generators for all HTML table types:

- `GenerateSubtitleTableHTML` - Basic subtitle listings
- `GenerateSubtitleTableHTMLWithPagination` - Subtitles with pagination
- `GenerateShowTableHTML` - Single-column show listings
- `GenerateShowTableHTMLMultiColumn` - Multi-column grid layouts (e.g., 2 shows per row)
- `GenerateThirdPartyIDHTML` - Third-party ID detail pages
- `GeneratePaginationHTML` - Pagination elements

**Evolution**: Added `GenerateShowTableHTMLMultiColumn` to support the actual website's grid layout for special show listing pages, ensuring tests match production HTML structure.
