package client

import (
	"SuperSubtitles/internal/config"
	"context"
	"os"
	"strings"
	"testing"
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
		if subtitle.Filename == "" {
			t.Errorf("Subtitle %d: Filename is empty", i)
		}
		if subtitle.DownloadURL == "" {
			t.Errorf("Subtitle %d: DownloadURL is empty", i)
		}
		if subtitle.Uploader == "" {
			t.Errorf("Subtitle %d: Uploader is empty", i)
		}

		// Log first few subtitles for debugging
		if i < 3 {
			t.Logf("Subtitle %d: ID=%s, Language=%s, Quality=%s, Season=%d, Episode=%d, IsSeasonPack=%t",
				i, subtitle.ID, subtitle.Language, subtitle.Quality.String(),
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
