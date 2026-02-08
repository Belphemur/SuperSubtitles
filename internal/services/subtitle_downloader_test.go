package services

import (
	"SuperSubtitles/internal/models"
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestDownloadSubtitle_NonZipFile(t *testing.T) {
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
	result, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
		SubtitleID: "123456789",
		Episode:    0,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
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
	result, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
		SubtitleID: "123456789",
		Episode:    0,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !bytes.Equal(result.Content, zipContent) {
		t.Error("Expected ZIP content to be returned as-is")
	}

	if result.ContentType != "application/zip" {
		t.Errorf("Expected content type 'application/zip', got '%s'", result.ContentType)
	}
}

func TestDownloadSubtitle_ExtractEpisodeFromZip(t *testing.T) {
	tests := []struct {
		name            string
		zipFiles        map[string]string
		requestEpisode  int
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
			requestEpisode:  1,
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
			requestEpisode:  2,
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
			requestEpisode:  5,
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
			requestEpisode:  7,
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
			requestEpisode:  1,
			expectedFile:    "Hightown.S03E01.Good.Times.NF.WEB-DL.en.srt",
			expectedContent: "Episode 1 content",
			shouldFail:      false,
		},
		{
			name: "Episode not found in ZIP",
			zipFiles: map[string]string{
				"show.s03e01.srt": "Episode 1 content",
				"show.s03e02.srt": "Episode 2 content",
			},
			requestEpisode: 10,
			shouldFail:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			result, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
				SubtitleID: "123456789",
				Episode:    tt.requestEpisode,
			})

			if tt.shouldFail {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), "not found") {
					t.Errorf("Expected 'not found' error, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
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
	result1, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
		SubtitleID: "123456789",
		Episode:    1,
	})
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("Expected 1 request after first download, got %d", requestCount)
	}

	// Second request for same URL but different episode - should use cache
	result2, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
		SubtitleID: "123456789",
		Episode:    2,
	})
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
	// Create test HTTP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create downloader
	downloader := NewSubtitleDownloader(server.Client())

	// Test download
	_, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
		SubtitleID: "123456789",
		Episode:    0,
	})

	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error message to contain '404', got: %v", err)
	}
}

func TestDownloadSubtitle_InvalidZip(t *testing.T) {
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
	_, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
		SubtitleID: "123456789",
		Episode:    1,
	})

	if err == nil {
		t.Fatal("Expected error for invalid ZIP, got nil")
	}

	if !strings.Contains(err.Error(), "ZIP") {
		t.Errorf("Expected error message to mention ZIP, got: %v", err)
	}
}

func TestDownloadSubtitle_ContextCancellation(t *testing.T) {
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
	_, err := downloader.DownloadSubtitle(ctx, server.URL, models.DownloadRequest{
		SubtitleID: "123456789",
		Episode:    0,
	})

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

	zipContent := createTestZip(&testing.T{}, zipFiles)

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
		_, err := downloader.DownloadSubtitle(context.Background(), server.URL, models.DownloadRequest{
			SubtitleID: "123456789",
			Episode:    episode,
		})
		if err != nil {
			b.Fatalf("Download failed: %v", err)
		}
	}
}
