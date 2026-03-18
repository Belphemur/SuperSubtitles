package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/nwaples/rardecode/v2"
)

// Size limit constants for archive operations.
const (
	// Maximum compression ratio (uncompressed/compressed).
	// Highly repetitive content can legitimately compress to 1000:1 or more.
	// Real subtitle files rarely exceed 20:1, but we set a generous limit to avoid false positives.
	MaxCompressionRatio = 10000
	// Maximum uncompressed size for a single file (20 MB).
	MaxUncompressedFileSize = 20 * 1024 * 1024
	// Maximum total uncompressed size for all files in an archive (100 MB).
	MaxTotalUncompressedSize = 100 * 1024 * 1024
)

// archiveLimitWriter is an io.Writer that enforces per-file and total uncompressed size limits
// to guard against ZIP/RAR bomb extraction attacks.
type archiveLimitWriter struct {
	writer       io.Writer
	fileName     string
	fileWritten  int64
	totalWritten *int64
}

func (w *archiveLimitWriter) Write(p []byte) (int, error) {
	fileSize := w.fileWritten + int64(len(p))
	if fileSize > MaxUncompressedFileSize {
		return 0, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
			w.fileName, fileSize, MaxUncompressedFileSize)
	}

	totalSize := *w.totalWritten + int64(len(p))
	if totalSize > MaxTotalUncompressedSize {
		return 0, fmt.Errorf("RAR archive total uncompressed size exceeds limit (%d bytes > %d bytes limit)",
			totalSize, MaxTotalUncompressedSize)
	}

	n, err := w.writer.Write(p)
	w.fileWritten += int64(n)
	*w.totalWritten += int64(n)
	return n, err
}

// ConvertRarToZip converts a RAR archive to a ZIP archive.
// It sanitizes entry names to prevent path traversal attacks and enforces
// per-file and total uncompressed size limits.
func ConvertRarToZip(rarContent []byte) ([]byte, error) {
	rarReader, err := rardecode.NewReader(
		bytes.NewReader(rarContent),
		rardecode.MaxDictionarySize(MaxTotalUncompressedSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open RAR archive: %w", err)
	}

	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)
	var totalWritten int64

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read RAR entry: %w", err)
		}
		if header.IsDir {
			continue
		}

		entryName := strings.ToValidUTF8(header.Name, "")

		// Sanitize the entry name to prevent Zip-Slip path traversal attacks.
		// RAR archives can store paths with backslashes or absolute paths.
		entryName = strings.ReplaceAll(entryName, "\\", "/")
		entryName = path.Clean(entryName)
		entryName = strings.TrimLeft(entryName, "/")
		for _, component := range strings.Split(entryName, "/") {
			if component == ".." {
				return nil, fmt.Errorf("RAR archive contains path traversal in entry name: %q", header.Name)
			}
		}
		if entryName == "" || entryName == "." {
			entryName = "subtitle"
		}

		if header.UnPackedSize > MaxUncompressedFileSize {
			return nil, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
				entryName, header.UnPackedSize, MaxUncompressedFileSize)
		}

		entryWriter, err := zipWriter.Create(entryName)
		if err != nil {
			return nil, fmt.Errorf("failed to create ZIP entry %s: %w", entryName, err)
		}

		limitWriter := &archiveLimitWriter{
			writer:       entryWriter,
			fileName:     entryName,
			totalWritten: &totalWritten,
		}

		if _, err := io.Copy(limitWriter, rarReader); err != nil {
			return nil, fmt.Errorf("failed to copy RAR entry %s: %w", entryName, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize ZIP archive: %w", err)
	}

	return zipBuffer.Bytes(), nil
}
