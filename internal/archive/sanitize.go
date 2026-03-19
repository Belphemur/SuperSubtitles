package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// subtitleExtensions lists recognized subtitle file extensions.
var subtitleExtensions = map[string]bool{
	".srt": true,
	".ass": true,
	".vtt": true,
	".sub": true,
}

// isSubtitleFile checks whether a filename has a recognized subtitle extension.
func isSubtitleFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return subtitleExtensions[ext]
}

// SanitizeZip removes non-subtitle files from a ZIP archive and flattens the directory structure.
// Only files with recognized subtitle extensions (.srt, .ass, .vtt, .sub) are kept.
// All retained files are placed at the root level of the resulting archive.
// Duplicate filenames after flattening are disambiguated with a numeric suffix.
// It performs ZIP bomb detection before processing.
func SanitizeZip(zipContent []byte) ([]byte, error) {
	if err := DetectZipBomb(zipContent); err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP archive for sanitization: %w", err)
	}

	outBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(outBuf)

	usedNames := make(map[string]int)
	var totalRead int64

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Normalize backslashes before extracting base name — on non-Windows,
		// filepath.Base does not split on '\', which could preserve path components.
		normalized := strings.ReplaceAll(file.Name, "\\", "/")
		baseName := strings.ToValidUTF8(filepath.Base(normalized), "�")
		if !isSubtitleFile(baseName) {
			continue
		}

		// Deduplicate filenames after flattening
		flatName := deduplicate(baseName, usedNames)

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open ZIP entry %s: %w", file.Name, err)
		}

		writer, err := zipWriter.Create(flatName)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to create ZIP entry %s: %w", flatName, err)
		}

		// Enforce per-file size limit during decompression to guard against
		// spoofed ZIP headers that pass DetectZipBomb but expand beyond limits.
		limitedReader := io.LimitReader(rc, MaxUncompressedFileSize+1)
		content, err := io.ReadAll(limitedReader)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read ZIP entry %s: %w", flatName, err)
		}
		if int64(len(content)) > MaxUncompressedFileSize {
			return nil, fmt.Errorf("ZIP entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
				flatName, len(content), MaxUncompressedFileSize)
		}
		totalRead += int64(len(content))
		if totalRead > MaxTotalUncompressedSize {
			return nil, fmt.Errorf("ZIP archive total uncompressed size exceeds limit (%d bytes > %d bytes limit)",
				totalRead, MaxTotalUncompressedSize)
		}

		content = convertToUTF8(content)

		if _, err := writer.Write(content); err != nil {
			return nil, fmt.Errorf("failed to write ZIP entry %s: %w", flatName, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize sanitized ZIP archive: %w", err)
	}

	return outBuf.Bytes(), nil
}

// deduplicate returns a unique filename by appending a numeric suffix when needed.
// It tracks usage via the provided map.
func deduplicate(name string, used map[string]int) string {
	lower := strings.ToLower(name)
	count, exists := used[lower]
	if !exists {
		used[lower] = 1
		return name
	}
	used[lower] = count + 1
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s_%d%s", base, count+1, ext)
}

// convertToUTF8 detects the character encoding of text content and converts it to UTF-8.
// It handles BOM detection and uses heuristic charset detection.
// If the content is already valid UTF-8, it is returned as-is.
func convertToUTF8(content []byte) []byte {
	if len(content) == 0 || utf8.Valid(content) {
		return content
	}

	encoding, _, _ := charset.DetermineEncoding(content, "text/plain")

	decoded, _, err := transform.Bytes(encoding.NewDecoder(), content)
	if err != nil {
		return content
	}

	return decoded
}
