package services

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"github.com/Belphemur/SuperSubtitles/v2/internal/cache"
	internalConfig "github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/metrics"
	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// createTestZip creates a test ZIP file with season pack structure
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

func buildDownloadURL(baseURL, subtitleID string) string {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/index.php"
	query := parsedURL.Query()
	query.Set("action", "letolt")
	query.Set("felirat", subtitleID)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String()
}

func TestDownloadSubtitle_NonZipFile(t *testing.T) {
	t.Parallel()
	// Create test HTTP server
	content := "1\n00:00:01,000 --> 00:00:02,000\nTest subtitle\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-subrip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	// Create downloader
	downloader := NewSubtitleDownloader(server.Client())

	// Test download
	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		nil,
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}

	if result.Filename != "123456789.srt" {
		t.Errorf("Expected filename '123456789.srt', got '%s'", result.Filename)
	}

	if string(result.Content) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(result.Content))
	}

	if result.ContentType != "application/x-subrip" {
		t.Errorf("Expected content type 'application/x-subrip', got '%s'", result.ContentType)
	}
}

func TestDownloadSubtitle_ZipFileNoEpisode(t *testing.T) {
	t.Parallel()
	// Create test ZIP
	zipContent := createTestZip(t, map[string]string{
		"Show.S03E01.srt": "Episode 1 content",
		"Show.S03E02.srt": "Episode 2 content",
	})

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	// Create downloader
	downloader := NewSubtitleDownloader(server.Client())

	// Test download without episode number (should return ZIP as-is)
	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		nil,
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}

	if !bytes.Equal(result.Content, zipContent) {
		t.Error("Expected ZIP content to be returned as-is")
	}

	if result.ContentType != "application/zip" {
		t.Errorf("Expected content type 'application/zip', got '%s'", result.ContentType)
	}

	// Assert filename has .zip extension for ZIP files
	expectedFilename := "123456789.zip"
	if result.Filename != expectedFilename {
		t.Errorf("Expected filename '%s', got '%s'", expectedFilename, result.Filename)
	}
}

