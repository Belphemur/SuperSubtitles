package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/testutil"
)

func TestClient_GetRecentSubtitles(t *testing.T) {
	// Create a test server that serves main page
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			// Main page with recent subtitles
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{
					SubtitleID:       1770600001,
					MagyarTitle:      "Recent Subtitle 1",
					EredetiTitle:     "Test Show 1 - 1x01",
					DownloadFilename: "recent1.srt",
					ShowID:           123,
				},
				{
					SubtitleID:       1770600002,
					MagyarTitle:      "Recent Subtitle 2",
					EredetiTitle:     "Test Show 1 - 1x02",
					DownloadFilename: "recent2.srt",
					ShowID:           123,
				},
				{
					SubtitleID:       1770600003,
					MagyarTitle:      "Recent Subtitle 3",
					EredetiTitle:     "Test Show 2 - 1x01",
					DownloadFilename: "recent3.srt",
					ShowID:           456,
				},
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		}
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	client := NewClient(testConfig)
	ctx := context.Background()

	// Test without filter (all subtitles)
	subtitles, err := client.GetRecentSubtitles(ctx, 0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return all 3 subtitles
	if len(subtitles) != 3 {
		t.Fatalf("Expected 3 subtitles, got %d", len(subtitles))
	}

	// Verify subtitles contain show info
	showIDs := make(map[int]int)
	for _, sub := range subtitles {
		showIDs[sub.ShowID]++
	}
	if showIDs[123] != 2 {
		t.Errorf("Expected 2 subtitles for show 123, got %d", showIDs[123])
	}
	if showIDs[456] != 1 {
		t.Errorf("Expected 1 subtitle for show 456, got %d", showIDs[456])
	}
}

func TestClient_GetRecentSubtitles_WithFilter(t *testing.T) {
	// Create a test server that serves main page
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{
					SubtitleID:       1770500000,
					MagyarTitle:      "Old Subtitle",
					EredetiTitle:     "Test Show - 1x01",
					DownloadFilename: "old.srt",
					ShowID:           123,
				},
				{
					SubtitleID:       1770617276,
					MagyarTitle:      "New Subtitle",
					EredetiTitle:     "Test Show - 1x02",
					DownloadFilename: "new.srt",
					ShowID:           123,
				},
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		}
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	client := NewClient(testConfig)
	ctx := context.Background()

	// Test with filter (only subtitles with ID > 1770600000)
	subtitles, err := client.GetRecentSubtitles(ctx, 1770600000)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only return the subtitle with ID 1770617276
	if len(subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(subtitles))
	}
	if subtitles[0].ID != 1770617276 {
		t.Errorf("Expected subtitle ID 1770617276, got %d", subtitles[0].ID)
	}
}

func TestClient_GetRecentSubtitles_EmptyResult(t *testing.T) {
	// Create a test server that returns empty main page
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := "<html><body><table></table></body></html>"
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		}
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	client := NewClient(testConfig)
	ctx := context.Background()

	subtitles, err := client.GetRecentSubtitles(ctx, 0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(subtitles) != 0 {
		t.Errorf("Expected 0 subtitles, got %d", len(subtitles))
	}
}

func TestClient_GetRecentSubtitles_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	client := NewClient(testConfig)
	ctx := context.Background()

	_, err := client.GetRecentSubtitles(ctx, 0)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}
