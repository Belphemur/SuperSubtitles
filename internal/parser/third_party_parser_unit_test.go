// third_party_parser_unit_test.go tests individual third-party ID URL extraction
// functions in isolation. The companion third_party_parser_test.go covers integration-style
// tests using testutil HTML fixtures; this file focuses on edge cases for each extraction
// method and the ParseHtml method with URLs that match service names but fail extraction.
package parser

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// extractTVDBIDFromURL – non-numeric and negative id parameter
// ---------------------------------------------------------------------------

func TestThirdPartyIdParser_extractTVDBIDFromURL_NonNumeric(t *testing.T) {
	t.Parallel()
	p := &ThirdPartyIdParser{}

	tests := []struct {
		name string
		href string
	}{
		{"non-numeric id", "http://thetvdb.com/?tab=series&id=xyz"},
		{"negative id", "http://thetvdb.com/?tab=series&id=-5"},
		{"float id", "http://thetvdb.com/?tab=series&id=3.14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := p.extractTVDBIDFromURL(tt.href)
			if err == nil {
				t.Errorf("extractTVDBIDFromURL(%q) = %d, expected error", tt.href, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extractTVMazeIDFromURL – non-numeric ID in path
// ---------------------------------------------------------------------------

func TestThirdPartyIdParser_extractTVMazeIDFromURL_NonNumeric(t *testing.T) {
	t.Parallel()
	p := &ThirdPartyIdParser{}

	tests := []struct {
		name string
		href string
	}{
		{"non-numeric path segment", "http://www.tvmaze.com/shows/xyz-show"},
		{"path with trailing text", "http://www.tvmaze.com/shows/abc/episodes"},
		{"empty path segment", "http://www.tvmaze.com/shows/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := p.extractTVMazeIDFromURL(tt.href)
			if err == nil {
				t.Errorf("extractTVMazeIDFromURL(%q) = %d, expected error", tt.href, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extractTraktIDFromURL – non-numeric query parameter
// ---------------------------------------------------------------------------

func TestThirdPartyIdParser_extractTraktIDFromURL_NonNumeric(t *testing.T) {
	t.Parallel()
	p := &ThirdPartyIdParser{}

	tests := []struct {
		name string
		href string
	}{
		{"non-numeric query", "http://trakt.tv/search/tvdb?query=xyz"},
		{"negative query", "http://trakt.tv/search/tvdb?query=-10"},
		{"float query", "http://trakt.tv/search/tvdb?query=1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := p.extractTraktIDFromURL(tt.href)
			if err == nil {
				t.Errorf("extractTraktIDFromURL(%q) = %d, expected error", tt.href, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseHtml – invalid URLs that match service names but fail extraction
// ---------------------------------------------------------------------------

func TestThirdPartyIdParser_ParseHtml_InvalidServiceURLs(t *testing.T) {
	t.Parallel()
	p := &ThirdPartyIdParser{}

	// HTML with links that contain recognized service domains but have
	// invalid ID parameters that will fail individual extraction functions.
	html := `<html><body>
		<div class="adatlapRow">
			<a href="http://www.imdb.com/title/invalid/">IMDB</a>
			<a href="http://thetvdb.com/?tab=series&id=notanumber">TVDB</a>
			<a href="http://www.tvmaze.com/shows/notanumber">TVMaze</a>
			<a href="http://trakt.tv/search/tvdb?query=notanumber">Trakt</a>
		</div>
	</body></html>`

	result, err := p.ParseHtml(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseHtml() unexpected error: %v", err)
	}

	// All extractions should fail, leaving zero values
	if result.IMDBID != "" {
		t.Errorf("IMDBID = %q, want empty", result.IMDBID)
	}
	if result.TVDBID != 0 {
		t.Errorf("TVDBID = %d, want 0", result.TVDBID)
	}
	if result.TVMazeID != 0 {
		t.Errorf("TVMazeID = %d, want 0", result.TVMazeID)
	}
	if result.TraktID != 0 {
		t.Errorf("TraktID = %d, want 0", result.TraktID)
	}
}

func TestThirdPartyIdParser_ParseHtml_MixedValidAndInvalidURLs(t *testing.T) {
	t.Parallel()
	p := &ThirdPartyIdParser{}

	// One valid IMDB link, rest have invalid IDs
	html := `<html><body>
		<div class="adatlapRow">
			<a href="http://www.imdb.com/title/tt99887766/">IMDB</a>
			<a href="http://thetvdb.com/?tab=series&id=abc">TVDB</a>
			<a href="http://www.tvmaze.com/shows/">TVMaze</a>
			<a href="http://trakt.tv/search/tvdb">Trakt</a>
		</div>
	</body></html>`

	result, err := p.ParseHtml(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseHtml() unexpected error: %v", err)
	}

	if result.IMDBID != "tt99887766" {
		t.Errorf("IMDBID = %q, want %q", result.IMDBID, "tt99887766")
	}
	if result.TVDBID != 0 {
		t.Errorf("TVDBID = %d, want 0", result.TVDBID)
	}
	if result.TVMazeID != 0 {
		t.Errorf("TVMazeID = %d, want 0", result.TVMazeID)
	}
	if result.TraktID != 0 {
		t.Errorf("TraktID = %d, want 0", result.TraktID)
	}
}
