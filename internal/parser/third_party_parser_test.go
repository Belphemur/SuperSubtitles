package parser

import (
	"strings"
	"testing"

	"github.com/Belphemur/SuperSubtitles/internal/models"
	"github.com/Belphemur/SuperSubtitles/internal/testutil"
)

func TestThirdPartyIdParser_ParseHtml(t *testing.T) {
	// Generate proper HTML content using the helper
	htmlContent := testutil.GenerateThirdPartyIDHTML("tt14261112", 366532, 60743, 366532)

	parser := NewThirdPartyIdParser()
	result, err := parser.ParseHtml(strings.NewReader(htmlContent))

	// Test that parsing succeeds
	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	// Test expected values
	expected := models.ThirdPartyIds{
		IMDBID:   "tt14261112",
		TVDBID:   366532,
		TVMazeID: 60743,
		TraktID:  366532,
	}

	if result.IMDBID != expected.IMDBID {
		t.Errorf("Expected IMDB ID %q, got %q", expected.IMDBID, result.IMDBID)
	}
	if result.TVDBID != expected.TVDBID {
		t.Errorf("Expected TVDB ID %d, got %d", expected.TVDBID, result.TVDBID)
	}
	if result.TVMazeID != expected.TVMazeID {
		t.Errorf("Expected TVMaze ID %d, got %d", expected.TVMazeID, result.TVMazeID)
	}
	if result.TraktID != expected.TraktID {
		t.Errorf("Expected Trakt ID %d, got %d", expected.TraktID, result.TraktID)
	}
}

func TestThirdPartyIdParser_ParseHtml_EmptyHTML(t *testing.T) {
	htmlContent := testutil.GenerateEmptyHTML()

	parser := NewThirdPartyIdParser()
	result, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed on empty HTML: %v", err)
	}

	// Should return empty struct when no links are found
	if result.IMDBID != "" || result.TVDBID != 0 || result.TVMazeID != 0 || result.TraktID != 0 {
		t.Errorf("Expected empty result for HTML without third-party links, got %+v", result)
	}
}

func TestThirdPartyIdParser_ParseHtml_PartialLinks(t *testing.T) {
	// Generate HTML with only IMDB and TVDB IDs
	htmlContent := testutil.GenerateThirdPartyIDHTML("tt12345678", 123456, 0, 0)

	parser := NewThirdPartyIdParser()
	result, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	expected := models.ThirdPartyIds{
		IMDBID: "tt12345678",
		TVDBID: 123456,
	}

	if result.IMDBID != expected.IMDBID {
		t.Errorf("Expected IMDB ID %q, got %q", expected.IMDBID, result.IMDBID)
	}
	if result.TVDBID != expected.TVDBID {
		t.Errorf("Expected TVDB ID %d, got %d", expected.TVDBID, result.TVDBID)
	}
	if result.TVMazeID != 0 {
		t.Errorf("Expected TVMaze ID 0, got %d", result.TVMazeID)
	}
	if result.TraktID != 0 {
		t.Errorf("Expected Trakt ID 0, got %d", result.TraktID)
	}
}

func TestThirdPartyIdParser_ParseHtml_InvalidHTML(t *testing.T) {
	htmlContent := testutil.GenerateInvalidThirdPartyHTML()

	parser := NewThirdPartyIdParser()
	result, err := parser.ParseHtml(strings.NewReader(htmlContent))

	if err != nil {
		t.Fatalf("ParseHtml failed on invalid HTML: %v", err)
	}

	// Should return empty struct for HTML without proper link structure
	if result.IMDBID != "" || result.TVDBID != 0 || result.TVMazeID != 0 || result.TraktID != 0 {
		t.Errorf("Expected empty result for HTML without adatlapRow links, got %+v", result)
	}
}

func TestThirdPartyIdParser_extractIMDBIDFromURL(t *testing.T) {
	parser := &ThirdPartyIdParser{}

	tests := []struct {
		href     string
		expected string
		hasError bool
	}{
		{"http://www.imdb.com/title/tt14261112/", "tt14261112", false},
		{"http://www.imdb.com/title/tt12345678", "tt12345678", false},
		{"https://imdb.com/title/tt98765432/", "tt98765432", false},
		{"http://www.imdb.com/title/", "", true},
		{"http://www.imdb.com/title/invalid/", "", true},
		{"http://www.themoviedb.org/movie/12345", "", true},
		{"", "", true},
	}

	for _, test := range tests {
		result, err := parser.extractIMDBIDFromURL(test.href)
		if test.hasError {
			if err == nil {
				t.Errorf("extractIMDBIDFromURL(%q) expected error, got %q", test.href, result)
			}
		} else {
			if err != nil {
				t.Errorf("extractIMDBIDFromURL(%q) unexpected error: %v", test.href, err)
			}
			if result != test.expected {
				t.Errorf("extractIMDBIDFromURL(%q) = %q, expected %q", test.href, result, test.expected)
			}
		}
	}
}

