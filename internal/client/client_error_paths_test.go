package client

// Tests for error paths and edge cases in the client package.
// These complement the happy-path tests in other test files by covering
// HTTP error responses, invalid data handling, filtering, and resource cleanup.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