func TestDownloadSubtitle_ExtractEpisodeFromZip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		zipFiles        map[string]string
		requestEpisode  *int
		expectedFile    string
		expectedContent string
		shouldFail      bool
	}{
		{
			name: "Extract S03E01",
			zipFiles: map[string]string{
				"Hightown.S03E01.Good.Times.NF.WEB-DL.en.srt":      "Episode 1 subtitle content",
				"Hightown.S03E02.I.Said.No.No.No.NF.WEB-DL.en.srt": "Episode 2 subtitle content",
				"Hightown.S03E03.Fall.Brook.NF.WEB-DL.en.srt":      "Episode 3 subtitle content",
			},
			requestEpisode:  testutil.IntPtr(1),
			expectedFile:    "Hightown.S03E01.Good.Times.NF.WEB-DL.en.srt",
			expectedContent: "Episode 1 subtitle content",
			shouldFail:      false,
		},
		{
			name: "Extract S03E02",
			zipFiles: map[string]string{
				"Hightown.S03E01.Good.Times.NF.WEB-DL.en.srt":      "Episode 1 subtitle content",
				"Hightown.S03E02.I.Said.No.No.No.NF.WEB-DL.en.srt": "Episode 2 subtitle content",
				"Hightown.S03E03.Fall.Brook.NF.WEB-DL.en.srt":      "Episode 3 subtitle content",
			},
			requestEpisode:  testutil.IntPtr(2),
			expectedFile:    "Hightown.S03E02.I.Said.No.No.No.NF.WEB-DL.en.srt",
			expectedContent: "Episode 2 subtitle content",
			shouldFail:      false,
		},
		{
			name: "Extract with lowercase pattern (s03e05)",
			zipFiles: map[string]string{
				"show.s03e04.srt": "Episode 4 content",
				"show.s03e05.srt": "Episode 5 content",
				"show.s03e06.srt": "Episode 6 content",
			},
			requestEpisode:  testutil.IntPtr(5),
			expectedFile:    "show.s03e05.srt",
			expectedContent: "Episode 5 content",
			shouldFail:      false,
		},
		{
			name: "Extract with 3x07 pattern",
			zipFiles: map[string]string{
				"show.3x06.srt": "Episode 6 content",
				"show.3x07.srt": "Episode 7 content",
				"show.3x08.srt": "Episode 8 content",
			},
			requestEpisode:  testutil.IntPtr(7),
			expectedFile:    "show.3x07.srt",
			expectedContent: "Episode 7 content",
			shouldFail:      false,
		},
		{
			name: "Extract with nested folder structure",
			zipFiles: map[string]string{
				"Hightown.S03.NF.WEB-DL.en/Hightown.S03E01.Good.Times.NF.WEB-DL.en.srt":      "Episode 1 content",
				"Hightown.S03.NF.WEB-DL.en/Hightown.S03E02.I.Said.No.No.No.NF.WEB-DL.en.srt": "Episode 2 content",
			},
			requestEpisode:  testutil.IntPtr(1),
			expectedFile:    "Hightown.S03E01.Good.Times.NF.WEB-DL.en.srt",
			expectedContent: "Episode 1 content",
			shouldFail:      false,
		},
		{
			name: "Episode 1 does not match Episode 10 (regex boundary test)",
			zipFiles: map[string]string{
				"show.s03e10.srt": "Episode 10 content",
				"show.s03e11.srt": "Episode 11 content",
			},
			requestEpisode: testutil.IntPtr(1),
			shouldFail:     true,
		},
		{
			name: "Episode not found in ZIP",
			zipFiles: map[string]string{
				"show.s03e01.srt": "Episode 1 content",
				"show.s03e02.srt": "Episode 2 content",
			},
			requestEpisode: testutil.IntPtr(10),
			shouldFail:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create test ZIP
			zipContent := createTestZip(t, tt.zipFiles)

			// Create test HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/zip")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(zipContent)
			}))
			defer server.Close()

			// Create downloader
			downloader := NewSubtitleDownloader(server.Client())

			// Test download with episode extraction
			result, err := downloader.DownloadSubtitle(
				context.Background(),
				buildDownloadURL(server.URL, "123456789"),
				tt.requestEpisode,
			)

			if tt.shouldFail {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), "not found") {
					t.Errorf("Expected 'not found' error, got: %v", err)
				}
				if !errors.Is(err, &apperrors.ErrSubtitleNotFoundInZip{}) {
					t.Errorf("Expected errors.Is to match ErrSubtitleNotFoundInZip, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
				return
			}

			if result.Filename != tt.expectedFile {
				t.Errorf("Expected filename '%s', got '%s'", tt.expectedFile, result.Filename)
			}

			if string(result.Content) != tt.expectedContent {
				t.Errorf("Expected content '%s', got '%s'", tt.expectedContent, string(result.Content))
			}

			if result.ContentType != "application/x-subrip" {
				t.Errorf("Expected content type 'application/x-subrip', got '%s'", result.ContentType)
			}
		})
	}
}

func TestDownloadSubtitle_Caching(t *testing.T) {
	t.Parallel()
	requestCount := 0
	zipContent := createTestZip(t, map[string]string{
		"show.s03e01.srt": "Episode 1 content",
		"show.s03e02.srt": "Episode 2 content",
	})

	// Create test HTTP server that counts requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	// Create downloader
	downloader := NewSubtitleDownloader(server.Client())

	// First request - should hit the server
	result1, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(1),
	)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("Expected 1 request after first download, got %d", requestCount)
	}

	// Second request for same URL but different episode - should use cache
	result2, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(2),
	)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("Expected 1 request after second download (should use cache), got %d", requestCount)
	}

	// Verify different episodes were extracted
	if result1.Filename == result2.Filename {
		t.Error("Expected different filenames for different episodes")
	}
	if string(result1.Content) == string(result2.Content) {
		t.Error("Expected different content for different episodes")
	}
}

func TestDownloadSubtitle_HTTPError(t *testing.T) {
	t.Parallel()
	// Create test HTTP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create downloader
	downloader := NewSubtitleDownloader(server.Client())

	// Test download
	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		nil,
	)

	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}

	if !errors.Is(err, &apperrors.ErrSubtitleResourceNotFound{}) {
		t.Errorf("Expected errors.Is to match ErrSubtitleResourceNotFound, got: %v", err)
	}
}

func TestDownloadSubtitle_InvalidZip(t *testing.T) {
	t.Parallel()
	// Create test HTTP server with invalid ZIP content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("This is not a valid ZIP file"))
	}))
	defer server.Close()

	// Create downloader
	downloader := NewSubtitleDownloader(server.Client())

	// Test download with episode extraction from invalid ZIP
	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(1),
	)

	if err == nil {
		t.Fatal("Expected error for invalid ZIP, got nil")
	}

	if !strings.Contains(err.Error(), "ZIP") {
		t.Errorf("Expected error message to mention ZIP, got: %v", err)
	}
}

