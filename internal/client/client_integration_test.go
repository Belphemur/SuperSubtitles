package client

import (
	"context"
	"fmt"
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
		if subtitle.ID == 0 {
			t.Errorf("Subtitle %d: ID is 0", i)
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
			t.Logf("Subtitle %d: ID=%d, Language=%s, Qualities=%v, Season=%d, Episode=%d, IsSeasonPack=%t",
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
		expectedURLPrefix := "https://feliratok.eu/index.php?action=letolt"
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
		if subtitle.ID == 0 {
			t.Errorf("Subtitle %d: ID is 0", i)
		}
		if subtitle.Language == "" {
			t.Errorf("Subtitle %d: Language is empty", i)
		}
		if subtitle.DownloadURL == "" {
			t.Errorf("Subtitle %d: DownloadURL is empty", i)
		}

		// Log first few subtitles for debugging
		if i < 3 {
			t.Logf("Subtitle %d: ID=%d, Language=%s, Qualities=%v, Season=%d, Episode=%d, SeasonPack=%t",
				i, subtitle.ID, subtitle.Language, subtitle.Qualities,
				subtitle.Season, subtitle.Episode, subtitle.IsSeasonPack)
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

// TestClient_GetRecentSubtitles_Integration is an integration test that calls the real SuperSubtitles website
// to fetch recently uploaded subtitles
func TestClient_GetRecentSubtitles_Integration(t *testing.T) {
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

	// Call GetRecentSubtitles without filter (get all recent subtitles)
	ctx := context.Background()
	showSubtitles, err := client.GetRecentSubtitles(ctx, 0)

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Integration test failed: GetRecentSubtitles returned error: %v", err)
	}

	// Basic smoke test: ensure we got some results
	if len(showSubtitles) == 0 {
		t.Error("Integration test failed: expected to get at least some recent subtitles, but got 0")
		return
	}

	// Log summary information
	t.Logf("\n========================================")
	t.Logf("Successfully fetched recent subtitles")
	t.Logf("Total shows with recent subtitles: %d", len(showSubtitles))
	t.Logf("========================================\n")

	// Count total subtitles across all shows
	totalSubtitles := 0
	for _, ss := range showSubtitles {
		totalSubtitles += ss.SubtitleCollection.Total
	}
	t.Logf("Total recent subtitles: %d\n", totalSubtitles)

	// Display detailed information about each show and its subtitles
	for i, ss := range showSubtitles {
		t.Logf("─────────────────────────────────────────")
		t.Logf("Show #%d: %s (ID: %d)", i+1, ss.Name, ss.ID)

		// Display third-party IDs if available
		if ss.ThirdPartyIds.IMDBID != "" || ss.ThirdPartyIds.TVDBID != 0 ||
			ss.ThirdPartyIds.TVMazeID != 0 || ss.ThirdPartyIds.TraktID != 0 {
			t.Logf("Third-party IDs:")
			if ss.ThirdPartyIds.IMDBID != "" {
				t.Logf("  • IMDB: %s", ss.ThirdPartyIds.IMDBID)
			}
			if ss.ThirdPartyIds.TVDBID != 0 {
				t.Logf("  • TVDB: %d", ss.ThirdPartyIds.TVDBID)
			}
			if ss.ThirdPartyIds.TVMazeID != 0 {
				t.Logf("  • TVMaze: %d", ss.ThirdPartyIds.TVMazeID)
			}
			if ss.ThirdPartyIds.TraktID != 0 {
				t.Logf("  • Trakt: %d", ss.ThirdPartyIds.TraktID)
			}
		}

		// Display subtitle information
		t.Logf("Recent subtitles: %d", ss.SubtitleCollection.Total)

		maxSubtitlesToShow := 5
		if ss.SubtitleCollection.Total < maxSubtitlesToShow {
			maxSubtitlesToShow = ss.SubtitleCollection.Total
		}

		for j := 0; j < maxSubtitlesToShow; j++ {
			sub := ss.SubtitleCollection.Subtitles[j]

			episodeInfo := ""
			if sub.IsSeasonPack {
				episodeInfo = " (Season Pack)"
			} else if sub.Season > 0 && sub.Episode > 0 {
				episodeInfo = fmt.Sprintf(" (S%02dE%02d)", sub.Season, sub.Episode)
			}

			qualitiesStr := ""
			if len(sub.Qualities) > 0 {
				qualityStrs := make([]string, len(sub.Qualities))
				for k, q := range sub.Qualities {
					qualityStrs[k] = q.String()
				}
				qualitiesStr = " [" + strings.Join(qualityStrs, ", ") + "]"
			}

			t.Logf("  %d. [%s] %s%s%s", j+1, sub.Language, sub.Name, episodeInfo, qualitiesStr)
			t.Logf("     Uploader: %s | ID: %d", sub.Uploader, sub.ID)
		}

		if ss.SubtitleCollection.Total > maxSubtitlesToShow {
			t.Logf("  ... and %d more subtitle(s)", ss.SubtitleCollection.Total-maxSubtitlesToShow)
		}

		// Verify data integrity
		if ss.ID == 0 {
			t.Errorf("Show %d: ID is 0", i)
		}
		if ss.Name == "" {
			t.Errorf("Show %d: Name is empty", i)
		}

		for j, sub := range ss.SubtitleCollection.Subtitles {
			if sub.ID == 0 {
				t.Errorf("Show %d, Subtitle %d: ID is 0", i, j)
			}
			if sub.Language == "" {
				t.Errorf("Show %d, Subtitle %d: Language is empty", i, j)
			}
			if sub.DownloadURL == "" {
				t.Errorf("Show %d, Subtitle %d: DownloadURL is empty", i, j)
			}
		}

		// Only show first 3 shows in detail to avoid too much output
		if i >= 2 {
			if len(showSubtitles) > 3 {
				t.Logf("\n... and %d more show(s) with recent subtitles", len(showSubtitles)-3)
			}
			break
		}
	}

	t.Logf("\n========================================")
	t.Logf("Integration test completed successfully!")
	t.Logf("========================================")
}

// TestClient_GetRecentSubtitles_WithFilter_Integration is an integration test that calls
// the real SuperSubtitles website to fetch recent subtitles with ID filtering
func TestClient_GetRecentSubtitles_WithFilter_Integration(t *testing.T) {
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
		ClientTimeout:       "30s",
	}

	// Create the client
	client := NewClient(testConfig)
	ctx := context.Background()

	// First, get all recent subtitles to find a valid ID to use as filter
	t.Log("Fetching all recent subtitles to determine filter ID...")
	allShowSubtitles, err := client.GetRecentSubtitles(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to fetch recent subtitles: %v", err)
	}

	if len(allShowSubtitles) == 0 {
		t.Skip("No recent subtitles available to test filtering")
	}

	// Find a subtitle ID to use as filter from the first show
	var filterID int
	if len(allShowSubtitles) > 0 && len(allShowSubtitles[0].SubtitleCollection.Subtitles) > 1 {
		middleIdx := len(allShowSubtitles[0].SubtitleCollection.Subtitles) / 2
		filterID = allShowSubtitles[0].SubtitleCollection.Subtitles[middleIdx].ID
	} else if len(allShowSubtitles) > 0 && len(allShowSubtitles[0].SubtitleCollection.Subtitles) > 0 {
		filterID = allShowSubtitles[0].SubtitleCollection.Subtitles[0].ID
	} else {
		t.Skip("No suitable subtitle ID found for filter test")
	}

	t.Logf("\n========================================")
	t.Logf("Testing with filter ID: %d", filterID)
	t.Logf("========================================\n")

	// Now fetch with filter
	filteredShowSubtitles, err := client.GetRecentSubtitles(ctx, filterID)
	if err != nil {
		t.Fatalf("Integration test failed: GetRecentSubtitles with filter returned error: %v", err)
	}

	// Log results
	t.Logf("Shows returned with filter: %d", len(filteredShowSubtitles))

	totalFiltered := 0
	for _, ss := range filteredShowSubtitles {
		totalFiltered += ss.SubtitleCollection.Total
	}
	t.Logf("Total subtitles with filter: %d", totalFiltered)

	// Verify all filtered subtitles have IDs greater than the filter ID
	for i, ss := range filteredShowSubtitles {
		for j, sub := range ss.SubtitleCollection.Subtitles {
			if sub.ID <= filterID {
				t.Errorf("Show %d, Subtitle %d: ID %d is not greater than filter ID %d",
					i, j, sub.ID, filterID)
			}
		}
	}

	t.Logf("\n========================================")
	t.Logf("Filter test completed successfully!")
	t.Logf("Verified all returned IDs > %d", filterID)
	t.Logf("========================================")
}
