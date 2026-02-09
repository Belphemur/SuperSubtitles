package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/testutil"
)

func TestClient_GetRecentSubtitles(t *testing.T) {
	// Track requests to detail pages
	var detailRequests sync.Map

	// Create a test server that serves main page and detail pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			// Main page with recent subtitles
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{
					SubtitleID: "1770600001",
					Title:      "Recent Subtitle 1",
					FileName:   "recent1.srt",
					Season:     1,
					Episode:    1,
					ShowName:   "Test Show 1",
					ShowID:     123,
				},
				{
					SubtitleID: "1770600002",
					Title:      "Recent Subtitle 2",
					FileName:   "recent2.srt",
					Season:     1,
					Episode:    2,
					ShowName:   "Test Show 1",
					ShowID:     123,
				},
				{
					SubtitleID: "1770600003",
					Title:      "Recent Subtitle 3",
					FileName:   "recent3.srt",
					Season:     1,
					Episode:    1,
					ShowName:   "Test Show 2",
					ShowID:     456,
				},
			})
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
		} else if r.URL.Query().Get("tipus") == "adatlap" {
			// Detail page with third-party IDs
			azon := r.URL.Query().Get("azon")
			detailRequests.Store(azon, true)

			html := `<html><body><div class="adatlapTabla"><div class="adatlapAdat"><div class="adatlapRow">
				<a href="http://www.imdb.com/title/tt1234567/" target="_blank" alt="iMDB"></a>
				<a href="http://thetvdb.com/?tab=series&id=987654" target="_blank" alt="TheTVDB"></a>
				</div></div></div></body></html>`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
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
	showSubtitles, err := client.GetRecentSubtitles(ctx, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify detail pages were fetched
	if _, ok := detailRequests.Load("a_1770600001"); !ok {
		t.Error("Expected detail page request for Show 1")
	}

	// Verify show data
	if len(showSubtitles) != 2 {
		t.Fatalf("Expected 2 shows, got %d", len(showSubtitles))
	}

	// Verify subtitles are grouped correctly
	for _, ss := range showSubtitles {
		if ss.Show.ID == 123 && len(ss.SubtitleCollection.Subtitles) != 2 {
			t.Errorf("Expected 2 subtitles for show 123, got %d", len(ss.SubtitleCollection.Subtitles))
		}
		if ss.Show.ID == 456 && len(ss.SubtitleCollection.Subtitles) != 1 {
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
					SubtitleID: "1770500000",
					Title:      "Old Subtitle",
					FileName:   "old.srt",
					Season:     1,
					Episode:    1,
					ShowName:   "Test Show",
					ShowID:     123,
				},
				{
					SubtitleID: "1770617276",
					Title:      "New Subtitle",
					FileName:   "new.srt",
					Season:     1,
					Episode:    2,
					ShowName:   "Test Show",
					ShowID:     123,
				},
			})
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
		} else if r.URL.Query().Get("tipus") == "adatlap" {
			html := `<html><body><div class="adatlapTabla"></div></body></html>`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
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
	showSubtitles, err := client.GetRecentSubtitles(ctx, "1770600000")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only return the subtitle with ID 1770617276
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(showSubtitles))
	}
	if len(showSubtitles[0].SubtitleCollection.Subtitles) != 1 {
		t.Errorf("Expected 1 subtitle, got %d", len(showSubtitles[0].SubtitleCollection.Subtitles))
	}
	if showSubtitles[0].SubtitleCollection.Subtitles[0].ID != "1770617276" {
		t.Errorf("Expected subtitle ID 1770617276, got %s", showSubtitles[0].SubtitleCollection.Subtitles[0].ID)
	}
}

func TestClient_GetRecentSubtitles_EmptyResult(t *testing.T) {
	// Create a test server that returns empty main page
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := "<html><body><table></table></body></html>"
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
		}
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	client := NewClient(testConfig)
	ctx := context.Background()

	showSubtitles, err := client.GetRecentSubtitles(ctx, "")
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

	_, err := client.GetRecentSubtitles(ctx, "")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}
