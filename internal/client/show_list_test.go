package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

func TestClient_GetShowList(t *testing.T) {
	// HTML for waiting (varakozik) endpoint
	waitingHTML := `
		<html><body><table><tbody>
		<tr><td colspan="10">2025</td></tr>
		<tr><td><a href="index.php?sid=12190"><img src="sorozat_cat.php?kep=12190"/></a></td><td class="sangol"><div>7 Bears</div></td></tr>
		<tr><td><a href="index.php?sid=12347"><img src="sorozat_cat.php?kep=12347"/></a></td><td class="sangol"><div>#1 Happy Family USA</div></td></tr>
		<tr><td><a href="index.php?sid=12549"><img src="sorozat_cat.php?kep=12549"/></a></td><td class="sangol"><div>A Thousand Blows</div></td></tr>
		</tbody></table></body></html>`

	// HTML for under translation (alatt) endpoint
	underHTML := `
		<html><body><table><tbody>
		<tr><td colspan="10">2024</td></tr>
		<tr><td><a href="index.php?sid=12076"><img src="sorozat_cat.php?kep=12076"/></a></td><td class="sangol"><div>Adults</div></td></tr>
		<tr><td><a href="index.php?sid=12007"><img src="sorozat_cat.php?kep=12007"/></a></td><td class="sangol"><div>Asura</div></td></tr>
		</tbody></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sorf") == "varakozik-subrip" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(waitingHTML))
		} else if r.URL.Query().Get("sorf") == "alatt-subrip" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(underHTML))
		} else if r.URL.Query().Get("sorf") == "nem-all-forditas-alatt" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body><table></table></body></html>"))
		}
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call GetShowList
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that we got the expected number of shows
	expectedCount := 5
	if len(shows) != expectedCount {
		t.Fatalf("Expected %d shows, got %d", expectedCount, len(shows))
	}

	// Test specific shows - map by ID since order is not deterministic
	expectedShows := map[int]models.Show{
		12190: {Name: "7 Bears", ID: 12190, Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12190"},
		12347: {Name: "#1 Happy Family USA", ID: 12347, Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12347"},
		12549: {Name: "A Thousand Blows", ID: 12549, Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12549"},
		12076: {Name: "Adults", ID: 12076, Year: 2024, ImageURL: server.URL + "/sorozat_cat.php?kep=12076"},
		12007: {Name: "Asura", ID: 12007, Year: 2024, ImageURL: server.URL + "/sorozat_cat.php?kep=12007"},
	}

	// Verify each show by ID
	seenIDs := make(map[int]bool)
	for _, show := range shows {
		expected, exists := expectedShows[show.ID]
		if !exists {
			t.Errorf("Unexpected show ID %d", show.ID)
			continue
		}
		seenIDs[show.ID] = true

		if show.Name != expected.Name {
			t.Errorf("Show %d: expected name %s, got %s", show.ID, expected.Name, show.Name)
		}
		if show.Year != expected.Year {
			t.Errorf("Show %d: expected year %d, got %d", show.ID, expected.Year, show.Year)
		}
		if show.ImageURL != expected.ImageURL {
			t.Errorf("Show %d: expected image URL %s, got %s", show.ID, expected.ImageURL, show.ImageURL)
		}
	}

	// Verify we got all expected shows
	if len(seenIDs) != len(expectedShows) {
		missing := make([]int, 0)
		for id := range expectedShows {
			if !seenIDs[id] {
				missing = append(missing, id)
			}
		}
		t.Errorf("Missing shows with IDs: %v", missing)
	}
}

func TestClient_GetShowList_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call GetShowList
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	// Test that the call fails with an error
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if shows != nil {
		t.Fatalf("Expected nil shows, got %v", shows)
	}
}

func TestClient_GetShowList_PartialFailure(t *testing.T) {
	// One endpoint succeeds, the other fails (500)
	waitingHTML := `<html><body><table><tbody><tr><td colspan="10">2025</td></tr><tr><td><a href="index.php?sid=999"><img src="sorozat_cat.php?kep=999"/></a></td><td class="sangol"><div>Only Show</div></td></tr></tbody></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sorf") == "varakozik-subrip" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(waitingHTML))
		} else {
			// Other endpoints fail
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	testConfig := &config.Config{SuperSubtitleDomain: server.URL, ClientTimeout: "5s"}
	client := NewClient(testConfig)
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	if err != nil { // Should not fail completely when one endpoint succeeds
		t.Fatalf("Expected no error with partial success, got: %v", err)
	}
	if len(shows) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(shows))
	}
	if shows[0].Name != "Only Show" || shows[0].ID != 999 {
		t.Errorf("Expected show 'Only Show' with ID 999, got %s with ID %d", shows[0].Name, shows[0].ID)
	}
}

func TestClient_GetShowList_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // Delay longer than timeout
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer server.Close()

	// Create a test config with short timeout
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "500ms",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call GetShowList
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	// Test that the call fails with timeout error
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if shows != nil {
		t.Fatalf("Expected nil shows on timeout, got %v", shows)
	}
}

func TestClient_GetShowList_InvalidHTML(t *testing.T) {
	// Create a test server that returns invalid HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not html"))
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call GetShowList
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	// Test that the call succeeds but returns empty results
	if err != nil {
		t.Fatalf("Expected no error with invalid HTML, got: %v", err)
	}

	if len(shows) != 0 {
		t.Fatalf("Expected 0 shows with invalid HTML, got %d", len(shows))
	}
}

func TestClient_GetShowList_WithProxy(t *testing.T) {
	// Sample HTML content
	htmlContent := `
		<html>
		<body>
			<table>
				<tr>
					<td colspan="10">2025</td>
				</tr>
				<tr>
					<td><a href="index.php?sid=12345"><img src="sorozat_cat.php?kep=12345"/></a></td>
					<td class="sangol"><div>Test Show</div></td>
				</tr>
			</table>
		</body>
		</html>
	`

	// Create a test server that returns the sample HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	// Create a test config with proxy (using the same server as proxy for simplicity)
	testConfig := &config.Config{
		SuperSubtitleDomain:   server.URL,
		ClientTimeout:         "10s",
		ProxyConnectionString: server.URL, // This won't actually proxy but tests the configuration
	}

	// Create the client
	client := NewClient(testConfig)

	// Call GetShowList
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	// Test that the call succeeds (proxy configuration should not break the request)
	if err != nil {
		t.Fatalf("Expected no error with proxy, got: %v", err)
	}

	// Should still get the show
	if len(shows) != 1 {
		t.Fatalf("Expected 1 show, got %d", len(shows))
	}
}