func TestDownloadSubtitle_ContextCancellation(t *testing.T) {
	t.Parallel()
	// Create test HTTP server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/x-subrip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	// Create downloader
	downloader := NewSubtitleDownloader(server.Client())

	// Create context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test download with cancelled context
	_, err := downloader.DownloadSubtitle(
		ctx,
		buildDownloadURL(server.URL, "123456789"),
		nil,
	)

	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}
}

func BenchmarkDownloadSubtitle_ExtractFromZip(b *testing.B) {
	// Create large season pack
	zipFiles := make(map[string]string)
	for i := 1; i <= 20; i++ {
		filename := fmt.Sprintf("show.s03e%02d.srt", i)
		zipFiles[filename] = strings.Repeat("Subtitle content line\n", 100)
	}

	// Create ZIP using a simple inline implementation for benchmarks
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for filename, content := range zipFiles {
		f, err := w.Create(filename)
		if err != nil {
			b.Fatalf("Failed to create file %s in ZIP: %v", filename, err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			b.Fatalf("Failed to write content to %s in ZIP: %v", filename, err)
		}
	}
	if err := w.Close(); err != nil {
		b.Fatalf("Failed to close ZIP writer: %v", err)
	}
	zipContent := buf.Bytes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate between different episodes to test cache
		episode := (i % 20) + 1
		_, err := downloader.DownloadSubtitle(
			context.Background(),
			buildDownloadURL(server.URL, "123456789"),
			testutil.IntPtr(episode),
		)
		if err != nil {
			b.Fatalf("Download failed: %v", err)
		}
	}
}

func TestDownloadSubtitle_DifferentFileTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		contentType         string
		expectedFilename    string
		expectedContentType string
	}{
		{
			name:                "SRT file",
			contentType:         "application/x-subrip",
			expectedFilename:    "123456789.srt",
			expectedContentType: "application/x-subrip",
		},
		{
			name:                "ZIP file",
			contentType:         "application/zip",
			expectedFilename:    "123456789.zip",
			expectedContentType: "application/zip",
		},
		{
			name:                "ASS file",
			contentType:         "application/x-ass",
			expectedFilename:    "123456789.ass",
			expectedContentType: "application/x-ass",
		},
		{
			name:                "VTT file",
			contentType:         "text/vtt",
			expectedFilename:    "123456789.vtt",
			expectedContentType: "text/vtt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := "Test content"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(content))
			}))
			defer server.Close()

			downloader := NewSubtitleDownloader(server.Client())

			result, err := downloader.DownloadSubtitle(
				context.Background(),
				buildDownloadURL(server.URL, "123456789"),
				nil,
			)

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result.Filename != tt.expectedFilename {
				t.Errorf("Expected filename '%s', got '%s'", tt.expectedFilename, result.Filename)
			}

			if result.ContentType != tt.expectedContentType {
				t.Errorf("Expected content type '%s', got '%s'", tt.expectedContentType, result.ContentType)
			}

			if string(result.Content) != content {
				t.Errorf("Expected content '%s', got '%s'", content, string(result.Content))
			}
		})
	}
}

func TestExtractEpisodeFromZip_DifferentFileTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		filename            string
		expectedContentType string
	}{
		{
			name:                "SRT file",
			filename:            "show.s03e01.srt",
			expectedContentType: "application/x-subrip",
		},
		{
			name:                "ASS file",
			filename:            "show.s03e01.ass",
			expectedContentType: "application/x-ass",
		},
		{
			name:                "VTT file",
			filename:            "show.s03e01.vtt",
			expectedContentType: "text/vtt",
		},
		{
			name:                "SUB file",
			filename:            "show.s03e01.sub",
			expectedContentType: "application/x-sub",
		},
		{
			name:                "Unknown file type",
			filename:            "show.s03e01.xyz",
			expectedContentType: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			zipContent := createTestZip(t, map[string]string{
				tt.filename: "Test content",
			})

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/zip")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(zipContent)
			}))
			defer server.Close()

			downloader := NewSubtitleDownloader(server.Client())

			result, err := downloader.DownloadSubtitle(
				context.Background(),
				buildDownloadURL(server.URL, "123456789"),
				testutil.IntPtr(1),
			)

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result.ContentType != tt.expectedContentType {
				t.Errorf("Expected content type '%s', got '%s'", tt.expectedContentType, result.ContentType)
			}

			if result.Filename != tt.filename {
				t.Errorf("Expected filename '%s', got '%s'", tt.filename, result.Filename)
			}
		})
	}
}

