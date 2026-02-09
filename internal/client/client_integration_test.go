package client

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// TestClient_GetShowList_Integration is an integration test that calls the real SuperSubtitles website
// This test is skipped in CI environments to avoid external dependencies
func TestClient_GetShowList_Integration(t *testing.T) {
	// Skip this test if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Skip if explicitly requested to skip integration tests
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test due to SKIP_INTEGRATION_TESTS environment variable")
	}

	// Create a config that points to the real SuperSubtitles website
	testConfig := &config.Config{
		SuperSubtitleDomain: "https://feliratok.eu",
		ClientTimeout:       "30s", // Use a reasonable timeout for real network calls
	}

	// Create the client
	client := NewClient(testConfig)

	// Call GetShowList with the real website
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Integration test failed: GetShowList returned error: %v", err)
	}

	// Basic smoke test: ensure we got some shows
	if len(shows) == 0 {
		t.Error("Integration test failed: expected to get at least some shows from the real website, but got 0")
	}

	// Log some basic information about what we received
	t.Logf("Successfully fetched %d shows from SuperSubtitles website", len(shows))

	// Verify that shows have basic required fields
	for i, show := range shows {
		if show.ID == 0 {
			t.Errorf("Show %d: ID is 0", i)
		}
		if show.Name == "" {
			t.Errorf("Show %d: Name is empty", i)
		}
		if show.ImageURL == "" {
			t.Errorf("Show %d: ImageURL is empty", i)
		}
		// Year can be 0 if parsing fails, so we don't check it strictly

		// Log first few shows for debugging
		if i < 3 {
			t.Logf("Show %d: ID=%d, Name=%s, Year=%d", i, show.ID, show.Name, show.Year)
		}
	}

	// Ensure we have a reasonable number of shows (the website typically has many shows)
	if len(shows) < 10 {
		t.Logf("Warning: Only got %d shows, which seems low for a real website response", len(shows))
	}
}

// TestClient_GetSubtitles_Integration is an integration test that calls the real SuperSubtitles website
// to fetch subtitles for a specific show ID
func TestClient_GetSubtitles_Integration(t *testing.T) {
	// Skip this test if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Skip if explicitly requested to skip integration tests
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test due to SKIP_INTEGRATION_TESTS environment variable")
	}

	// Create a config that points to the real SuperSubtitles website
	testConfig := &config.Config{
		SuperSubtitleDomain: "https://feliratok.eu",
		ClientTimeout:       "30s", // Use a reasonable timeout for real network calls
	}

	// Create the client
	client := NewClient(testConfig)

	// Call GetSubtitles with a known show ID
	ctx := context.Background()
	showID := 3217
	subtitles, err := client.GetSubtitles(ctx, showID)

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Integration test failed: GetSubtitles returned error: %v", err)
	}

	// Basic smoke test: ensure we got a subtitle collection
	if subtitles == nil {
		t.Error("Integration test failed: expected to get subtitle collection, but got nil")
		return
	}

	// Log some basic information about what we received
	t.Logf("Successfully fetched subtitle collection for show ID %d", showID)
	t.Logf("Show name: %s", subtitles.ShowName)
	t.Logf("Total subtitles: %d", subtitles.Total)

	// Verify that we have subtitles
	if subtitles.Total == 0 {
		t.Error("Integration test failed: expected to get at least some subtitles, but got 0")
	}

	if len(subtitles.Subtitles) != subtitles.Total {
		t.Errorf("Integration test failed: subtitle count mismatch - Total: %d, actual length: %d",
			subtitles.Total, len(subtitles.Subtitles))
	}

	// Verify that subtitles have basic required fields
	for i, subtitle := range subtitles.Subtitles {
		if subtitle.ID == "" {
			t.Errorf("Subtitle %d: ID is empty", i)
		}
		if subtitle.Language == "" {
			t.Errorf("Subtitle %d: Language is empty", i)
		}
		if subtitle.DownloadURL == "" {
			t.Errorf("Subtitle %d: DownloadURL is empty", i)
		}
		if subtitle.Uploader == "" {
			t.Errorf("Subtitle %d: Uploader is empty", i)
		}

		// Log first few subtitles for debugging
		if i < 3 {
			t.Logf("Subtitle %d: ID=%s, Language=%s, Qualities=%v, Season=%d, Episode=%d, IsSeasonPack=%t",
				i, subtitle.ID, subtitle.Language, subtitle.Qualities,
				subtitle.Season, subtitle.Episode, subtitle.IsSeasonPack)
		}
	}

	// Verify that we have some languages available
	if subtitles.ShowName == "" {
		t.Error("Integration test failed: show name is empty")
	}

	// Test that download URLs are properly constructed
	for i, subtitle := range subtitles.Subtitles {
		expectedURLPrefix := "https://feliratok.eu/index.php?action=letolt&felirat="
		if !strings.HasPrefix(subtitle.DownloadURL, expectedURLPrefix) {
			t.Errorf("Subtitle %d: DownloadURL does not have expected prefix. Got: %s", i, subtitle.DownloadURL)
		}

		// Only check first few to avoid spam
		if i >= 3 {
			break
		}
	}

	// Ensure we have a reasonable number of subtitles (the show should have multiple language options)
	if subtitles.Total < 2 {
		t.Logf("Warning: Only got %d subtitles, which seems low for a real show response", subtitles.Total)
	}
}

