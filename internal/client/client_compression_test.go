package client

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/config"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// TestClient_GetShowList_WithGzipCompression tests that the client properly handles gzip-compressed responses
func TestClient_GetShowList_WithGzipCompression(t *testing.T) {
	// HTML for waiting (varakozik) endpoint
	waitingHTML := `
		<html><body><table><tbody>
		<tr><td colspan="10">2025</td></tr>
		<tr><td><a href="index.php?sid=12190"><img src="sorozat_cat.php?kep=12190"/></a></td><td class="sangol"><div>7 Bears</div></td></tr>
		</tbody></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept-Encoding header includes gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			t.Errorf("Expected Accept-Encoding to contain 'gzip', got %q", acceptEncoding)
		}

		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=varakozik-subrip" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)

			// Write gzip-compressed response
			gzWriter := gzip.NewWriter(w)
			_, _ = gzWriter.Write([]byte(waitingHTML))
			_ = gzWriter.Close()
			return
		}
		if r.URL.Path == "/index.php" && strings.Contains(r.URL.RawQuery, "sorf=") {
			// Return empty response for other endpoints to avoid noise
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body><table><tbody></tbody></table></body></html>"))
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
	shows, err := client.GetShowList(ctx)

	if err != nil {
		t.Fatalf("GetShowList failed: %v", err)
	}

	if len(shows) != 1 {
		t.Errorf("Expected 1 show, got %d", len(shows))
	}

	if len(shows) > 0 && shows[0].Name != "7 Bears" {
		t.Errorf("Expected show name '7 Bears', got %q", shows[0].Name)
	}
}

// TestClient_GetShowList_WithBrotliCompression tests that the client properly handles brotli-compressed responses
func TestClient_GetShowList_WithBrotliCompression(t *testing.T) {
	waitingHTML := `
		<html><body><table><tbody>
		<tr><td colspan="10">2025</td></tr>
		<tr><td><a href="index.php?sid=12347"><img src="sorozat_cat.php?kep=12347"/></a></td><td class="sangol"><div>#1 Happy Family USA</div></td></tr>
		</tbody></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept-Encoding header includes br (brotli)
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "br") {
			t.Errorf("Expected Accept-Encoding to contain 'br', got %q", acceptEncoding)
		}

		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=varakozik-subrip" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "br")
			w.WriteHeader(http.StatusOK)

			// Write brotli-compressed response
			brWriter := brotli.NewWriter(w)
			_, _ = brWriter.Write([]byte(waitingHTML))
			_ = brWriter.Close()
			return
		}
		if r.URL.Path == "/index.php" && strings.Contains(r.URL.RawQuery, "sorf=") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body><table><tbody></tbody></table></body></html>"))
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
	shows, err := client.GetShowList(ctx)

	if err != nil {
		t.Fatalf("GetShowList failed: %v", err)
	}

	if len(shows) != 1 {
		t.Errorf("Expected 1 show, got %d", len(shows))
	}

	if len(shows) > 0 && shows[0].Name != "#1 Happy Family USA" {
		t.Errorf("Expected show name '#1 Happy Family USA', got %q", shows[0].Name)
	}
}

// TestClient_GetShowList_WithZstdCompression tests that the client properly handles zstd-compressed responses
func TestClient_GetShowList_WithZstdCompression(t *testing.T) {
	waitingHTML := `
		<html><body><table><tbody>
		<tr><td colspan="10">2025</td></tr>
		<tr><td><a href="index.php?sid=12549"><img src="sorozat_cat.php?kep=12549"/></a></td><td class="sangol"><div>A Thousand Blows</div></td></tr>
		</tbody></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept-Encoding header includes zstd
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "zstd") {
			t.Errorf("Expected Accept-Encoding to contain 'zstd', got %q", acceptEncoding)
		}

		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sorf=varakozik-subrip" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "zstd")
			w.WriteHeader(http.StatusOK)

			// Write zstd-compressed response
			// zstd.NewWriter() with default options never fails
			zstdWriter, _ := zstd.NewWriter(w)
			_, _ = zstdWriter.Write([]byte(waitingHTML))
			_ = zstdWriter.Close()
			return
		}
		if r.URL.Path == "/index.php" && strings.Contains(r.URL.RawQuery, "sorf=") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body><table><tbody></tbody></table></body></html>"))
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
	shows, err := client.GetShowList(ctx)

	if err != nil {
		t.Fatalf("GetShowList failed: %v", err)
	}

	if len(shows) != 1 {
		t.Errorf("Expected 1 show, got %d", len(shows))
	}

	if len(shows) > 0 && shows[0].Name != "A Thousand Blows" {
		t.Errorf("Expected show name 'A Thousand Blows', got %q", shows[0].Name)
	}
}

// TestClient_GetSubtitles_WithGzipCompression tests that GetSubtitles works with gzip compression
func TestClient_GetSubtitles_WithGzipCompression(t *testing.T) {
	htmlResponse := `<html><body>
<table><tbody>
<tr><td>Kategoria</td><td>Nyelv</td><td>Felirat</td><td>Feltöltő</td><td>Dátum</td><td>Letöltés</td></tr>
<tr>
	<td>cat</td>
	<td>Magyar</td>
	<td>
		<a href="/subtitle.php?feliratid=12345">
		Billy the Kid - 3x07 - The Last Buffalo (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR)
		</a>
	</td>
	<td>gricsi</td>
	<td>2026-01-31</td>
	<td><a href="/download?file=12345">Letöltés</a></td>
</tr>
</tbody></table>
</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sid=12345" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)

			gzWriter := gzip.NewWriter(w)
			_, _ = gzWriter.Write([]byte(htmlResponse))
			_ = gzWriter.Close()
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
	subtitles, err := client.GetSubtitles(ctx, 12345)

	if err != nil {
		t.Fatalf("GetSubtitles failed: %v", err)
	}

	if subtitles.Total != 1 {
		t.Errorf("Expected 1 subtitle, got %d", subtitles.Total)
	}

	if subtitles.ShowName != "Billy the Kid" {
		t.Errorf("Expected show name 'Billy the Kid', got %q", subtitles.ShowName)
	}

	if len(subtitles.Subtitles) > 0 && subtitles.Subtitles[0].Season != 3 {
		t.Errorf("Expected season 3, got %d", subtitles.Subtitles[0].Season)
	}

	if len(subtitles.Subtitles) > 0 && subtitles.Subtitles[0].Episode != 7 {
		t.Errorf("Expected episode 7, got %d", subtitles.Subtitles[0].Episode)
	}
}

