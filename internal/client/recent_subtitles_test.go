package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
)

func TestClient_GetRecentSubtitles(t *testing.T) {
	t.Parallel()
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
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, client.StreamRecentSubtitles(ctx, 0))
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
	t.Parallel()
	// Create a test server that serves main page
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{
					SubtitleID:       1770617276,
					MagyarTitle:      "New Subtitle",
					EredetiTitle:     "Test Show - 1x02",
					DownloadFilename: "new.srt",
					ShowID:           123,
				},
				{
					SubtitleID:       1770500000,
					MagyarTitle:      "Old Subtitle",
					EredetiTitle:     "Test Show - 1x01",
					DownloadFilename: "old.srt",
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
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, client.StreamRecentSubtitles(ctx, 1770600000))
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
	t.Parallel()
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

	showSubtitles, err := testutil.CollectShowSubtitles(ctx, client.StreamRecentSubtitles(ctx, 0))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(showSubtitles) != 0 {
		t.Errorf("Expected 0 shows, got %d", len(showSubtitles))
	}
}

func TestClient_GetRecentSubtitles_ServerError(t *testing.T) {
	t.Parallel()
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

	_, err := testutil.CollectShowSubtitles(ctx, client.StreamRecentSubtitles(ctx, 0))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestClient_StreamRecentSubtitles_ShowInfoSentOncePerShow(t *testing.T) {
	t.Parallel()
	// Verify that each show appears exactly once with all its subtitles grouped together
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

	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamRecentSubtitles(ctx, 0))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have exactly 2 shows
	if len(showSubtitles) != 2 {
		t.Fatalf("Expected 2 shows, got %d", len(showSubtitles))
	}

	// Verify each show has the correct number of subtitles
	totalSubtitles := 0
	for _, ss := range showSubtitles {
		if ss.Name == "" {
			t.Errorf("Expected non-empty show name for show ID %d", ss.ID)
		}
		if ss.ID == 10 && len(ss.SubtitleCollection.Subtitles) != 2 {
			t.Errorf("Expected 2 subtitles for show 10, got %d", len(ss.SubtitleCollection.Subtitles))
		}
		if ss.ID == 20 && len(ss.SubtitleCollection.Subtitles) != 1 {
			t.Errorf("Expected 1 subtitle for show 20, got %d", len(ss.SubtitleCollection.Subtitles))
		}
		totalSubtitles += len(ss.SubtitleCollection.Subtitles)
	}

	// Should have 3 total subtitles
	if totalSubtitles != 3 {
		t.Errorf("Expected 3 total subtitles, got %d", totalSubtitles)
	}
}

func TestClient_StreamRecentSubtitles_Pagination(t *testing.T) {
	t.Parallel()
	// Page 1: IDs 5000, 4000 (all above sinceID 2500)
	// Page 2: IDs 3000, 2000 (2000 <= sinceID 2500, triggers boundary)
	// Page 3: should never be requested
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") != "sorozat" {
			if r.URL.Query().Get("tipus") == "adatlap" {
				html := testutil.GenerateThirdPartyIDHTML("tt999", 111, 0, 0)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(html))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		page := 1
		if p := r.URL.Query().Get("oldal"); p != "" {
			page, _ = strconv.Atoi(p)
		}

		totalPages := 3
		switch page {
		case 1:
			html := testutil.GenerateSubtitleTableHTMLWithPagination([]testutil.SubtitleRowOptions{
				{SubtitleID: 5000, MagyarTitle: "Sub A", EredetiTitle: "Show A - 1x01", DownloadFilename: "a.srt", ShowID: 10},
				{SubtitleID: 4000, MagyarTitle: "Sub B", EredetiTitle: "Show B - 1x01", DownloadFilename: "b.srt", ShowID: 20},
			}, page, totalPages, true)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		case 2:
			html := testutil.GenerateSubtitleTableHTMLWithPagination([]testutil.SubtitleRowOptions{
				{SubtitleID: 3000, MagyarTitle: "Sub C", EredetiTitle: "Show A - 1x02", DownloadFilename: "c.srt", ShowID: 10},
				{SubtitleID: 2000, MagyarTitle: "Sub D", EredetiTitle: "Show C - 1x01", DownloadFilename: "d.srt", ShowID: 30},
			}, page, totalPages, true)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		default:
			// Page 3 should never be fetched
			t.Errorf("Unexpected page %d requested", page)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(testutil.GenerateSubtitleTableHTML(nil)))
		}
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamRecentSubtitles(ctx, 2500))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have shows 10 (2 subs: 5000, 3000) and 20 (1 sub: 4000)
	// Sub 2000 (show 30) should be excluded (ID <= sinceID)
	if len(showSubtitles) != 2 {
		t.Fatalf("Expected 2 shows, got %d", len(showSubtitles))
	}

	totalSubs := 0
	for _, ss := range showSubtitles {
		switch ss.ID {
		case 10:
			if len(ss.SubtitleCollection.Subtitles) != 2 {
				t.Errorf("Expected 2 subtitles for show 10, got %d", len(ss.SubtitleCollection.Subtitles))
			}
		case 20:
			if len(ss.SubtitleCollection.Subtitles) != 1 {
				t.Errorf("Expected 1 subtitle for show 20, got %d", len(ss.SubtitleCollection.Subtitles))
			}
		default:
			t.Errorf("Unexpected show ID %d", ss.ID)
		}
		totalSubs += len(ss.SubtitleCollection.Subtitles)
	}
	if totalSubs != 3 {
		t.Errorf("Expected 3 total subtitles, got %d", totalSubs)
	}
}