func TestGetExtensionFromContentType_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{
			name:        "x-subrip takes precedence over generic srt",
			contentType: "application/x-subrip",
			expected:    ".srt",
		},
		{
			name:        "generic srt fallback",
			contentType: "text/srt",
			expected:    ".srt",
		},
		{
			name:        "x-ass specific",
			contentType: "application/x-ass",
			expected:    ".ass",
		},
		{
			name:        "slash-ass pattern",
			contentType: "text/ass",
			expected:    ".ass",
		},
		{
			name:        "x-sub specific",
			contentType: "application/x-sub",
			expected:    ".sub",
		},
		{
			name:        "unknown type defaults to srt",
			contentType: "application/octet-stream",
			expected:    ".srt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getExtensionFromContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("Expected extension '%s', got '%s' for content type '%s'", tt.expected, result, tt.contentType)
			}
		})
	}
}

func TestIsZipFile_MagicNumber(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Standard ZIP magic number",
			content:  []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Empty ZIP magic number",
			content:  []byte{0x50, 0x4B, 0x05, 0x06, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Spanned ZIP magic number",
			content:  []byte{0x50, 0x4B, 0x07, 0x08, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Not a ZIP file - gzip",
			content:  []byte{0x1F, 0x8B, 0x08, 0x00},
			expected: false,
		},
		{
			name:     "Not a ZIP file - random data",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			expected: false,
		},
		{
			name:     "Too short",
			content:  []byte{0x50, 0x4B},
			expected: false,
		},
		{
			name:     "Empty",
			content:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isZipFile(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for content %v", tt.expected, result, tt.content)
			}
		})
	}
}

func TestIsZipContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "application/zip",
			contentType: "application/zip",
			expected:    true,
		},
		{
			name:        "application/x-zip-compressed",
			contentType: "application/x-zip-compressed",
			expected:    true,
		},
		{
			name:        "application/zip with charset",
			contentType: "application/zip; charset=utf-8",
			expected:    true,
		},
		{
			name:        "Application/ZIP (uppercase)",
			contentType: "Application/ZIP",
			expected:    true,
		},
		{
			name:        "application/gzip - should NOT match",
			contentType: "application/gzip",
			expected:    false,
		},
		{
			name:        "application/x-gzip - should NOT match",
			contentType: "application/x-gzip",
			expected:    false,
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "application/octet-stream",
			contentType: "application/octet-stream",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isZipContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for content type '%s'", tt.expected, result, tt.contentType)
			}
		})
	}
}

