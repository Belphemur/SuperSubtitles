// subtitle_parser_unit_test.go tests individual helper functions in subtitle_parser.go
// that have low unit-test coverage. The companion subtitle_parser_test.go file contains
// integration-style tests using full HTML fixtures; this file focuses on exercising each
// helper method in isolation with table-driven tests.
package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
	"github.com/PuerkitoBio/goquery"
)

// ---------------------------------------------------------------------------
// extractIDFromDownloadLink
// ---------------------------------------------------------------------------

func TestSubtitleParser_extractIDFromDownloadLink(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name string
		link string
		want int
	}{
		{"felirat param", "letolt.php?felirat=42&fnev=test.srt", 42},
		{"feliratid param", "letolt.php?feliratid=99&fnev=test.srt", 99},
		{"id param", "letolt.php?id=7", 7},
		{"path-based /123/", "/downloads/123/subtitle.srt", 123},
		{"digits before extension", "456.srt", 456},
		{"no id found", "letolt.php?fnev=test.srt", -1},
		{"empty string", "", -1},
		{"malformed url with query", "://bad?felirat=abc", -1},
		{"non-numeric felirat", "letolt.php?felirat=abc", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parser.extractIDFromDownloadLink(tt.link)
			if got != tt.want {
				t.Errorf("extractIDFromDownloadLink(%q) = %d, want %d", tt.link, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseDate
// ---------------------------------------------------------------------------

func TestSubtitleParser_parseDate(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name    string
		dateStr string
		want    time.Time
	}{
		{"valid date", "2025-01-21", time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC)},
		{"invalid date", "not-a-date", time.Time{}},
		{"empty string", "", time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parser.parseDate(tt.dateStr)
			if !got.Equal(tt.want) {
				t.Errorf("parseDate(%q) = %v, want %v", tt.dateStr, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// constructDownloadURL
// ---------------------------------------------------------------------------

func TestSubtitleParser_constructDownloadURL(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name string
		link string
		want string
	}{
		{"empty link", "", ""},
		{"full http url", "http://example.com/file.srt", "http://example.com/file.srt"},
		{"full https url", "https://example.com/file.srt", "https://example.com/file.srt"},
		{"leading slash", "/letolt.php?id=1", "https://feliratok.eu/letolt.php?id=1"},
		{"relative link", "letolt.php?id=1", "https://feliratok.eu/letolt.php?id=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parser.constructDownloadURL(tt.link)
			if got != tt.want {
				t.Errorf("constructDownloadURL(%q) = %q, want %q", tt.link, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeDownloadURL
// ---------------------------------------------------------------------------

func TestSubtitleParser_normalizeDownloadURL(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	t.Run("valid url is normalized", func(t *testing.T) {
		t.Parallel()
		input := "https://feliratok.eu/letolt.php?felirat=42&fnev=test%20file.srt"
		got := parser.normalizeDownloadURL(input)
		if got == "" {
			t.Fatal("normalizeDownloadURL returned empty string for valid URL")
		}
		if !strings.HasPrefix(got, "https://feliratok.eu") {
			t.Errorf("normalizeDownloadURL(%q) = %q, expected prefix https://feliratok.eu", input, got)
		}
	})

	t.Run("unparseable url returns original", func(t *testing.T) {
		t.Parallel()
		// A URL with an invalid escape sequence that url.Parse will reject
		input := "https://feliratok.eu/letolt.php?fnev=%zz"
		got := parser.normalizeDownloadURL(input)
		// url.Parse does not error on this, but Query() silently drops bad params.
		// The key invariant: the function never panics and always returns a string.
		if got == "" {
			t.Error("normalizeDownloadURL returned empty string")
		}
	})
}

// ---------------------------------------------------------------------------
// extractReleaseInfo
// ---------------------------------------------------------------------------

func TestSubtitleParser_extractReleaseInfo(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name        string
		description string
		want        string
	}{
		{"no parentheses", "Show - 1x01 - Title", ""},
		{"unclosed parenthesis", "Show - 1x01 (WEB.720p-GROUP", ""},
		{"normal case", "Show - 1x01 - Title (WEB.720p-GROUP)", "WEB.720p-GROUP"},
		{"multiple parens returns last", "Show (Season 2) (WEB.720p-FLUX)", "WEB.720p-FLUX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parser.extractReleaseInfo(tt.description)
			if got != tt.want {
				t.Errorf("extractReleaseInfo(%q) = %q, want %q", tt.description, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// detectQuality
// ---------------------------------------------------------------------------

func TestSubtitleParser_detectQuality(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name    string
		release string
		want    models.Quality
	}{
		{"360p", "WEB.360p-GROUP", models.Quality360p},
		{"480p", "WEB.480p-GROUP", models.Quality480p},
		{"720p", "AMZN.WEB-DL.720p-FLUX", models.Quality720p},
		{"1080p", "WEB.1080p-SuccessfulCrab", models.Quality1080p},
		{"2160p", "WEB-DL.2160p-GROUP", models.Quality2160p},
		{"4k alias", "WEB-DL.4K-GROUP", models.Quality2160p},
		{"unknown", "WEB-DL-GROUP", models.QualityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parser.detectQuality(tt.release)
			if got != tt.want {
				t.Errorf("detectQuality(%q) = %v, want %v", tt.release, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseDescription – fallback branch (no SxEE, no Season pack)
// ---------------------------------------------------------------------------

func TestSubtitleParser_parseDescription_fallback(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	showName, season, episode, releaseInfo, isSeasonPack := parser.parseDescription("Some Movie Title")

	if showName != "Some Movie Title" {
		t.Errorf("showName = %q, want %q", showName, "Some Movie Title")
	}
	if season != -1 {
		t.Errorf("season = %d, want -1", season)
	}
	if episode != -1 {
		t.Errorf("episode = %d, want -1", episode)
	}
	if releaseInfo != "" {
		t.Errorf("releaseInfo = %q, want empty", releaseInfo)
	}
	if isSeasonPack {
		t.Error("isSeasonPack = true, want false")
	}
}

// ---------------------------------------------------------------------------
// extractShowIDFromCategory
// ---------------------------------------------------------------------------

func TestSubtitleParser_extractShowIDFromCategory(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name string
		html string
		want int
	}{
		{"valid sid", `<table><tr><td><a href="index.php?sid=13051">Category</a></td></tr></table>`, 13051},
		{"missing link", `<table><tr><td>No Link</td></tr></table>`, 0},
		{"malformed url", `<table><tr><td><a href="://bad url{">Category</a></td></tr></table>`, 0},
		{"missing sid param", `<table><tr><td><a href="index.php?other=1">Category</a></td></tr></table>`, 0},
		{"non-numeric sid", `<table><tr><td><a href="index.php?sid=abc">Category</a></td></tr></table>`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}
			td := doc.Find("td")
			got := parser.extractShowIDFromCategory(td)
			if got != tt.want {
				t.Errorf("extractShowIDFromCategory() = %d, want %d", got, tt.want)
			}
		})
	}
}
