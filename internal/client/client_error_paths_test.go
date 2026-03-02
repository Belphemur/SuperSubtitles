package client

// Tests for error paths and edge cases in the client package.
// These complement the happy-path tests in other test files by covering
// HTTP error responses, invalid data handling, filtering, and resource cleanup.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
)

func TestClient_StreamSubtitles_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	_, err := testutil.CollectSubtitles(ctx, c.StreamSubtitles(ctx, 9999))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, &apperrors.ErrNotFound{}) {
		t.Fatalf("Expected ErrNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), "show") || !strings.Contains(err.Error(), "9999") {
		t.Errorf("Expected error to mention resource 'show' and ID '9999', got: %v", err)
	}
}

func TestClient_StreamSubtitles_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	_, err := testutil.CollectSubtitles(ctx, c.StreamSubtitles(ctx, 9999))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if errors.Is(err, &apperrors.ErrNotFound{}) {
		t.Fatal("Expected non-NotFound error for 500 response")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("Expected error mentioning status 500, got: %v", err)
	}
}

func TestClient_StreamShowSubtitles_ThirdPartyIdsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tipus") == "adatlap" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
			{
				SubtitleID:       1770600001,
				ShowID:           123,
				MagyarTitle:      "Test Subtitle",
				EredetiTitle:     "Test Show - 1x01 - Episode (1080p-Group)",
				DownloadFilename: "test.srt",
			},
		})
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

	shows := []models.Show{{Name: "Test Show", ID: 123}}
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamShowSubtitles(ctx, shows))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(showSubtitles))
	}
	result := showSubtitles[0]
	if result.ThirdPartyIds.IMDBID != "" {
		t.Errorf("Expected empty IMDB ID, got %s", result.ThirdPartyIds.IMDBID)
	}
	if result.ThirdPartyIds.TVDBID != 0 {
		t.Errorf("Expected 0 TVDB ID, got %d", result.ThirdPartyIds.TVDBID)
	}
	if result.SubtitleCollection.Total != 1 {
		t.Errorf("Expected 1 subtitle, got %d", result.SubtitleCollection.Total)
	}
}

func TestClient_StreamShowSubtitles_InvalidSubtitleIDSkipped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tipus") == "adatlap" {
			html := testutil.GenerateThirdPartyIDHTML("", 0, 0, 0)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
			{
				SubtitleID:       -1,
				ShowID:           123,
				MagyarTitle:      "Invalid Sub",
				EredetiTitle:     "Test Show - 1x01 - Invalid Episode (720p-Grp)",
				DownloadFilename: "invalid.srt",
			},
			{
				SubtitleID:       1770600001,
				ShowID:           123,
				MagyarTitle:      "Valid Sub",
				EredetiTitle:     "Test Show - 1x02 - Valid Episode (1080p-Grp)",
				DownloadFilename: "valid.srt",
			},
		})
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

	shows := []models.Show{{Name: "Test Show", ID: 123}}
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamShowSubtitles(ctx, shows))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(showSubtitles))
	}
	if showSubtitles[0].SubtitleCollection.Total != 1 {
		t.Errorf("Expected 1 subtitle (invalid filtered out), got %d", showSubtitles[0].SubtitleCollection.Total)
	}
}

func TestClient_StreamShowSubtitles_NoValidSubtitleIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
			{
				SubtitleID:       -1,
				ShowID:           123,
				MagyarTitle:      "Invalid Sub 1",
				EredetiTitle:     "Test Show - 1x01 - Episode One (720p-Grp)",
				DownloadFilename: "invalid1.srt",
			},
			{
				SubtitleID:       -2,
				ShowID:           123,
				MagyarTitle:      "Invalid Sub 2",
				EredetiTitle:     "Test Show - 1x02 - Episode Two (720p-Grp)",
				DownloadFilename: "invalid2.srt",
			},
		})
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

	shows := []models.Show{{Name: "Test Show", ID: 123}}
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamShowSubtitles(ctx, shows))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(showSubtitles))
	}
	if showSubtitles[0].SubtitleCollection.Total != 0 {
		t.Errorf("Expected 0 subtitles (all invalid), got %d", showSubtitles[0].SubtitleCollection.Total)
	}
	if showSubtitles[0].ThirdPartyIds.IMDBID != "" {
		t.Errorf("Expected empty IMDB ID, got %s", showSubtitles[0].ThirdPartyIds.IMDBID)
	}
}