func TestGetExtensionFromContentType_GzipEdgeCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{
			name:        "application/zip",
			contentType: "application/zip",
			expected:    ".zip",
		},
		{
			name:        "application/gzip should NOT return .zip",
			contentType: "application/gzip",
			expected:    ".srt", // defaults to .srt
		},
		{
			name:        "application/x-gzip should NOT return .zip",
			contentType: "application/x-gzip",
			expected:    ".srt", // defaults to .srt
		},
		{
			name:        "application/zip with parameters",
			contentType: "application/zip; charset=binary",
			expected:    ".zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getExtensionFromContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("Expected extension '%s', got '%s' for content type '%s'", tt.expected, result, tt.contentType)
			}
		})
	}
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
			name: "File exceeds individual size limit - should fail",
			files: map[string]string{
				"malicious.srt": strings.Repeat("X", 25*1024*1024), // 25 MB > 20 MB limit
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			zipContent := createTestZip(t, tt.files)
			err := detectZipBomb(zipContent)

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

func TestExtractEpisodeFromZip_ZipBombProtection(t *testing.T) {
	t.Parallel()
	// Create a ZIP with a file that exceeds size limits
	zipContent := createTestZip(t, map[string]string{
		"malicious.s03e01.srt": strings.Repeat("Q", 25*1024*1024), // 25 MB (> 20 MB limit)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	// Attempt to extract episode - should fail due to ZIP bomb detection
	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(1),
	)

	if err == nil {
		t.Fatal("Expected error due to ZIP bomb detection, got nil")
	}

	if !strings.Contains(err.Error(), "ZIP bomb detected") {
		t.Errorf("Expected 'ZIP bomb detected' error, got: %v", err)
	}
}

func TestDetectZipBomb_CompressionRatio(t *testing.T) {
	t.Parallel()
	// Create a small test to verify compression ratio check works
	// Note: In practice, creating a true high-compression-ratio ZIP is complex
	// This test verifies the function handles normal files correctly
	normalFiles := map[string]string{
		"test1.srt": "Normal content that compresses well but not suspiciously\n",
		"test2.srt": "Another normal file with typical subtitle content\n",
	}

	zipContent := createTestZip(t, normalFiles)
	err := detectZipBomb(zipContent)

	if err != nil {
		t.Errorf("Normal files should not trigger ZIP bomb detection, got: %v", err)
	}
}

func TestDownloadSubtitle_NestedFolderStructure(t *testing.T) {
	t.Parallel()
	// Create ZIP with nested folder structure matching real-world season packs
	// Structure: ShowName.S03E01/English.srt, ShowName.S03E02/English.srt, etc.
	zipFiles := map[string]string{
		"Hightown.S03E01/Hightown.S03E01.Good.Times.NF.WEB-DL.en.srt":      "Episode 1 content",
		"Hightown.S03E02/Hightown.S03E02.I.Said.No.No.No.NF.WEB-DL.en.srt": "Episode 2 content",
		"Hightown.S03E03/Hightown.S03E03.Fall.Brook.NF.WEB-DL.en.srt":      "Episode 3 content",
	}
	zipContent := createTestZip(t, zipFiles)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	// Request episode 2 - should match the folder name "Hightown.S03E02"
	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(2),
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}

	// Verify we got episode 2 content
	if string(result.Content) != "Episode 2 content" {
		t.Errorf("Expected 'Episode 2 content', got: %s", string(result.Content))
	}

	// Verify filename is from the matched file
	if !strings.Contains(result.Filename, "S03E02") {
		t.Errorf("Expected filename to contain S03E02, got: %s", result.Filename)
	}

	// Verify content type
	if result.ContentType != "application/x-subrip" {
		t.Errorf("Expected content type 'application/x-subrip', got: %s", result.ContentType)
	}
}

func TestDownloadSubtitle_ExceedsDownloadSizeLimit(t *testing.T) {
	t.Parallel()
	// Create a server that returns a very large response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		// Write more than maxDownloadSize (150 MB)
		// Write in chunks to avoid memory issues in test
		chunk := make([]byte, 1024*1024) // 1 MB chunks
		for i := 0; i < 151; i++ {
			_, _ = w.Write(chunk)
		}
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	// Test download that exceeds size limit
	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		nil,
	)

	if err == nil {
		t.Fatal("Expected error for oversized download, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected error message about size limit, got: %v", err)
	}
}

func TestExtractEpisodeFromZip_MultipleMatches(t *testing.T) {
	t.Parallel()
	// Create ZIP with multiple files matching the same episode
	// Including both subtitle files and non-subtitle files
	zipFiles := map[string]string{
		"show.s03e01.nfo":     "NFO file content",
		"show.s03e01.sub":     "SUB subtitle content",
		"show.s03e01.ass":     "ASS subtitle content",
		"show.s03e01.srt":     "SRT subtitle content", // Should be preferred
		"show.s03e01.txt":     "Text file content",
		"show.s03e01.vtt":     "VTT subtitle content",
		"show.s03e01.unknown": "Unknown file content",
	}
	zipContent := createTestZip(t, zipFiles)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	// Request episode 1 - should prefer .srt over other types
	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(1),
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
		return
	}

	// Verify we got the .srt file (highest priority)
	if string(result.Content) != "SRT subtitle content" {
		t.Errorf("Expected SRT content, got: %s", string(result.Content))
	}

	// Verify filename
	if !strings.HasSuffix(result.Filename, ".srt") {
		t.Errorf("Expected .srt filename, got: %s", result.Filename)
	}

	// Verify content type
	if result.ContentType != "application/x-subrip" {
		t.Errorf("Expected content type 'application/x-subrip', got: %s", result.ContentType)
	}
}

func TestExtractEpisodeFromZip_PreferSubtitleOverNonSubtitle(t *testing.T) {
	t.Parallel()
	// Create ZIP with subtitle and non-subtitle files for the same episode
	zipFiles := map[string]string{
		"show.s03e02.nfo": "NFO file content",
		"show.s03e02.txt": "Text file content",
		"show.s03e02.ass": "ASS subtitle content", // Should be selected over non-subtitle files
	}
	zipContent := createTestZip(t, zipFiles)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(2),
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify we got the .ass file (subtitle type preferred over non-subtitle)
	if string(result.Content) != "ASS subtitle content" {
		t.Errorf("Expected ASS content, got: %s", string(result.Content))
	}

	if !strings.HasSuffix(result.Filename, ".ass") {
		t.Errorf("Expected .ass filename, got: %s", result.Filename)
	}
}

