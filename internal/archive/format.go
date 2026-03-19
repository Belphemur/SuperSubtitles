package archive

import (
	"bytes"
	"mime"
	"path/filepath"
	"strings"
)

// Archive format constants.
const (
	FormatUnknown = ""
	FormatZIP     = "zip"
	FormatRAR     = "rar"
)

// IsZipFile checks if the content is a ZIP file using magic number detection.
// ZIP files start with PK\x03\x04 (0x504B0304) or PK\x05\x06 (empty archive) or PK\x07\x08 (spanned archive).
func IsZipFile(content []byte) bool {
	if len(content) < 4 {
		return false
	}
	return (content[0] == 0x50 && content[1] == 0x4B &&
		(content[2] == 0x03 && content[3] == 0x04 || // Standard ZIP
			content[2] == 0x05 && content[3] == 0x06 || // Empty ZIP
			content[2] == 0x07 && content[3] == 0x08)) // Spanned ZIP
}

// IsRarFile checks if the content is a RAR file using magic number detection.
// RAR 4 uses Rar!\x1A\x07\x00 and RAR 5 uses Rar!\x1A\x07\x01\x00.
func IsRarFile(content []byte) bool {
	return (len(content) >= 7 && bytes.Equal(content[:7], []byte{'R', 'a', 'r', '!', 0x1A, 0x07, 0x00})) ||
		(len(content) >= 8 && bytes.Equal(content[:8], []byte{'R', 'a', 'r', '!', 0x1A, 0x07, 0x01, 0x00}))
}

// IsZipContentType checks if the MIME type indicates a ZIP file.
func IsZipContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.EqualFold(contentType, "application/zip")
	}
	return mediaType == "application/zip" ||
		mediaType == "application/x-zip-compressed"
}

// IsRarContentType checks if the MIME type indicates a RAR file.
func IsRarContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(contentType))
	}

	switch mediaType {
	case "application/vnd.rar", "application/x-rar-compressed", "application/x-rar":
		return true
	default:
		return false
	}
}

// DetectFormat determines the archive format from content bytes and content type.
func DetectFormat(content []byte, contentType string) string {
	switch {
	case IsRarFile(content):
		return FormatRAR
	case IsZipFile(content):
		return FormatZIP
	case IsRarContentType(contentType):
		return FormatRAR
	case IsZipContentType(contentType):
		return FormatZIP
	default:
		return FormatUnknown
	}
}

// NormalizeContentType returns the canonical MIME type for the given archive format.
func NormalizeContentType(contentType, format string) string {
	switch format {
	case FormatZIP:
		return "application/zip"
	case FormatRAR:
		return "application/vnd.rar"
	default:
		return contentType
	}
}

// ExtensionForContentType returns the preferred filename extension for a MIME type.
func ExtensionForContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		if before, _, ok := strings.Cut(contentType, ";"); ok {
			mediaType = strings.TrimSpace(before)
		} else {
			mediaType = contentType
		}
	}
	mediaType = strings.ToLower(mediaType)

	switch mediaType {
	case "application/zip", "application/x-zip-compressed":
		return ".zip"
	case "application/vnd.rar", "application/x-rar-compressed", "application/x-rar":
		return ".rar"
	case "application/x-subrip":
		return ".srt"
	case "application/x-ass", "text/ass":
		return ".ass"
	case "text/vtt", "text/webvtt":
		return ".vtt"
	case "application/x-sub":
		return ".sub"
	}

	if strings.Contains(mediaType, "srt") {
		return ".srt"
	}

	return ".srt"
}

// ContentTypeForFilename returns the canonical content type for a filename.
func ContentTypeForFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".srt":
		return "application/x-subrip"
	case ".ass":
		return "application/x-ass"
	case ".vtt":
		return "text/vtt"
	case ".sub":
		return "application/x-sub"
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/vnd.rar"
	default:
		return "application/octet-stream"
	}
}