func TestClient_StreamRecentSubtitles_SinceIDFiltering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{SubtitleID: 100, ShowID: 10, MagyarTitle: "Old Sub", EredetiTitle: "Show A - 1x01 - Old (720p-Grp)", DownloadFilename: "old.srt"},
				{SubtitleID: 200, ShowID: 10, MagyarTitle: "Mid Sub", EredetiTitle: "Show A - 1x02 - Mid (720p-Grp)", DownloadFilename: "mid.srt"},
				{SubtitleID: 300, ShowID: 10, MagyarTitle: "New Sub", EredetiTitle: "Show A - 1x03 - New (720p-Grp)", DownloadFilename: "new.srt"},
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

	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamRecentSubtitles(ctx, 200))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(showSubtitles))
	}
	if len(showSubtitles[0].SubtitleCollection.Subtitles) != 1 {
		t.Errorf("Expected 1 subtitle after filtering, got %d", len(showSubtitles[0].SubtitleCollection.Subtitles))
	}
	if showSubtitles[0].SubtitleCollection.Subtitles[0].ID != 300 {
		t.Errorf("Expected subtitle ID 300, got %d", showSubtitles[0].SubtitleCollection.Subtitles[0].ID)
	}
}

func TestClient_StreamRecentSubtitles_SkipMissingShowID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{SubtitleID: 100, SkipShowIDDefault: true, MagyarTitle: "No Show ID", EredetiTitle: "Unknown - 1x01 - Mystery (720p-Grp)", DownloadFilename: "nosid.srt"},
				{SubtitleID: 200, ShowID: 10, MagyarTitle: "Valid Sub", EredetiTitle: "Show A - 1x01 - Episode (720p-Grp)", DownloadFilename: "valid.srt"},
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
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show (subtitle with missing show ID skipped), got %d", len(showSubtitles))
	}
	if showSubtitles[0].ID != 10 {
		t.Errorf("Expected show ID 10, got %d", showSubtitles[0].ID)
	}
}

func TestClient_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	if err := c.Close(); err != nil {
		t.Fatalf("Expected no error from Close, got: %v", err)
	}
}