// TestClient_GetShowSubtitles_Integration is an integration test that calls the real SuperSubtitles website
// to fetch third-party IDs and subtitles for shows
func TestClient_GetShowSubtitles_Integration(t *testing.T) {
	// Skip this test if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Skip if explicitly requested to skip integration tests
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test due to SKIP_INTEGRATION_TESTS environment variable")
	}

	// Create a config that points to the real SuperSubtitles website
	testConfig := &config.Config{
		SuperSubtitleDomain: "https://feliratok.eu",
		ClientTimeout:       "30s", // Use a reasonable timeout for real network calls
	}

	// Create the client
	client := NewClient(testConfig)

	// First get a list of shows to pick from
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)
	if err != nil {
		t.Fatalf("Integration test failed: GetShowList returned error: %v", err)
	}

	if len(shows) == 0 {
		t.Fatal("Integration test failed: no shows available to test GetShowSubtitles")
	}

	// Pick the first show for testing (should have some basic data)
	testShow := shows[0]
	t.Logf("Testing GetShowSubtitles with show: ID=%d, Name=%s", testShow.ID, testShow.Name)

	// Call GetShowSubtitles with the test show
	showSubtitlesList, err := client.GetShowSubtitles(ctx, []models.Show{testShow})

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Integration test failed: GetShowSubtitles returned error: %v", err)
	}

	// Basic smoke test: ensure we got results
	if len(showSubtitlesList) == 0 {
		t.Error("Integration test failed: expected to get at least one ShowSubtitles result, but got 0")
		return
	}

	if len(showSubtitlesList) > 1 {
		t.Logf("Warning: Got %d results, expected 1. This might indicate duplicate processing.", len(showSubtitlesList))
	}

	// Get the first result
	result := showSubtitlesList[0]

	// Log some basic information about what we received
	t.Logf("Successfully fetched ShowSubtitles for show ID %d", testShow.ID)
	t.Logf("Show name: %s", result.Name)
	t.Logf("Third-party IDs - IMDB: %s, TVDB: %d, TVMaze: %d, Trakt: %d",
		result.ThirdPartyIds.IMDBID, result.ThirdPartyIds.TVDBID,
		result.ThirdPartyIds.TVMazeID, result.ThirdPartyIds.TraktID)
	t.Logf("Total subtitles: %d", result.SubtitleCollection.Total)

	// Verify basic show data
	if result.ID != testShow.ID {
		t.Errorf("Expected show ID %d, got %d", testShow.ID, result.ID)
	}
	if result.Name != testShow.Name {
		t.Errorf("Expected show name %s, got %s", testShow.Name, result.Name)
	}

	// Verify subtitle collection exists and has data
	if result.SubtitleCollection.Total == 0 {
		t.Error("Integration test failed: expected to get at least some subtitles")
	}

	if len(result.SubtitleCollection.Subtitles) != result.SubtitleCollection.Total {
		t.Errorf("Integration test failed: subtitle count mismatch - Total: %d, actual length: %d",
			result.SubtitleCollection.Total, len(result.SubtitleCollection.Subtitles))
	}

	// Verify that subtitles have basic required fields (check first few)
	for i, subtitle := range result.SubtitleCollection.Subtitles {
		if subtitle.ID == "" {
			t.Errorf("Subtitle %d: ID is empty", i)
		}
		if subtitle.Language == "" {
			t.Errorf("Subtitle %d: Language is empty", i)
		}
		if subtitle.DownloadURL == "" {
			t.Errorf("Subtitle %d: DownloadURL is empty", i)
		}

		// Log first few subtitles for debugging
		if i < 3 {
			t.Logf("Subtitle %d: ID=%s, Language=%s, Qualities=%v, Season=%d, Episode=%d",
				i, subtitle.ID, subtitle.Language, subtitle.Qualities,
				subtitle.Season, subtitle.Episode)
		}

		// Only check first few to avoid spam
		if i >= 5 {
			break
		}
	}

	// Third-party IDs are optional (may not be available for all shows), so we don't fail if they're empty
	// But log them for visibility
	if result.ThirdPartyIds.IMDBID != "" {
		t.Logf("Successfully extracted IMDB ID: %s", result.ThirdPartyIds.IMDBID)
	}
	if result.ThirdPartyIds.TVDBID != 0 {
		t.Logf("Successfully extracted TVDB ID: %d", result.ThirdPartyIds.TVDBID)
	}
	if result.ThirdPartyIds.TVMazeID != 0 {
		t.Logf("Successfully extracted TVMaze ID: %d", result.ThirdPartyIds.TVMazeID)
	}
	if result.ThirdPartyIds.TraktID != 0 {
		t.Logf("Successfully extracted Trakt ID: %d", result.ThirdPartyIds.TraktID)
	}
}
