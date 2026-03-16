package services

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// readRARFixtureByName reads a RAR test fixture by filename from the shared .tests-files directory.
func readRARFixtureByName(t *testing.T, filename string) []byte {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current test file path")
	}

	fixturePath := filepath.Join(filepath.Dir(currentFile), "..", "..", ".tests-files", filename)
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read RAR fixture %s: %v", fixturePath, err)
	}

	return content
}

// assertValidConvertedZIP is a shared helper that verifies a ZIP produced by
// convertRarToZip is structurally valid: non-empty, all entries readable, no
// path traversal, no empty names or empty content.
func assertValidConvertedZIP(t *testing.T, zipContent []byte) {
	t.Helper()

	if len(zipContent) == 0 {
		t.Fatal("convertRarToZip returned empty ZIP")
	}

	zr, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		t.Fatalf("output is not a valid ZIP archive: %v", err)
	}

	if len(zr.File) == 0 {
		t.Fatal("ZIP archive contains no files")
	}

	for _, f := range zr.File {
		if f.Name == "" {
			t.Errorf("ZIP contains an entry with an empty name")
		}
		if strings.Contains(f.Name, "..") {
			t.Errorf("ZIP entry %q contains path traversal component", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			t.Errorf("failed to open ZIP entry %q: %v", f.Name, err)
			continue
		}
		n, err := io.Copy(io.Discard, rc)
		_ = rc.Close()
		if err != nil {
			t.Errorf("failed to read ZIP entry %q: %v", f.Name, err)
		}
		if n == 0 {
			t.Errorf("ZIP entry %q is empty", f.Name)
		}
	}
}

func TestConvertRarToZip_RenegadeFixture(t *testing.T) {
	t.Parallel()

	rarContent := readRARFixtureByName(t, "Renegade.S01.WEB-DL.H.264-JiTB.eng.rar")

	zipContent, err := convertRarToZip(rarContent)
	if err != nil {
		t.Fatalf("convertRarToZip returned unexpected error: %v", err)
	}
	assertValidConvertedZIP(t, zipContent)
}

func TestConvertRarToZip_AncladosFixture(t *testing.T) {
	t.Parallel()
	t.Skip("Currently unsupported by rardecode, issue opened")

	rarContent := readRARFixtureByName(t, "Anclados.S01.1080p.AMZN.WEB-DL.DD+2.0.H.264-CasStudio_eng.rar")

	zipContent, err := convertRarToZip(rarContent)
	if err != nil {
		t.Fatalf("convertRarToZip returned unexpected error: %v", err)
	}
	assertValidConvertedZIP(t, zipContent)
}

func TestConvertRarToZip_InvalidInput(t *testing.T) {
	t.Parallel()

	_, err := convertRarToZip([]byte("this is not a rar file"))
	if err == nil {
		t.Fatal("expected error for invalid RAR input, got nil")
	}
}

func TestArchiveLimitWriter_Write_WithinLimits(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	var total int64
	w := &archiveLimitWriter{
		writer:       &buf,
		fileName:     "test.srt",
		totalWritten: &total,
	}

	data := []byte("hello subtitle")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("wrote %d bytes, want %d", n, len(data))
	}
	if buf.String() != string(data) {
		t.Errorf("buffer content %q, want %q", buf.String(), string(data))
	}
	if total != int64(len(data)) {
		t.Errorf("totalWritten = %d, want %d", total, len(data))
	}
}

func TestArchiveLimitWriter_Write_ExceedsFileLimit(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	var total int64
	w := &archiveLimitWriter{
		writer:       &buf,
		fileName:     "big.srt",
		fileWritten:  maxUncompressedFileSize - 1,
		totalWritten: &total,
	}

	// This single write pushes the per-file total over the limit.
	_, err := w.Write([]byte("xx"))
	if err == nil {
		t.Fatal("expected per-file size limit error, got nil")
	}
	if !strings.Contains(err.Error(), "big.srt") {
		t.Errorf("error message should mention the filename, got: %v", err)
	}
}

func TestArchiveLimitWriter_Write_ExceedsTotalLimit(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	total := int64(maxTotalUncompressedSize - 1)
	w := &archiveLimitWriter{
		writer:       &buf,
		fileName:     "entry.srt",
		totalWritten: &total,
	}

	// This single write pushes the aggregate total over the limit.
	_, err := w.Write([]byte("xx"))
	if err == nil {
		t.Fatal("expected total size limit error, got nil")
	}
	if !strings.Contains(err.Error(), "total") {
		t.Errorf("error message should mention total limit, got: %v", err)
	}
}