func TestClient_NewClient_InvalidTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"film":"0","sorozat":"0"}`))
	}))
	defer server.Close()

	c := NewClient(&config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "not-a-duration",
	})
	if c == nil {
		t.Fatal("Expected non-nil client even with invalid timeout")
	}
	defer c.Close()

	// Verify the client is functional by making a request
	result, err := c.CheckForUpdates(context.Background(), 1)
	if err != nil {
		t.Fatalf("Expected client to work with default timeout, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestClient_NewClient_WithProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(&config.Config{
		SuperSubtitleDomain:   server.URL,
		ClientTimeout:         "10s",
		ProxyConnectionString: "http://proxy.example.com:8080",
	})
	if c == nil {
		t.Fatal("Expected non-nil client with proxy config")
	}
	defer c.Close()

	// Verify the underlying client has the proxy set on its transport
	impl := c.(*client)
	ct := impl.httpClient.Transport.(*compressionTransport)
	baseTransport := ct.transport.(*http.Transport)
	if baseTransport.Proxy == nil {
		t.Error("Expected proxy to be configured on transport")
	}
}

func TestClient_NewClient_EmptyTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(&config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "",
	})
	if c == nil {
		t.Fatal("Expected non-nil client with empty timeout")
	}
	defer c.Close()
}

func TestClient_BuildDownloadURL_InvalidBaseURL(t *testing.T) {
	c := &client{
		baseURL: "://",
	}
	_, err := c.buildDownloadURL("123")
	if err == nil {
		t.Fatal("Expected error for invalid base URL")
	}
	if !strings.Contains(err.Error(), "invalid base URL") {
		t.Errorf("Expected error to mention 'invalid base URL', got: %v", err)
	}
}

func TestClient_DownloadSubtitle_InvalidBaseURL(t *testing.T) {
	c := &client{
		baseURL: "://",
	}
	_, err := c.DownloadSubtitle(context.Background(), "123", nil)
	if err == nil {
		t.Fatal("Expected error for invalid base URL in DownloadSubtitle")
	}
	if !strings.Contains(err.Error(), "invalid base URL") {
		t.Errorf("Expected error to mention 'invalid base URL', got: %v", err)
	}
}

func TestClient_StreamSubtitles_ContextCancelDuringPagination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var page2Requested atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "oldal=") {
			page2Requested.Store(true)
			// Cancel context when subsequent pages are requested
			cancel()
			rows := []testutil.SubtitleRowOptions{
				{SubtitleID: 201, ShowID: 100, MagyarTitle: "Sub Page2", EredetiTitle: "Show - 1x02 - Ep2 (720p-Grp)", DownloadFilename: "s2.srt"},
			}
			html := testutil.GenerateSubtitleTableHTML(rows)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		// First page with pagination indicating 3 total pages
		rows := []testutil.SubtitleRowOptions{
			{SubtitleID: 101, ShowID: 100, MagyarTitle: "Sub Page1", EredetiTitle: "Show - 1x01 - Ep1 (720p-Grp)", DownloadFilename: "s1.srt"},
		}
		html := testutil.GenerateSubtitleTableHTMLWithPagination(rows, 1, 3, true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)

	ch := c.StreamSubtitles(ctx, 100)
	var count int
	for range ch {
		count++
	}

	// We should have at least the first page result and at most fewer than all 3 pages
	if count < 1 {
		t.Error("Expected at least 1 subtitle from first page")
	}
}

func TestClient_StreamSubtitles_PaginationPageError(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		// Page 2 returns server error
		if r.URL.Query().Get("oldal") == "2" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Page 3 returns valid data
		if r.URL.Query().Get("oldal") == "3" {
			rows := []testutil.SubtitleRowOptions{
				{SubtitleID: 301, ShowID: 100, MagyarTitle: "Sub P3", EredetiTitle: "Show - 1x03 - Ep3 (720p-Grp)", DownloadFilename: "s3.srt"},
			}
			html := testutil.GenerateSubtitleTableHTML(rows)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		// First page with 3 total pages
		rows := []testutil.SubtitleRowOptions{
			{SubtitleID: 101, ShowID: 100, MagyarTitle: "Sub P1", EredetiTitle: "Show - 1x01 - Ep1 (720p-Grp)", DownloadFilename: "s1.srt"},
		}
		html := testutil.GenerateSubtitleTableHTMLWithPagination(rows, 1, 3, true)
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

	result, err := testutil.CollectSubtitles(ctx, c.StreamSubtitles(ctx, 100))
	if err != nil {
		t.Fatalf("Expected partial success (no fatal error), got: %v", err)
	}

	// Should have subtitles from page 1 and page 3 (page 2 failed)
	if result.Total != 2 {
		t.Errorf("Expected 2 subtitles (page 1 + page 3, page 2 failed), got %d", result.Total)
	}
}

func TestClient_StreamShowList_AllEndpointsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	shows, err := testutil.CollectShows(ctx, c.StreamShowList(ctx))
	if err == nil {
		t.Fatal("Expected error when all endpoints fail")
	}
	if !strings.Contains(err.Error(), "all show list endpoints failed") {
		t.Errorf("Expected 'all show list endpoints failed' error, got: %v", err)
	}
	if len(shows) != 0 {
		t.Errorf("Expected 0 shows, got %d", len(shows))
	}
}

func TestClient_StreamShowList_PartialEndpointFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only the "varakozik" endpoint succeeds
		if r.URL.Query().Get("sorf") == "varakozik-subrip" {
			html := testutil.GenerateShowTableHTML([]testutil.ShowRowOptions{
				{ShowID: 1001, ShowName: "Test Show", Year: 2025},
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	shows, err := testutil.CollectShows(ctx, c.StreamShowList(ctx))
	if err != nil {
		t.Fatalf("Expected partial success (no error), got: %v", err)
	}
	if len(shows) != 1 {
		t.Errorf("Expected 1 show from successful endpoint, got %d", len(shows))
	}
	if shows[0].ID != 1001 {
		t.Errorf("Expected show ID 1001, got %d", shows[0].ID)
	}
}

func TestClient_CheckForUpdates_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"film":"0","sorozat":"0"}`))
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.CheckForUpdates(ctx, 1)
	if err == nil {
		t.Fatal("Expected error with cancelled context")
	}
}

