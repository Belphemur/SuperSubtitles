package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/nwaples/rardecode/v2"
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
	if fileSize > maxUncompressedFileSize {
		return 0, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
			w.fileName, fileSize, maxUncompressedFileSize)
	}

	totalSize := *w.totalWritten + int64(len(p))
	if totalSize > maxTotalUncompressedSize {
		return 0, fmt.Errorf("RAR archive total uncompressed size exceeds limit (%d bytes > %d bytes limit)",
			totalSize, maxTotalUncompressedSize)
	}

	n, err := w.writer.Write(p)
	w.fileWritten += int64(n)
	*w.totalWritten += int64(n)
	return n, err
}

// convertRarToZip converts a RAR archive to a ZIP archive.
// It sanitizes entry names to prevent path traversal attacks and enforces
// per-file and total uncompressed size limits.
func convertRarToZip(rarContent []byte) ([]byte, error) {
	rarReader, err := rardecode.NewReader(
		bytes.NewReader(rarContent),
		rardecode.MaxDictionarySize(maxTotalUncompressedSize),
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

		if header.UnPackedSize > maxUncompressedFileSize {
			return nil, fmt.Errorf("RAR archive entry %s exceeds maximum uncompressed size (%d bytes > %d bytes limit)",
				entryName, header.UnPackedSize, maxUncompressedFileSize)
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
