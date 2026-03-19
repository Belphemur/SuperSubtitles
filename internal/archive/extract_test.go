package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// createTestZip creates a test ZIP file with the given files.
func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for filename, content := range files {
		f, err := w.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create file %s in ZIP: %v", filename, err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatalf("Failed to write content to %s in ZIP: %v", filename, err)
		}
	}

	err := w.Close()
	if err != nil {
		t.Fatalf("Failed to close ZIP writer: %v", err)
	}

	return buf.Bytes()
}

func testLogger() zerolog.Logger {
	return zerolog.New(io.Discard)
}

func TestDetectZipBomb(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		files       map[string]string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Normal season pack - should pass",
			files: map[string]string{
				"show.s03e01.srt": strings.Repeat("Normal subtitle content\n", 100),
				"show.s03e02.srt": strings.Repeat("Normal subtitle content\n", 100),
				"show.s03e03.srt": strings.Repeat("Normal subtitle content\n", 100),
			},
			shouldError: false,
		},
		{
			name: "Single large file within limits - should pass",
			files: map[string]string{
				"show.s03e01.srt": strings.Repeat("A", 15*1024*1024), // 15 MB (under 20 MB limit)
			},
			shouldError: false,
		},
		{
			name: "Large ASS file within ASS limit - should pass",
			files: map[string]string{
				"show.s03e01.ass": strings.Repeat("A", 37*1024*1024), // 37 MB (over 20 MB limit but under 100 MB ASS limit)
			},
			shouldError: false,
		},
		{
			name: "File exceeds individual size limit - should fail",
			files: map[string]string{
				"malicious.srt": strings.Repeat("X", 25*1024*1024), // 25 MB > 20 MB limit
			},
			shouldError: true,
			errorMsg:    "exceeds maximum uncompressed size",
		},
		{
			name: "ASS file exceeds ASS size limit - should fail",
			files: map[string]string{
				"malicious.ass": strings.Repeat("X", 101*1024*1024), // 101 MB > 100 MB ASS limit
			},
			shouldError: true,
			errorMsg:    "exceeds maximum uncompressed size",
		},
		{
			name: "Total size exceeds limit - should fail",
			files: map[string]string{
				"file1.srt": strings.Repeat("Y", 25*1024*1024), // 25 MB
				"file2.srt": strings.Repeat("Z", 25*1024*1024), // 25 MB
			},
			shouldError: true,
			errorMsg:    "exceeds maximum uncompressed size", // Fails on individual file first
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			zipContent := createTestZip(t, tt.files)
			err := DetectZipBomb(zipContent)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestDetectZipBomb_CompressionRatio(t *testing.T) {
	t.Parallel()
	normalFiles := map[string]string{
		"test1.srt": "Normal content that compresses well but not suspiciously\n",
		"test2.srt": "Another normal file with typical subtitle content\n",
	}

	zipContent := createTestZip(t, normalFiles)
	err := DetectZipBomb(zipContent)

	if err != nil {
		t.Errorf("Normal files should not trigger ZIP bomb detection, got: %v", err)
	}
}

func TestDetectZipBomb_InvalidZip(t *testing.T) {
	t.Parallel()

	err := DetectZipBomb([]byte("not a zip"))
	if err == nil {
		t.Fatal("expected error for invalid zip content")
	}
	if !strings.Contains(err.Error(), "failed to open ZIP") {
		t.Errorf("expected invalid zip error, got: %v", err)
	}
}

func TestDetectZipBomb_TotalUncompressedSize(t *testing.T) {
	t.Parallel()

	files := make(map[string]string)
	for i := 0; i < 6; i++ {
		files["show.s01e"+strconv.Itoa(i+1)+".ass"] = strings.Repeat("A", 18*1024*1024)
	}

	err := DetectZipBomb(createTestZip(t, files))
	if err == nil {
		t.Fatal("expected total uncompressed size error")
	}
	if !strings.Contains(err.Error(), "total uncompressed size exceeds limit") {
		t.Errorf("expected total size error, got: %v", err)
	}
}

func TestExtractEpisodeFromZip_Basic(t *testing.T) {
	t.Parallel()
	zipContent := createTestZip(t, map[string]string{
		"Show.S03E01.srt": "Episode 1 content",
		"Show.S03E02.srt": "Episode 2 content",
	})

	result, err := ExtractEpisodeFromZip(zipContent, 1, testLogger())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Filename != "Show.S03E01.srt" {
		t.Errorf("Expected filename 'Show.S03E01.srt', got '%s'", result.Filename)
	}

	if string(result.Content) != "Episode 1 content" {
		t.Errorf("Expected 'Episode 1 content', got '%s'", string(result.Content))
	}
}

func TestExtractEpisodeFromZip_NotFound(t *testing.T) {
	t.Parallel()
	zipContent := createTestZip(t, map[string]string{
		"Show.S03E01.srt": "Episode 1 content",
	})

	_, err := ExtractEpisodeFromZip(zipContent, 5, testLogger())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var episodeErr *ErrEpisodeNotFound
	if !errors.As(err, &episodeErr) {
		t.Errorf("Expected ErrEpisodeNotFound, got: %v", err)
	}
	if episodeErr.Error() == "" {
		t.Error("expected ErrEpisodeNotFound error message to be populated")
	}
	if !errors.Is(err, &ErrEpisodeNotFound{}) {
		t.Error("expected errors.Is to match ErrEpisodeNotFound")
	}
}

func TestExtractEpisodeFromZip_PrefersSrt(t *testing.T) {
	t.Parallel()
	zipContent := createTestZip(t, map[string]string{
		"show.s03e01.ass": "ASS content",
		"show.s03e01.srt": "SRT content",
		"show.s03e01.vtt": "VTT content",
	})

	result, err := ExtractEpisodeFromZip(zipContent, 1, testLogger())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.HasSuffix(result.Filename, ".srt") {
		t.Errorf("Expected .srt file, got '%s'", result.Filename)
	}

	if string(result.Content) != "SRT content" {
		t.Errorf("Expected 'SRT content', got '%s'", string(result.Content))
	}
}

func TestExtractEpisodeFromZip_MatchesPathAndPrefersSubtitleType(t *testing.T) {
	t.Parallel()

	zipContent := createTestZip(t, map[string]string{
		"Show/1x07/subtitle.txt": "text content",
		"Show/1x07/subtitle.ass": "ass content",
	})

	result, err := ExtractEpisodeFromZip(zipContent, 7, testLogger())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Filename != "subtitle.ass" {
		t.Errorf("Expected subtitle.ass, got %q", result.Filename)
	}
	if string(result.Content) != "ass content" {
		t.Errorf("Expected ASS content, got %q", result.Content)
	}
}

func TestExtractEpisodeFromZip_InvalidZip(t *testing.T) {
	t.Parallel()

	_, err := ExtractEpisodeFromZip([]byte("not a zip"), 1, testLogger())
	if err == nil {
		t.Fatal("expected invalid zip error")
	}
	if !strings.Contains(err.Error(), "failed to open ZIP for bomb detection") {
		t.Errorf("expected bomb detection open error, got: %v", err)
	}
}
