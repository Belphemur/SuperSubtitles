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
	// Maximum uncompressed size for a single ASS file (100 MB).
	// ASS files can legitimately be large because they often embed font data as base64.
	MaxUncompressedAssFileSize = 100 * 1024 * 1024
	// Maximum total uncompressed size for all files in an archive (100 MB).
	MaxTotalUncompressedSize = 100 * 1024 * 1024
)

// maxFileSizeForExtension returns the maximum allowed uncompressed size for a file
// based on its extension. ASS subtitle files can legitimately contain embedded fonts
// and may exceed the standard limit, so they receive a higher allowance.
func maxFileSizeForExtension(filename string) int64 {
	if strings.ToLower(path.Ext(filename)) == ".ass" {
		return MaxUncompressedAssFileSize
	}
	return MaxUncompressedFileSize
}

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
	if fileSize > maxFileSizeForExtension(w.fileName) {
		return 0, NewUnrecoverableError(
			"RAR archive entry exceeds maximum uncompressed size",
			fmt.Errorf("entry %s is %d bytes > %d bytes limit", w.fileName, fileSize, maxFileSizeForExtension(w.fileName)),
		)
	}

	totalSize := *w.totalWritten + int64(len(p))
	if totalSize > MaxTotalUncompressedSize {
		return 0, NewUnrecoverableError(
			"RAR archive total uncompressed size exceeds limit",
			fmt.Errorf("%d bytes > %d bytes limit", totalSize, MaxTotalUncompressedSize),
		)
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
		return nil, NewUnrecoverableError("failed to open RAR archive", err)
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
			return nil, NewUnrecoverableError("failed to read RAR entry", err)
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
				return nil, NewUnrecoverableError(
					"RAR archive contains path traversal in entry name",
					fmt.Errorf("%q", header.Name),
				)
			}
		}
		if entryName == "" || entryName == "." {
			entryName = "subtitle"
		}

		if header.UnPackedSize > maxFileSizeForExtension(entryName) {
			return nil, NewUnrecoverableError(
				"RAR archive entry exceeds maximum uncompressed size",
				fmt.Errorf("entry %s is %d bytes > %d bytes limit", entryName, header.UnPackedSize, maxFileSizeForExtension(entryName)),
			)
		}

		entryWriter, err := zipWriter.Create(entryName)
		if err != nil {
			return nil, NewError(fmt.Sprintf("failed to create ZIP entry %s", entryName), err)
		}

		limitWriter := &archiveLimitWriter{
			writer:       entryWriter,
			fileName:     entryName,
			totalWritten: &totalWritten,
		}

		if _, err := io.Copy(limitWriter, rarReader); err != nil {
			return nil, NewUnrecoverableError(fmt.Sprintf("failed to copy RAR entry %s", entryName), err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, NewError("failed to finalize ZIP archive", err)
	}

	return zipBuffer.Bytes(), nil
}