func TestResolveCacheConfig_Defaults_NilConfig(t *testing.T) {
	t.Parallel()
	size, ttl := resolveCacheConfig(nil)
	if size != 2000 {
		t.Errorf("Expected default size 2000, got %d", size)
	}
	if ttl != 24*time.Hour {
		t.Errorf("Expected default TTL 24h, got %v", ttl)
	}
}

func TestResolveCacheConfig_ValidValues(t *testing.T) {
	t.Parallel()
	cfg := &internalConfig.Config{}
	cfg.Cache.Size = 500
	cfg.Cache.TTL = "6h"

	size, ttl := resolveCacheConfig(cfg)
	if size != 500 {
		t.Errorf("Expected size 500, got %d", size)
	}
	if ttl != 6*time.Hour {
		t.Errorf("Expected TTL 6h, got %v", ttl)
	}
}

func TestResolveCacheConfig_ZeroSize_UsesDefault(t *testing.T) {
	t.Parallel()
	cfg := &internalConfig.Config{}
	cfg.Cache.Size = 0
	cfg.Cache.TTL = "12h"

	size, ttl := resolveCacheConfig(cfg)
	if size != 2000 {
		t.Errorf("Expected default size 2000, got %d", size)
	}
	if ttl != 12*time.Hour {
		t.Errorf("Expected TTL 12h, got %v", ttl)
	}
}

func TestResolveCacheConfig_EmptyTTL_UsesDefault(t *testing.T) {
	t.Parallel()
	cfg := &internalConfig.Config{}
	cfg.Cache.Size = 100
	cfg.Cache.TTL = ""

	size, ttl := resolveCacheConfig(cfg)
	if size != 100 {
		t.Errorf("Expected size 100, got %d", size)
	}
	if ttl != 24*time.Hour {
		t.Errorf("Expected default TTL 24h, got %v", ttl)
	}
}

func TestResolveCacheConfig_InvalidTTL_UsesDefault(t *testing.T) {
	t.Parallel()
	cfg := &internalConfig.Config{}
	cfg.Cache.Size = 300
	cfg.Cache.TTL = "24hours" // invalid Go duration

	size, ttl := resolveCacheConfig(cfg)
	if size != 300 {
		t.Errorf("Expected size 300, got %d", size)
	}
	if ttl != 24*time.Hour {
		t.Errorf("Expected default TTL 24h on invalid input, got %v", ttl)
	}
}

