package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/testutil"
)

func TestClient_GetSubtitles_WithPagination(t *testing.T) {
	// Create test HTML for 3 pages with pagination links
	pageHTML := func(pageNum int, totalPages int) string {
		var rows []testutil.SubtitleRowOptions
		for i := 1; i <= 3; i++ {
			subtitleID := pageNum*100 + i
			rows = append(rows, testutil.SubtitleRowOptions{
				ShowID:           3217,
				Language:         "Magyar",
				FlagImage:        "hungary.gif",
				MagyarTitle:      "Stranger Things S01E0" + strconv.Itoa(i),
				EredetiTitle:     "Stranger Things S01E0" + strconv.Itoa(i) + " - Episode Title (1080p-RelGroup)",
				Uploader:         "Uploader" + strconv.Itoa(pageNum),
				UploaderBold:     false,
				UploadDate:       "2025-02-08",
				DownloadAction:   "letolt",
				DownloadFilename: "stranger.things.s01e0" + strconv.Itoa(i) + ".srt",
				SubtitleID:       subtitleID,
			})
		}

		// Use the dedicated function that generates HTML with pagination
		return testutil.GenerateSubtitleTableHTMLWithPagination(rows, pageNum, totalPages, true)
	}

	requestCount := 0
	pagesCalled := make(map[int]bool)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.URL.Path == "/index.php" && (r.URL.RawQuery == "sid=3217" || r.URL.RawQuery == "sid=3217&oldal=1") {
			// First page
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageHTML(1, 3)))
			pagesCalled[1] = true
			requestCount++
			return
		}
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sid=3217&oldal=2" {
			// Page 2
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageHTML(2, 3)))
			pagesCalled[2] = true
			requestCount++
			return
		}
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sid=3217&oldal=3" {
			// Page 3
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageHTML(3, 3)))
			pagesCalled[3] = true
			requestCount++
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	client := NewClient(testConfig)
	ctx := context.Background()

	result, err := testutil.CollectSubtitles(ctx, client.StreamSubtitles(ctx, 3217))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Should have 9 total subtitles (3 per page × 3 pages)
	expectedTotalSubtitles := 9
	if result.Total != expectedTotalSubtitles {
		t.Errorf("Expected %d total subtitles, got %d", expectedTotalSubtitles, result.Total)
	}

	// Should have made 3 requests
	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}

	// Verify all pages were called
	if !pagesCalled[1] || !pagesCalled[2] || !pagesCalled[3] {
		t.Errorf("Not all pages were called: page1=%v, page2=%v, page3=%v", pagesCalled[1], pagesCalled[2], pagesCalled[3])
	}
}

func TestClient_GetSubtitles_SinglePage(t *testing.T) {
	// Test with single page (no pagination)
	singlePageHTML := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
		{
			ShowID:           1234,
			Language:         "Magyar",
			FlagImage:        "hungary.gif",
			MagyarTitle:      "Game of Thrones - 1x1",
			EredetiTitle:     "Game of Thrones S01E01 - 1080p-Group",
			Uploader:         "UploaderA",
			UploaderBold:     false,
			UploadDate:       "2025-02-08",
			DownloadAction:   "letolt",
			DownloadFilename: "got.s01e01.srt",
			SubtitleID:       1,
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sid=1234" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(singlePageHTML))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	client := NewClient(testConfig)
	ctx := context.Background()

	result, err := testutil.CollectSubtitles(ctx, client.StreamSubtitles(ctx, 1234))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Total != 1 {
		t.Errorf("Expected 1 subtitle, got %d", result.Total)
	}

	if len(result.Subtitles) != 1 {
		t.Errorf("Expected 1 subtitle, got %d", len(result.Subtitles))
	}
}

func TestClient_GetSubtitles_NetworkError(t *testing.T) {
	// Test error handling for network failure
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

	result, err := testutil.CollectSubtitles(ctx, client.StreamSubtitles(ctx, 5555))

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if result != nil {
		t.Fatalf("Expected nil result for error case, got: %v", result)
	}
}
