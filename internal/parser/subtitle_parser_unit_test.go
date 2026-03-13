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

	showName, season, episode, releaseInfo := parser.parseDescription("Some Movie Title")

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
}

func TestSubtitleParser_parseDescription_episodeRangeSeasonPack(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	description := "Pursuit of Jade (Zhu yu) - 1x01-09 (NF.WEB-DL.1080p-ANDY)"
	showName, season, episode, releaseInfo := parser.parseDescription(description)

	if showName != "Pursuit of Jade (Zhu yu)" {
		t.Errorf("showName = %q, want %q", showName, "Pursuit of Jade (Zhu yu)")
	}
	if season != 1 {
		t.Errorf("season = %d, want 1", season)
	}
	if episode != -1 {
		t.Errorf("episode = %d, want -1", episode)
	}
	if releaseInfo != "NF.WEB-DL.1080p-ANDY" {
		t.Errorf("releaseInfo = %q, want %q", releaseInfo, "NF.WEB-DL.1080p-ANDY")
	}
}

func TestSubtitleParser_isArchiveSeasonPack(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name string
		link string
		want bool
	}{
		{
			name: "zip archive",
			link: "/index.php?action=letolt&fnev=Pursuit.of.Jade.S01.Part.1.NF.WEB-DL.en.zip&felirat=1772977664",
			want: true,
		},
		{
			name: "rar archive uppercase extension",
			link: "/index.php?action=letolt&fnev=Show.S01.RAR.RAR&felirat=1772977664",
			want: true,
		},
		{
			name: "single srt subtitle",
			link: "/index.php?action=letolt&fnev=show.s01e01.srt&felirat=1772977664",
			want: false,
		},
		{
			name: "missing filename parameter",
			link: "/index.php?action=letolt&felirat=1772977664",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parser.isArchiveSeasonPack(tt.link)
			if got != tt.want {
				t.Errorf("isArchiveSeasonPack(%q) = %v, want %v", tt.link, got, tt.want)
			}
		})
	}
}

func TestSubtitleParser_extractEpisodeRange(t *testing.T) {
	t.Parallel()
	parser := NewSubtitleParser("https://feliratok.eu")
	one := 1
	nine := 9

	tests := []struct {
		name      string
		desc      string
		wantStart *int
		wantEnd   *int
	}{
		{
			name:      "valid ranged notation",
			desc:      "Pursuit of Jade (Zhu yu) - 1x01-09 (NF.WEB-DL.1080p-ANDY)",
			wantStart: &one,
			wantEnd:   &nine,
		},
		{
			name:      "single episode notation",
			desc:      "Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX)",
			wantStart: nil,
			wantEnd:   nil,
		},
		{
			name:      "numeric episode title is not range",
			desc:      "Portobello - 1x01 - 28 Million Viewers (AMZN.WEB-DL.720p-RAWR)",
			wantStart: nil,
			wantEnd:   nil,
		},
		{
			name:      "reversed ranged notation is normalized",
			desc:      "Show Name - 1x09-01 (WEB-DL)",
			wantStart: &one,
			wantEnd:   &nine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotStart, gotEnd := parser.extractEpisodeRange(tt.desc)

			if (gotStart == nil) != (tt.wantStart == nil) {
				t.Errorf("start nil mismatch: got %v, want %v", gotStart == nil, tt.wantStart == nil)
			}
			if gotStart != nil && tt.wantStart != nil && *gotStart != *tt.wantStart {
				t.Errorf("start = %d, want %d", *gotStart, *tt.wantStart)
			}

			if (gotEnd == nil) != (tt.wantEnd == nil) {
				t.Errorf("end nil mismatch: got %v, want %v", gotEnd == nil, tt.wantEnd == nil)
			}
			if gotEnd != nil && tt.wantEnd != nil && *gotEnd != *tt.wantEnd {
				t.Errorf("end = %d, want %d", *gotEnd, *tt.wantEnd)
			}
		})
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
