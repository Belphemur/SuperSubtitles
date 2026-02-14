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
	// Create a test server that serves main page and detail pages
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
		} else if r.URL.Query().Get("tipus") == "adatlap" {
			// Detail page with third-party IDs
			html := testutil.GenerateThirdPartyIDHTML("tt1234567", 987654, 0, 0)
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
	showSubtitles, err := client.GetRecentSubtitles(ctx, 0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return 2 shows
	if len(showSubtitles) != 2 {
		t.Fatalf("Expected 2 shows, got %d", len(showSubtitles))
	}

	// Verify show names are included in ShowInfo
	for _, ss := range showSubtitles {
		if ss.Name == "" {
			t.Errorf("Expected non-empty show name for show ID %d", ss.ID)
		}
		if ss.ID == 123 && len(ss.SubtitleCollection.Subtitles) != 2 {
			t.Errorf("Expected 2 subtitles for show 123, got %d", len(ss.SubtitleCollection.Subtitles))
		}
		if ss.ID == 456 && len(ss.SubtitleCollection.Subtitles) != 1 {
			t.Errorf("Expected 1 subtitle for show 456, got %d", len(ss.SubtitleCollection.Subtitles))
		}
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
		} else if r.URL.Query().Get("tipus") == "adatlap" {
			html := testutil.GenerateThirdPartyIDHTML("", 0, 0, 0)
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
	showSubtitles, err := client.GetRecentSubtitles(ctx, 1770600000)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only return the subtitle with ID 1770617276 for 1 show
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(showSubtitles))
	}
	if len(showSubtitles[0].SubtitleCollection.Subtitles) != 1 {
		t.Errorf("Expected 1 subtitle, got %d", len(showSubtitles[0].SubtitleCollection.Subtitles))
	}
	if showSubtitles[0].SubtitleCollection.Subtitles[0].ID != 1770617276 {
		t.Errorf("Expected subtitle ID 1770617276, got %d", showSubtitles[0].SubtitleCollection.Subtitles[0].ID)
	}
}

func TestClient_GetRecentSubtitles_EmptyResult(t *testing.T) {
	// Create a test server that returns empty main page
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := testutil.GenerateSubtitleTableHTML(nil)
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

	showSubtitles, err := client.GetRecentSubtitles(ctx, 0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(showSubtitles) != 0 {
		t.Errorf("Expected 0 shows, got %d", len(showSubtitles))
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

func TestClient_StreamRecentSubtitles_ShowInfoSentOncePerShow(t *testing.T) {
	// Verify that ShowInfo is only sent once per unique show_id
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{
					SubtitleID:       100001,
					MagyarTitle:      "Sub 1",
					EredetiTitle:     "Show A - 1x01",
					DownloadFilename: "sub1.srt",
					ShowID:           10,
				},
				{
					SubtitleID:       100002,
					MagyarTitle:      "Sub 2",
					EredetiTitle:     "Show A - 1x02",
					DownloadFilename: "sub2.srt",
					ShowID:           10,
				},
				{
					SubtitleID:       100003,
					MagyarTitle:      "Sub 3",
					EredetiTitle:     "Show B - 1x01",
					DownloadFilename: "sub3.srt",
					ShowID:           20,
				},
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		} else if r.URL.Query().Get("tipus") == "adatlap" {
			html := testutil.GenerateThirdPartyIDHTML("", 0, 0, 0)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		}
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	showInfoCount := 0
	subtitleCount := 0
	showInfoIDs := make(map[int]int)

	for item := range c.StreamRecentSubtitles(ctx, 0) {
		if item.Err != nil {
			t.Fatalf("Unexpected error: %v", item.Err)
		}
		if item.Value.ShowInfo != nil {
			showInfoCount++
			showInfoIDs[item.Value.ShowInfo.Show.ID]++
			// Verify show name is set
			if item.Value.ShowInfo.Show.Name == "" {
				t.Errorf("Expected non-empty show name for show ID %d", item.Value.ShowInfo.Show.ID)
			}
		}
		if item.Value.Subtitle != nil {
			subtitleCount++
		}
	}

	// Should have exactly 2 ShowInfo items (one per unique show)
	if showInfoCount != 2 {
		t.Errorf("Expected 2 ShowInfo items, got %d", showInfoCount)
	}

	// Each show should only have 1 ShowInfo
	for showID, count := range showInfoIDs {
		if count != 1 {
			t.Errorf("Expected 1 ShowInfo for show %d, got %d", showID, count)
		}
	}

	// Should have 3 subtitle items
	if subtitleCount != 3 {
		t.Errorf("Expected 3 subtitle items, got %d", subtitleCount)
	}
}
