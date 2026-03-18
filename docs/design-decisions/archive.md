# Design Decisions — Archive

## Dedicated Archive Package

**Decision**: All archive format detection, RAR-to-ZIP conversion, ZIP bomb detection, and episode extraction live in a single `internal/archive` package, separate from the service layer.

**Rationale**:

- Isolates low-level byte/format concerns from download orchestration logic
- Each responsibility (detection, conversion, extraction) is independently testable with fixture-backed tests
- Prevents the service layer from growing with format-specific imports (`archive/zip`, `rardecode`)
- Makes it straightforward to add new archive formats or tighten security limits without touching service code

**Implementation**: Three files partition the responsibilities:

- `internal/archive/format.go` — `IsZipFile()`, `IsRarFile()`, `DetectFormat()`, content-type helpers (`IsZipContentType`, `IsRarContentType`, `NormalizeContentType`). Format constants `FormatZIP`, `FormatRAR`, `FormatUnknown`.
- `internal/archive/convert.go` — `ConvertRarToZip()` with `archiveLimitWriter` enforcing per-file and total size limits.
- `internal/archive/extract.go` — `ExtractEpisodeFromZip()`, `DetectZipBomb()`, `EpisodeFile` result type, `ErrEpisodeNotFound` error type.

## RAR Decode Fork

**Decision**: Use the `github.com/Belphemur/rardecode` fork of `github.com/nwaples/rardecode/v2` via a Go `replace` directive.

**Rationale**:

- The fork includes fixes not yet merged upstream
- A `replace` directive keeps the import path unchanged (`github.com/nwaples/rardecode/v2`) so no code changes are needed beyond `go.mod`
- Pinned to a specific pseudo-version for reproducible builds

**Implementation**: `go.mod` contains `replace github.com/nwaples/rardecode/v2 => github.com/Belphemur/rardecode/v2 v2.0.0-20260318154427-1044718e45a8`.

## Remove SkipCheck From RAR Decoding

**Decision**: Do not pass `rardecode.SkipCheck` when opening RAR archives. Only `MaxDictionarySize` is set.

**Rationale**:

- `SkipCheck` disables CRC validation of RAR entries, silently accepting corrupted data
- For a subtitle service the integrity of decompressed text matters — a corrupt subtitle is worse than a failed download
- `MaxDictionarySize` alone is sufficient to bound memory usage during decompression

**Implementation**: `ConvertRarToZip()` in `internal/archive/convert.go` creates the reader with `rardecode.NewReader(reader, rardecode.MaxDictionarySize(MaxTotalUncompressedSize))` — no `SkipCheck` option.

## ZIP Bomb Detection Strategy

**Decision**: Detect ZIP bombs by checking per-file size, total uncompressed size, and compression ratio, applied at two stages — during RAR-to-ZIP conversion (write-time limits) and before ZIP episode extraction (read-time ratio check).

**Rationale**:

- Write-time limits in `archiveLimitWriter` stop decompression bombs during RAR→ZIP conversion before they exhaust memory
- Read-time ratio checks in `DetectZipBomb()` catch pre-existing malicious ZIP files that were never converted
- Three complementary thresholds (`MaxCompressionRatio`, `MaxUncompressedFileSize`, `MaxTotalUncompressedSize`) cover both single-entry and multi-entry bomb patterns
- Generous limits (10 000:1 ratio, 20 MB per file, 100 MB total) avoid false positives on legitimate subtitle archives

**Implementation**: `archiveLimitWriter` in `internal/archive/convert.go` enforces per-file and total limits during `ConvertRarToZip()`. `DetectZipBomb()` in `internal/archive/extract.go` scans all ZIP entries and compares the total uncompressed size against the compressed archive size using `MaxCompressionRatio`.
