package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
)

func TestClient_GetShowSubtitles(t *testing.T) {
	t.Parallel()
	// Sample HTML for detail page with third-party IDs
	detailPageHTML := testutil.GenerateThirdPartyIDHTML("tt12345678", 987654, 555666, 987654)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if it's a detail page request
		if r.URL.Query().Get("tipus") == "adatlap" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(detailPageHTML))
			return
		}

		// Otherwise, return subtitle listing
		showIDStr := r.URL.Query().Get("sid")
		showID, _ := strconv.Atoi(showIDStr)

		html := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
			{
				SubtitleID:       1770600001,
				MagyarTitle:      "Test Subtitle",
				EredetiTitle:     "Test Show - 1x01",
				DownloadFilename: "test.srt",
				ShowID:           showID,
			},
		})

		_ = showID // Use the variable
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Test shows
	shows := []models.Show{
		{Name: "Test Show", ID: 12345, Year: 2023, ImageURL: server.URL + "/image.jpg"},
	}

	// Call StreamShowSubtitles and collect results
	ctx := context.Background()
	showSubtitles, err := testutil.CollectShowSubtitles(ctx, client.StreamShowSubtitles(ctx, shows))

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that we got the expected results
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show result, got %d", len(showSubtitles))
	}

	result := showSubtitles[0]

	// Test show data
	if result.Name != "Test Show" {
		t.Errorf("Expected show name 'Test Show', got %s", result.Name)
	}
	if result.ID != 12345 {
		t.Errorf("Expected show ID 12345, got %d", result.ID)
	}

	// Test third-party IDs
	if result.ThirdPartyIds.IMDBID != "tt12345678" {
		t.Errorf("Expected IMDB ID 'tt12345678', got %s", result.ThirdPartyIds.IMDBID)
	}
	if result.ThirdPartyIds.TVDBID != 987654 {
		t.Errorf("Expected TVDB ID 987654, got %d", result.ThirdPartyIds.TVDBID)
	}
	if result.ThirdPartyIds.TVMazeID != 555666 {
		t.Errorf("Expected TVMaze ID 555666, got %d", result.ThirdPartyIds.TVMazeID)
	}
	if result.ThirdPartyIds.TraktID != 987654 {
		t.Errorf("Expected Trakt ID 987654, got %d", result.ThirdPartyIds.TraktID)
	}

	// Test subtitle collection
	if result.SubtitleCollection.Total != 1 {
		t.Errorf("Expected 1 subtitle, got %d", result.SubtitleCollection.Total)
	}
	if result.SubtitleCollection.ShowName != "Test Show" {
		t.Errorf("Expected subtitle show name 'Test Show', got %s", result.SubtitleCollection.ShowName)
	}
}