// TestClient_GetSubtitles_WithBrotliCompression tests that GetSubtitles works with brotli compression
func TestClient_GetSubtitles_WithBrotliCompression(t *testing.T) {
	htmlResponse := `<html><body>
<table><tbody>
<tr><td>Kategoria</td><td>Nyelv</td><td>Felirat</td><td>Feltöltő</td><td>Dátum</td><td>Letöltés</td></tr>
<tr>
	<td>cat</td>
	<td>Magyar</td>
	<td>
		<a href="/subtitle.php?feliratid=23456">
		Billy the Kid - 3x06 - The Chain Gang (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR)
		</a>
	</td>
	<td>gricsi</td>
	<td>2026-01-21</td>
	<td><a href="/download?file=23456">Letöltés</a></td>
</tr>
</tbody></table>
</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sid=12345" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "br")
			w.WriteHeader(http.StatusOK)

			brWriter := brotli.NewWriter(w)
			_, _ = brWriter.Write([]byte(htmlResponse))
			_ = brWriter.Close()
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
	subtitles, err := client.GetSubtitles(ctx, 12345)

	if err != nil {
		t.Fatalf("GetSubtitles failed: %v", err)
	}

	if subtitles.Total != 1 {
		t.Errorf("Expected 1 subtitle, got %d", subtitles.Total)
	}

	if subtitles.ShowName != "Billy the Kid" {
		t.Errorf("Expected show name 'Billy the Kid', got %q", subtitles.ShowName)
	}

	if len(subtitles.Subtitles) > 0 && subtitles.Subtitles[0].Season != 3 {
		t.Errorf("Expected season 3, got %d", subtitles.Subtitles[0].Season)
	}

	if len(subtitles.Subtitles) > 0 && subtitles.Subtitles[0].Episode != 6 {
		t.Errorf("Expected episode 6, got %d", subtitles.Subtitles[0].Episode)
	}
}

// TestClient_GetSubtitles_WithZstdCompression tests that GetSubtitles works with zstd compression
func TestClient_GetSubtitles_WithZstdCompression(t *testing.T) {
	htmlResponse := `<html><body>
<table><tbody>
<tr><td>Kategoria</td><td>Nyelv</td><td>Felirat</td><td>Feltöltő</td><td>Dátum</td><td>Letöltés</td></tr>
<tr>
	<td>cat</td>
	<td>Magyar</td>
	<td>
		<a href="/subtitle.php?feliratid=34567">
		Billy the Kid - 3x05 - Breaking the Shackles (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR)
		</a>
	</td>
	<td>gricsi</td>
	<td>2026-01-12</td>
	<td><a href="/download?file=34567">Letöltés</a></td>
</tr>
</tbody></table>
</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.php" && r.URL.RawQuery == "sid=12345" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "zstd")
			w.WriteHeader(http.StatusOK)

			// zstd.NewWriter() with default options never fails
			zstdWriter, _ := zstd.NewWriter(w)
			_, _ = zstdWriter.Write([]byte(htmlResponse))
			_ = zstdWriter.Close()
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
	subtitles, err := client.GetSubtitles(ctx, 12345)

	if err != nil {
		t.Fatalf("GetSubtitles failed: %v", err)
	}

	if subtitles.Total != 1 {
		t.Errorf("Expected 1 subtitle, got %d", subtitles.Total)
	}

	if subtitles.ShowName != "Billy the Kid" {
		t.Errorf("Expected show name 'Billy the Kid', got %q", subtitles.ShowName)
	}

	if len(subtitles.Subtitles) > 0 && subtitles.Subtitles[0].Season != 3 {
		t.Errorf("Expected season 3, got %d", subtitles.Subtitles[0].Season)
	}

	if len(subtitles.Subtitles) > 0 && subtitles.Subtitles[0].Episode != 5 {
		t.Errorf("Expected episode 5, got %d", subtitles.Subtitles[0].Episode)
	}
}