func TestClient_StreamRecentSubtitles_PaginationStopsOnLastPage(t *testing.T) {
	t.Parallel()
	// sinceID is very old so all subtitles qualify, but there are only 2 pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") != "sorozat" {
			if r.URL.Query().Get("tipus") == "adatlap" {
				html := testutil.GenerateThirdPartyIDHTML("", 0, 0, 0)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(html))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		page := 1
		if p := r.URL.Query().Get("oldal"); p != "" {
			page, _ = strconv.Atoi(p)
		}

		totalPages := 2
		switch page {
		case 1:
			html := testutil.GenerateSubtitleTableHTMLWithPagination([]testutil.SubtitleRowOptions{
				{SubtitleID: 500, MagyarTitle: "Sub 1", EredetiTitle: "Show - 1x01", DownloadFilename: "s1.srt", ShowID: 10},
			}, page, totalPages, true)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		case 2:
			html := testutil.GenerateSubtitleTableHTMLWithPagination([]testutil.SubtitleRowOptions{
				{SubtitleID: 400, MagyarTitle: "Sub 2", EredetiTitle: "Show - 1x02", DownloadFilename: "s2.srt", ShowID: 10},
			}, page, totalPages, true)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
		default:
			t.Errorf("Unexpected page %d requested", page)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	// sinceID=1 means all subtitles (400, 500) are > sinceID, but only 2 pages exist
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamRecentSubtitles(ctx, 1))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(showSubtitles))
	}
	if len(showSubtitles[0].SubtitleCollection.Subtitles) != 2 {
		t.Errorf("Expected 2 subtitles, got %d", len(showSubtitles[0].SubtitleCollection.Subtitles))
	}
}

func TestClient_StreamRecentSubtitles_SinceZeroFetchesOnlyPage1(t *testing.T) {
	t.Parallel()
	// Even though pagination exists, sinceID=0 should only fetch page 1
	pagesFetched := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") != "sorozat" {
			if r.URL.Query().Get("tipus") == "adatlap" {
				html := testutil.GenerateThirdPartyIDHTML("", 0, 0, 0)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(html))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		pagesFetched++
		html := testutil.GenerateSubtitleTableHTMLWithPagination([]testutil.SubtitleRowOptions{
			{SubtitleID: 100, MagyarTitle: "Sub", EredetiTitle: "Show - 1x01", DownloadFilename: "s.srt", ShowID: 10},
		}, 1, 5, true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	_, err := testutil.CollectShowSubtitles(ctx, c.StreamRecentSubtitles(ctx, 0))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if pagesFetched != 1 {
		t.Errorf("Expected only 1 page fetched for sinceID=0, got %d", pagesFetched)
	}
}