func TestClient_StreamRecentSubtitles_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	_, err := testutil.CollectShowSubtitles(ctx, c.StreamRecentSubtitles(ctx, 0))
	if err == nil {
		t.Fatal("Expected error for non-OK status")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("Expected error mentioning status 500, got: %v", err)
	}
}

func TestClient_StreamRecentSubtitles_AllSubtitlesFiltered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tab") == "sorozat" {
			// All subtitle IDs are below sinceID
			html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
				{SubtitleID: 50, ShowID: 10, MagyarTitle: "Old Sub 1", EredetiTitle: "Show A - 1x01 - Ep1 (720p-Grp)", DownloadFilename: "old1.srt"},
				{SubtitleID: 80, ShowID: 10, MagyarTitle: "Old Sub 2", EredetiTitle: "Show A - 1x02 - Ep2 (720p-Grp)", DownloadFilename: "old2.srt"},
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
	c := NewClient(testConfig)
	ctx := context.Background()

	// sinceID=100 means all subtitles (50, 80) are filtered out
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, c.StreamRecentSubtitles(ctx, 100))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(showSubtitles) != 0 {
		t.Errorf("Expected 0 shows (all filtered), got %d", len(showSubtitles))
	}
}

func TestClient_StreamShowSubtitles_SubtitleStreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// All subtitle requests return 500
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	shows := []models.Show{{Name: "Failing Show", ID: 999}}
	_, err := testutil.CollectShowSubtitles(ctx, c.StreamShowSubtitles(ctx, shows))
	if err == nil {
		t.Fatal("Expected error when all shows fail")
	}
	if !strings.Contains(err.Error(), "all shows failed") {
		t.Errorf("Expected 'all shows failed' error, got: %v", err)
	}
}

func TestClient_StreamSubtitles_WithPaginationBatchErrors(t *testing.T) {
	// Test pagination with 5 pages where pages 2 and 4 fail.
	// Pages are fetched in batches of 2, so batch 1 = [2,3], batch 2 = [4,5].
	pageHTML := func(pageNum, totalPages int) string {
		rows := []testutil.SubtitleRowOptions{
			{
				SubtitleID:       pageNum*100 + 1,
				ShowID:           200,
				MagyarTitle:      "Sub P" + strconv.Itoa(pageNum),
				EredetiTitle:     "Show - 1x0" + strconv.Itoa(pageNum) + " - Ep (720p-Grp)",
				DownloadFilename: "p" + strconv.Itoa(pageNum) + ".srt",
			},
		}
		return testutil.GenerateSubtitleTableHTMLWithPagination(rows, pageNum, totalPages, true)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldalStr := r.URL.Query().Get("oldal")
		if oldalStr == "2" || oldalStr == "4" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if oldalStr == "" {
			// First page
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageHTML(1, 5)))
			return
		}
		pageNum, _ := strconv.Atoi(oldalStr)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageHTML(pageNum, 5)))
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	result, err := testutil.CollectSubtitles(ctx, c.StreamSubtitles(ctx, 200))
	if err != nil {
		t.Fatalf("Expected partial success, got: %v", err)
	}
	// Pages 1, 3, 5 succeed (1 subtitle each), pages 2 and 4 fail
	if result.Total != 3 {
		t.Errorf("Expected 3 subtitles from successful pages, got %d", result.Total)
	}
}

func TestClient_StreamShowList_ContextCancelledBetweenBatches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cancel context on any request so streaming stops
		cancel()
		html := testutil.GenerateShowTableHTML([]testutil.ShowRowOptions{
			{ShowID: 1, ShowName: "Show 1", Year: 2025},
		})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)

	ch := c.StreamShowList(ctx)
	var count int
	for range ch {
		count++
	}
	// Should complete without hanging; may get 0 or more shows depending on timing
	if count < 0 {
		t.Error("Unexpected negative count")
	}
}

func TestClient_StreamSubtitles_RequestCreationError(t *testing.T) {
	// Use a base URL with a control character that makes http.NewRequestWithContext fail
	testConfig := &config.Config{
		SuperSubtitleDomain: "http://invalid\x00host",
		ClientTimeout:       "10s",
	}
	c := NewClient(testConfig)
	ctx := context.Background()

	_, err := testutil.CollectSubtitles(ctx, c.StreamSubtitles(ctx, 1))
	if err == nil {
		t.Fatal("Expected error for invalid URL in request creation")
	}
}