// TestExtractEpisodeFromZip_InvalidUTF8Filename tests that filenames with invalid UTF-8 bytes
// (e.g., from non-UTF-8 encoded ZIP entries) are sanitized with the replacement character.
func TestExtractEpisodeFromZip_InvalidUTF8Filename(t *testing.T) {
	t.Parallel()
	// Simulate a ZIP where the filename contains invalid UTF-8 bytes,
	// like "Pokémon" encoded in ISO-8859-1 (é = 0xE9) instead of UTF-8 (é = 0xC3 0xA9)
	// This produces "Pok\xe9mon" which is not valid UTF-8.
	invalidFilename := "Pok\xe9mon.the.Series_.XYZ.S01E01.WEBRip.Netflix.en[cc].srt"
	subtitleContent := "1\n00:00:01,000 --> 00:00:02,000\nTest subtitle\n"

	zipContent := createTestZip(t, map[string]string{
		invalidFilename: subtitleContent,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(1),
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The filename should have the invalid byte replaced with the Unicode replacement character
	expectedFilename := "Pok�mon.the.Series_.XYZ.S01E01.WEBRip.Netflix.en[cc].srt"
	if result.Filename != expectedFilename {
		t.Errorf("Expected sanitized filename %q, got %q", expectedFilename, result.Filename)
	}

	if result.ContentType != "application/x-subrip" {
		t.Errorf("Expected content type 'application/x-subrip', got %q", result.ContentType)
	}

	if len(result.Content) == 0 {
		t.Error("Expected non-empty content")
	}
}

// TestExtractEpisodeFromZip_MultipleInvalidUTF8Filenames tests that the correct episode is
// matched even when multiple files have invalid UTF-8 filenames.
func TestExtractEpisodeFromZip_MultipleInvalidUTF8Filenames(t *testing.T) {
	t.Parallel()
	ep1Content := "1\n00:00:01,000 --> 00:00:02,000\nEpisode 1\n"
	ep2Content := "1\n00:00:01,000 --> 00:00:02,000\nEpisode 2\n"

	// Both filenames have invalid UTF-8 (ISO-8859-1 encoded accented chars)
	zipContent := createTestZip(t, map[string]string{
		"Pok\xe9mon.S01E01.srt": ep1Content,
		"Pok\xe9mon.S01E02.srt": ep2Content,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	// Request episode 2
	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(2),
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedFilename := "Pok�mon.S01E02.srt"
	if result.Filename != expectedFilename {
		t.Errorf("Expected sanitized filename %q, got %q", expectedFilename, result.Filename)
	}

	if string(result.Content) != ep2Content {
		t.Errorf("Expected episode 2 content, got %q", string(result.Content))
	}
}

// TestExtractEpisodeFromZip_Windows1252Filename tests filenames with Windows-1252 specific bytes
// that are invalid in UTF-8 (0x80-0x9F range).
func TestExtractEpisodeFromZip_Windows1252Filename(t *testing.T) {
	t.Parallel()
	// 0x93 and 0x94 are "smart quotes" in Windows-1252, invalid in UTF-8
	invalidFilename := "Show\x93s.Name\x94.S01E01.srt"
	subtitleContent := "1\n00:00:01,000 --> 00:00:02,000\nTest\n"

	zipContent := createTestZip(t, map[string]string{
		invalidFilename: subtitleContent,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	result, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		testutil.IntPtr(1),
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The invalid bytes should be replaced with replacement characters
	expectedFilename := "Show�s.Name�.S01E01.srt"
	if result.Filename != expectedFilename {
		t.Errorf("Expected sanitized filename %q, got %q", expectedFilename, result.Filename)
	}
}

// TestConvertToUTF8_ISO88591Content tests that subtitle content in ISO-8859-1 is converted to UTF-8
func TestConvertToUTF8_ISO88591Content(t *testing.T) {
	t.Parallel()
	// SRT content with ISO-8859-1 encoded "é" (0xE9)
	iso88591Content := []byte("1\r\n00:00:01,000 --> 00:00:02,000\r\nCaf\xe9\r\n")

	result := convertToUTF8(iso88591Content)

	resultStr := string(result)
	if !strings.Contains(resultStr, "Café") {
		t.Errorf("Expected converted content to contain 'Café', got %q", resultStr)
	}
}

// TestConvertToUTF8_AlreadyUTF8 tests that valid UTF-8 content passes through unchanged
func TestConvertToUTF8_AlreadyUTF8(t *testing.T) {
	t.Parallel()
	utf8Content := []byte("1\r\n00:00:01,000 --> 00:00:02,000\r\nCafé\r\n")

	result := convertToUTF8(utf8Content)

	if !bytes.Equal(result, utf8Content) {
		t.Errorf("Expected UTF-8 content to pass through unchanged")
	}
}

// TestConvertToUTF8_EmptyContent tests that empty content is handled
func TestConvertToUTF8_EmptyContent(t *testing.T) {
	t.Parallel()
	result := convertToUTF8([]byte{})
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d bytes", len(result))
	}
}

// TestIsTextSubtitleContentType tests the content type detection for text subtitles
func TestIsTextSubtitleContentType(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		contentType string
		expected    bool
	}{
		{"application/x-subrip", true},
		{"application/x-ass", true},
		{"text/ass", true},
		{"text/vtt", true},
		{"text/webvtt", true},
		{"application/x-sub", true},
		{"text/plain", true},
		{"application/zip", false},
		{"application/octet-stream", false},
		{"application/x-subrip; charset=utf-8", true},
	}

	for _, tc := range testCases {
		t.Run(tc.contentType, func(t *testing.T) {
			t.Parallel()
			result := isTextSubtitleContentType(tc.contentType)
			if result != tc.expected {
				t.Errorf("isTextSubtitleContentType(%q) = %v, want %v", tc.contentType, result, tc.expected)
			}
		})
	}
}

// Metric helper functions for integration tests

func getCounterVecValue(cv *prometheus.CounterVec, labels ...string) float64 {
	c, err := cv.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

func TestDownloadSubtitle_Metrics_SuccessIncrement(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,000\nTest subtitle\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-subrip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	before := getCounterVecValue(metrics.SubtitleDownloadsTotal, "success")

	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		nil,
	)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	after := getCounterVecValue(metrics.SubtitleDownloadsTotal, "success")
	if after != before+1 {
		t.Errorf("Expected success counter to increment by 1, got diff %.0f", after-before)
	}
}

func TestDownloadSubtitle_Metrics_ErrorIncrement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	before := getCounterVecValue(metrics.SubtitleDownloadsTotal, "error")

	_, _ = downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "123456789"),
		nil,
	)

	after := getCounterVecValue(metrics.SubtitleDownloadsTotal, "error")
	if after != before+1 {
		t.Errorf("Expected error counter to increment by 1, got diff %.0f", after-before)
	}
}

func TestDownloadSubtitle_Metrics_CacheHitMiss(t *testing.T) {
	zipContent := createTestZip(t, map[string]string{
		"show.s03e01.srt": "Episode 1 content",
		"show.s03e02.srt": "Episode 2 content",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	missBeforeFirst := getCounterVecValue(cache.MissesTotal, "zip")
	hitBeforeFirst := getCounterVecValue(cache.HitsTotal, "zip")

	// First request — cache miss
	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "metrics-test"),
		testutil.IntPtr(1),
	)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	missAfterFirst := getCounterVecValue(cache.MissesTotal, "zip")
	if missAfterFirst != missBeforeFirst+1 {
		t.Errorf("Expected cache misses to increment by 1, got diff %.0f", missAfterFirst-missBeforeFirst)
	}

	// Second request — cache hit
	_, err = downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "metrics-test"),
		testutil.IntPtr(2),
	)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}

	hitAfterSecond := getCounterVecValue(cache.HitsTotal, "zip")
	if hitAfterSecond != hitBeforeFirst+1 {
		t.Errorf("Expected cache hits to increment by 1, got diff %.0f", hitAfterSecond-hitBeforeFirst)
	}
}

