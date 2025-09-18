package client

import (
	"SuperSubtitles/internal/config"
	"SuperSubtitles/internal/models"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_GetShowList(t *testing.T) {
	// Sample HTML content based on the SuperSubtitles website structure
	htmlContent := `
		<html>
		<body>
			<table>
				<tbody>
					<tr>
						<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
							2025
						</td>
					</tr>
					<tr style="background-color: #ecf6fc">
						<td style="padding: 5px;">
							<a href="index.php?sid=12190"><img class="kategk" src="sorozat_cat.php?kep=12190"/></a>
						</td>
						<td class="sangol">
							<div>
								7 Bears
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr style="background-color: #fff">
						<td style="padding: 5px;">
							<a href="index.php?sid=12347"><img class="kategk" src="sorozat_cat.php?kep=12347"/></a>
						</td>
						<td class="sangol">
							<div>
								#1 Happy Family USA
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr style="background-color: #ecf6fc">
						<td style="padding: 5px;">
							<a href="index.php?sid=12549"><img class="kategk" src="sorozat_cat.php?kep=12549"/></a>
						</td>
						<td class="sangol">
							<div>
								A Thousand Blows
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr style="background-color: #fff">
						<td style="padding: 5px;">
							<a href="index.php?sid=12076"><img class="kategk" src="sorozat_cat.php?kep=12076"/></a>
						</td>
						<td class="sangol">
							<div>
								Adults
							</div>
							<div class="sev"></div>
						</td>
					</tr>
					<tr>
						<td colspan="10" style="text-align: center; background-color: #DDDDDD; font-size: 12pt; color:#0000CC; border-top: 2px solid #9B9B9B;">
							2024
						</td>
					</tr>
					<tr style="background-color: #ecf6fc">
						<td style="padding: 5px;">
							<a href="index.php?sid=12007"><img class="kategk" src="sorozat_cat.php?kep=12007"/></a>
						</td>
						<td class="sangol">
							<div>
								Asura
							</div>
							<div class="sev"></div>
						</td>
					</tr>
				</tbody>
			</table>
		</body>
		</html>
	`

	// Create a test server that returns the sample HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is for the correct endpoint
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=varakozik-subrip" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(htmlContent))
		} else {
			w.WriteHeader(http.StatusNotFound)
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
		t.Fatalf("GetShowList failed: %v", err)
	}

	// Test that we got the expected number of shows
	expectedCount := 5
	if len(shows) != expectedCount {
		t.Errorf("Expected %d shows, got %d", expectedCount, len(shows))
	}

	// Test specific shows
	expectedShows := []models.Show{
		{Name: "7 Bears", ID: "12190", Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12190"},
		{Name: "#1 Happy Family USA", ID: "12347", Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12347"},
		{Name: "A Thousand Blows", ID: "12549", Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12549"},
		{Name: "Adults", ID: "12076", Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12076"},
		{Name: "Asura", ID: "12007", Year: 2024, ImageURL: server.URL + "/sorozat_cat.php?kep=12007"},
	}

	for i, expected := range expectedShows {
		if i >= len(shows) {
			t.Errorf("Missing show at index %d", i)
			continue
		}

		actual := shows[i]
		if actual.Name != expected.Name {
			t.Errorf("Show %d: expected name %q, got %q", i, expected.Name, actual.Name)
		}
		if actual.ID != expected.ID {
			t.Errorf("Show %d: expected ID %q, got %q", i, expected.ID, actual.ID)
		}
		if actual.Year != expected.Year {
			t.Errorf("Show %d: expected year %d, got %d", i, expected.Year, actual.Year)
		}
		if actual.ImageURL != expected.ImageURL {
			t.Errorf("Show %d: expected imageURL %q, got %q", i, expected.ImageURL, actual.ImageURL)
		}
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
		t.Fatal("Expected GetShowList to fail with server error, but it succeeded")
	}

	if shows != nil {
		t.Errorf("Expected shows to be nil on error, got %v", shows)
	}
}

func TestClient_GetShowList_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html></html>"))
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
		t.Fatal("Expected GetShowList to fail with timeout, but it succeeded")
	}

	if shows != nil {
		t.Errorf("Expected shows to be nil on timeout, got %v", shows)
	}
}

func TestClient_GetShowList_InvalidHTML(t *testing.T) {
	// Create a test server that returns invalid HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Invalid HTML</body></html>"))
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
		t.Fatalf("GetShowList failed: %v", err)
	}

	if len(shows) != 0 {
		t.Errorf("Expected 0 shows from invalid HTML, got %d", len(shows))
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
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
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
		t.Fatalf("GetShowList failed with proxy config: %v", err)
	}

	// Should still get the show
	if len(shows) != 1 {
		t.Errorf("Expected 1 show with proxy config, got %d", len(shows))
	}
}
