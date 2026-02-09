package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

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
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=varakozik-subrip" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(waitingHTML))
			return
		}
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=alatt-subrip" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(underHTML))
			return
		}
		w.WriteHeader(http.StatusNotFound)
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

	// Test specific shows (order: from first endpoint then second)
	expectedShows := []models.Show{
		{Name: "7 Bears", ID: 12190, Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12190"},
		{Name: "#1 Happy Family USA", ID: 12347, Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12347"},
		{Name: "A Thousand Blows", ID: 12549, Year: 2025, ImageURL: server.URL + "/sorozat_cat.php?kep=12549"},
		{Name: "Adults", ID: 12076, Year: 2024, ImageURL: server.URL + "/sorozat_cat.php?kep=12076"},
		{Name: "Asura", ID: 12007, Year: 2024, ImageURL: server.URL + "/sorozat_cat.php?kep=12007"},
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
			t.Errorf("Show %d: expected ID %d, got %d", i, expected.ID, actual.ID)
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

func TestClient_GetShowList_PartialFailure(t *testing.T) {
	// One endpoint succeeds, the other fails (500)
	waitingHTML := `<html><body><table><tbody><tr><td colspan="10">2025</td></tr><tr><td><a href="index.php?sid=999"><img src="sorozat_cat.php?kep=999"/></a></td><td class="sangol"><div>Only Show</div></td></tr></tbody></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=varakozik-subrip" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(waitingHTML))
			return
		}
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=alatt-subrip" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	testConfig := &config.Config{SuperSubtitleDomain: server.URL, ClientTimeout: "5s"}
	client := NewClient(testConfig)
	ctx := context.Background()
	shows, err := client.GetShowList(ctx)

	if err != nil { // Should not fail completely when one endpoint succeeds
		t.Fatalf("Expected partial success without error, got: %v", err)
	}
	if len(shows) != 1 {
		t.Fatalf("Expected 1 show from successful endpoint, got %d", len(shows))
	}
	if shows[0].Name != "Only Show" || shows[0].ID != 999 {
		t.Errorf("Unexpected show data: %+v", shows[0])
	}
}

func TestClient_GetShowList_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
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
		_, _ = w.Write([]byte("<html><body>Invalid HTML</body></html>"))
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
		t.Fatalf("GetShowList failed with proxy config: %v", err)
	}

	// Should still get the show
	if len(shows) != 1 {
		t.Errorf("Expected 1 show with proxy config, got %d", len(shows))
	}
}

func TestClient_GetShowSubtitles(t *testing.T) {
	// Sample HTML for detail page with third-party IDs
	detailPageHTML := `
		<html>
		<body>
			<div class="adatlapTabla">
				<div class="adatlapAdat">
					<div class="adatlapRow">
						<a href="http://www.imdb.com/title/tt12345678/" target="_blank" alt="iMDB"></a>
						<a href="http://thetvdb.com/?tab=series&id=987654" target="_blank" alt="TheTVDB"></a>
						<a href="http://www.tvmaze.com/shows/555666" target="_blank" alt="TVMaze"></a>
						<a href="http://trakt.tv/search/tvdb?utf8=%E2%9C%93&query=987654" target="_blank" alt="trakt"></a>
					</div>
				</div>
			</div>
		</body>
		</html>
	`

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sid=12345" {
			// Subtitles request - HTML format
			htmlResponse := `
			<html><body>
			<table><tbody>
			<tr><td>Kategoria</td><td>Language</td><td>Description</td><td>Uploader</td><td>Date</td><td>Download</td></tr>
			<tr>
				<td>cat</td>
				<td>Angol</td>
				<td><a href="/subtitle.php?feliratid=1435431909">Test Show - 1x1 (1080p-RelGroup)</a></td>
				<td>TestUser</td>
				<td>2025-02-08</td>
				<td><a href="/download?id=1435431909">Download</a></td>
			</tr>
			</tbody></table>
			</body></html>`
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(htmlResponse))
		} else if r.URL.Path == "/index.php" && r.URL.RawQuery == "tipus=adatlap&azon=a_1435431909" {
			// Detail page request
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(detailPageHTML))
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

	// Test shows
	shows := []models.Show{
		{Name: "Test Show", ID: 12345, Year: 2023, ImageURL: server.URL + "/image.jpg"},
	}

	// Call GetShowSubtitles
	ctx := context.Background()
	showSubtitles, err := client.GetShowSubtitles(ctx, shows)

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("GetShowSubtitles failed: %v", err)
	}

	// Test that we got the expected results
	if len(showSubtitles) != 1 {
		t.Fatalf("Expected 1 show subtitle, got %d", len(showSubtitles))
	}

	result := showSubtitles[0]

	// Test show data
	if result.Name != "Test Show" {
		t.Errorf("Expected show name 'Test Show', got '%s'", result.Name)
	}
	if result.ID != 12345 {
		t.Errorf("Expected show ID 12345, got %d", result.ID)
	}

	// Test third-party IDs
	if result.ThirdPartyIds.IMDBID != "tt12345678" {
		t.Errorf("Expected IMDB ID 'tt12345678', got '%s'", result.ThirdPartyIds.IMDBID)
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
		t.Errorf("Expected subtitle show name 'Test Show', got '%s'", result.SubtitleCollection.ShowName)
	}
}

func TestClient_CheckForUpdates(t *testing.T) {
	// Sample JSON response for update check
	jsonResponse := `{"film":"2","sorozat":"5"}`

	// Create a test server that returns the sample JSON for update check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "action=recheck&azon=1760700519" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(jsonResponse))
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

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("CheckForUpdates failed: %v", err)
	}

	// Test that we got the expected result
	if result == nil {
		t.Fatal("Expected update check result, got nil")
	}

	// Test the counts
	if result.FilmCount != 2 {
		t.Errorf("Expected film count 2, got %d", result.FilmCount)
	}
	if result.SeriesCount != 5 {
		t.Errorf("Expected series count 5, got %d", result.SeriesCount)
	}
	if !result.HasUpdates {
		t.Errorf("Expected HasUpdates to be true, got %t", result.HasUpdates)
	}
}

func TestClient_CheckForUpdates_WithPrefix(t *testing.T) {
	// Sample JSON response for update check
	jsonResponse := `{"film":"0","sorozat":"1"}`

	// Create a test server that returns the sample JSON for update check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "action=recheck&azon=1760700519" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(jsonResponse))
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

	// Call CheckForUpdates with "a_" prefix (should be trimmed)
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "a_1760700519")

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("CheckForUpdates failed: %v", err)
	}

	// Test that we got the expected result
	if result == nil {
		t.Fatal("Expected update check result, got nil")
	}

	// Test the counts
	if result.FilmCount != 0 {
		t.Errorf("Expected film count 0, got %d", result.FilmCount)
	}
	if result.SeriesCount != 1 {
		t.Errorf("Expected series count 1, got %d", result.SeriesCount)
	}
	if !result.HasUpdates {
		t.Errorf("Expected HasUpdates to be true, got %t", result.HasUpdates)
	}
}

func TestClient_CheckForUpdates_NoUpdates(t *testing.T) {
	// Sample JSON response for no updates
	jsonResponse := `{"film":"0","sorozat":"0"}`

	// Create a test server that returns the sample JSON for update check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "action=recheck&azon=1760700519" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(jsonResponse))
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

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call succeeds
	if err != nil {
		t.Fatalf("CheckForUpdates failed: %v", err)
	}

	// Test that we got the expected result
	if result == nil {
		t.Fatal("Expected update check result, got nil")
	}

	// Test the counts
	if result.FilmCount != 0 {
		t.Errorf("Expected film count 0, got %d", result.FilmCount)
	}
	if result.SeriesCount != 0 {
		t.Errorf("Expected series count 0, got %d", result.SeriesCount)
	}
	if result.HasUpdates {
		t.Errorf("Expected HasUpdates to be false, got %t", result.HasUpdates)
	}
}

func TestClient_CheckForUpdates_ServerError(t *testing.T) {
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

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call fails with an error
	if err == nil {
		t.Fatal("Expected CheckForUpdates to fail with server error, but it succeeded")
	}

	if result != nil {
		t.Errorf("Expected result to be nil on error, got %v", result)
	}
}

func TestClient_CheckForUpdates_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create a test config
	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	// Create the client
	client := NewClient(testConfig)

	// Call CheckForUpdates
	ctx := context.Background()
	result, err := client.CheckForUpdates(ctx, "1760700519")

	// Test that the call fails with JSON decode error
	if err == nil {
		t.Fatal("Expected CheckForUpdates to fail with JSON decode error, but it succeeded")
	}

	if result != nil {
		t.Errorf("Expected result to be nil on error, got %v", result)
	}
}

func TestClient_DownloadSubtitle(t *testing.T) {
	// Test download of a regular (non-ZIP) subtitle file
	subtitleContent := "1\n00:00:01,000 --> 00:00:02,000\nTest subtitle line\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-subrip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(subtitleContent))
	}))
	defer server.Close()

	testConfig := &config.Config{
		SuperSubtitleDomain: server.URL,
		ClientTimeout:       "10s",
	}

	client := NewClient(testConfig)
	ctx := context.Background()

	result, err := client.DownloadSubtitle(ctx, server.URL, models.DownloadRequest{
		SubtitleID: "1234567890",
		Episode:    0,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if string(result.Content) != subtitleContent {
		t.Errorf("Expected content '%s', got '%s'", subtitleContent, string(result.Content))
	}

	if result.ContentType != "application/x-subrip" {
		t.Errorf("Expected content type 'application/x-subrip', got '%s'", result.ContentType)
	}
}
func TestClient_GetSubtitles_WithPagination(t *testing.T) {
	// Create test HTML for 3 pages with pagination links
	pageHTML := func(pageNum int, totalPages int) string {
		subtitleRows := ""
		for i := 1; i <= 3; i++ {
			subtitleID := pageNum*100 + i
			subtitleRows += `
		<tr>
			<td>cat</td>
			<td>Magyar</td>
			<td><a href="/subtitle.php?feliratid=123">Stranger Things S01E0` + strconv.Itoa(i) + ` - 1080p-RelGroup</a></td>
			<td>Uploader` + strconv.Itoa(pageNum) + `</td>
			<td>2025-02-08</td>
			<td><a href="/download?id=` + strconv.Itoa(subtitleID) + `">Download</a></td>
		</tr>`
		}

		paginationHTML := ""
		for p := 1; p <= totalPages; p++ {
			paginationHTML += `<a href="index.php?sid=3217&oldal=` + strconv.Itoa(p) + `">` + strconv.Itoa(p) + `</a> `
		}

		return `
	<html><body>
	<table><tbody>
	<tr><td>Kategoria</td><td>Language</td><td>Description</td><td>Uploader</td><td>Date</td><td>Download</td></tr>
	` + subtitleRows + `
	</tbody></table>
	<div class="pagination">` + paginationHTML + `</div>
	</body></html>`
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

	result, err := client.GetSubtitles(ctx, 3217)

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
	singlePageHTML := `
	<html><body>
	<table><tbody>
	<tr><td>Kategoria</td><td>Language</td><td>Description</td><td>Uploader</td><td>Date</td><td>Download</td></tr>
	<tr>
		<td>cat</td>
		<td>Magyar</td>
		<td><a href="/subtitle.php?feliratid=1">Game of Thrones S01E01 - 1080p-Group</a></td>
		<td>UploaderA</td>
		<td>2025-02-08</td>
		<td><a href="/download?id=1">Download</a></td>
	</tr>
	</tbody></table>
	</body></html>`

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

	result, err := client.GetSubtitles(ctx, 1234)

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

	result, err := client.GetSubtitles(ctx, 5555)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if result != nil {
		t.Fatalf("Expected nil result for error case, got: %v", result)
	}
}