func TestDownloadSubtitle_Metrics_CacheEntriesGauge(t *testing.T) {
	zipContent := createTestZip(t, map[string]string{
		"show.s03e01.srt": "Episode 1 content",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())
	d, ok := downloader.(*DefaultSubtitleDownloader)
	if !ok {
		t.Fatalf("NewSubtitleDownloader returned %T, want *DefaultSubtitleDownloader", downloader)
	}

	if d.zipCache.Len() != 0 {
		t.Fatalf("Expected 0 cache entries before download, got %d", d.zipCache.Len())
	}
	if v := gatherCacheEntriesMetric("zip"); v != 0 {
		t.Fatalf("Expected cache_entries{cache=\"zip\"} == 0 before download, got %.0f", v)
	}

	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "gauge-test-unique"),
		testutil.IntPtr(1),
	)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if d.zipCache.Len() != 1 {
		t.Errorf("Expected 1 cache entry after download, got %d", d.zipCache.Len())
	}
	if v := gatherCacheEntriesMetric("zip"); v != 1 {
		t.Errorf("Expected cache_entries{cache=\"zip\"} == 1 after download, got %.0f", v)
	}
}

// gatherCacheEntriesMetric reads the current value of cache_entries{cache=group}
// from the default Prometheus registry, returning -1 if the metric is not found.
func gatherCacheEntriesMetric(group string) float64 {
	mfs, _ := prometheus.DefaultGatherer.Gather()
	for _, mf := range mfs {
		if mf.GetName() != "cache_entries" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "cache" && lp.GetValue() == group {
					return m.GetGauge().GetValue()
				}
			}
		}
	}
	return -1
}

func TestDownloadSubtitle_Metrics_ZipEpisodeExtractionSuccess(t *testing.T) {
	zipContent := createTestZip(t, map[string]string{
		"show.s03e01.srt": "Episode 1 content",
		"show.s03e02.srt": "Episode 2 content",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	before := getCounterVecValue(metrics.SubtitleDownloadsTotal, "success")

	_, err := downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "zip-success-test"),
		testutil.IntPtr(1),
	)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	after := getCounterVecValue(metrics.SubtitleDownloadsTotal, "success")
	if after != before+1 {
		t.Errorf("Expected success counter to increment by 1 for ZIP extraction, got diff %.0f", after-before)
	}
}

func TestDownloadSubtitle_Metrics_ZipEpisodeExtractionError(t *testing.T) {
	zipContent := createTestZip(t, map[string]string{
		"show.s03e01.srt": "Episode 1 content",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipContent)
	}))
	defer server.Close()

	downloader := NewSubtitleDownloader(server.Client())

	before := getCounterVecValue(metrics.SubtitleDownloadsTotal, "error")

	// Request non-existent episode
	_, _ = downloader.DownloadSubtitle(
		context.Background(),
		buildDownloadURL(server.URL, "zip-error-test"),
		testutil.IntPtr(99),
	)

	after := getCounterVecValue(metrics.SubtitleDownloadsTotal, "error")
	if after != before+1 {
		t.Errorf("Expected error counter to increment by 1 for failed ZIP extraction, got diff %.0f", after-before)
	}
}