func TestThirdPartyIdParser_extractTVDBIDFromURL(t *testing.T) {
	parser := &ThirdPartyIdParser{}

	tests := []struct {
		href     string
		expected int
		hasError bool
	}{
		{"http://thetvdb.com/?tab=series&id=366532", 366532, false},
		{"https://thetvdb.com/?tab=series&id=123456", 123456, false},
		{"http://thetvdb.com/?id=789012", 789012, false},
		{"http://thetvdb.com/?tab=series&id=", 0, true},
		{"http://thetvdb.com/?tab=series&id=abc", 0, true},
		{"http://thetvdb.com/?tab=series&id=0", 0, true},
		{"http://thetvdb.com/", 0, true},
		{"", 0, true},
	}

	for _, test := range tests {
		result, err := parser.extractTVDBIDFromURL(test.href)
		if test.hasError {
			if err == nil {
				t.Errorf("extractTVDBIDFromURL(%q) expected error, got %d", test.href, result)
			}
		} else {
			if err != nil {
				t.Errorf("extractTVDBIDFromURL(%q) unexpected error: %v", test.href, err)
			}
			if result != test.expected {
				t.Errorf("extractTVDBIDFromURL(%q) = %d, expected %d", test.href, result, test.expected)
			}
		}
	}
}

func TestThirdPartyIdParser_extractTVMazeIDFromURL(t *testing.T) {
	parser := &ThirdPartyIdParser{}

	tests := []struct {
		href     string
		expected int
		hasError bool
	}{
		{"http://www.tvmaze.com/shows/60743", 60743, false},
		{"https://tvmaze.com/shows/12345", 12345, false},
		{"http://www.tvmaze.com/shows/999", 999, false},
		{"http://www.tvmaze.com/shows/", 0, true},
		{"http://www.tvmaze.com/shows/abc", 0, true},
		{"http://www.tvmaze.com/shows/0", 0, true},
		{"http://www.tvmaze.com/", 0, true},
		{"", 0, true},
	}

	for _, test := range tests {
		result, err := parser.extractTVMazeIDFromURL(test.href)
		if test.hasError {
			if err == nil {
				t.Errorf("extractTVMazeIDFromURL(%q) expected error, got %d", test.href, result)
			}
		} else {
			if err != nil {
				t.Errorf("extractTVMazeIDFromURL(%q) unexpected error: %v", test.href, err)
			}
			if result != test.expected {
				t.Errorf("extractTVMazeIDFromURL(%q) = %d, expected %d", test.href, result, test.expected)
			}
		}
	}
}

func TestThirdPartyIdParser_extractTraktIDFromURL(t *testing.T) {
	parser := &ThirdPartyIdParser{}

	tests := []struct {
		href     string
		expected int
		hasError bool
	}{
		{"http://trakt.tv/search/tvdb?utf8=%E2%9C%93&query=366532", 366532, false},
		{"https://trakt.tv/search/tvdb?query=123456", 123456, false},
		{"http://trakt.tv/search/tvdb?utf8=%E2%9C%93&query=999", 999, false},
		{"http://trakt.tv/search/tvdb?query=", 0, true},
		{"http://trakt.tv/search/tvdb?query=abc", 0, true},
		{"http://trakt.tv/search/tvdb?query=0", 0, true},
		{"http://trakt.tv/search/tvdb", 0, true},
		{"", 0, true},
	}

	for _, test := range tests {
		result, err := parser.extractTraktIDFromURL(test.href)
		if test.hasError {
			if err == nil {
				t.Errorf("extractTraktIDFromURL(%q) expected error, got %d", test.href, result)
			}
		} else {
			if err != nil {
				t.Errorf("extractTraktIDFromURL(%q) unexpected error: %v", test.href, err)
			}
			if result != test.expected {
				t.Errorf("extractTraktIDFromURL(%q) = %d, expected %d", test.href, result, test.expected)
			}
		}
	}
}
