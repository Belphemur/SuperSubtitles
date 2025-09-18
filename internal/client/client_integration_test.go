package client

import (
	"SuperSubtitles/internal/config"
	"context"
	"os"
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
