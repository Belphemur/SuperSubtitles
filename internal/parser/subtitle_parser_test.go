package parser

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/models"
	"github.com/Belphemur/SuperSubtitles/internal/testutil"
)

func TestSubtitleParser_ParseHtmlWithPagination_ExampleOutlander(t *testing.T) {
	// Generate proper HTML content based on the real feliratok.eu website structure
	htmlContent := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
		{
			ShowID:           2967,
			Language:         "Magyar",
			FlagImage:        "hungary.gif",
			MagyarTitle:      "Outlander - Az idegen - 7x16",
			EredetiTitle:     "Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab)",
			Uploader:         "kissoreg",
			UploaderBold:     false,
			UploadDate:       "2025-01-21",
			DownloadAction:   "letolt",
			DownloadFilename: "outlander.s07e16.srt",
			SubtitleID:       1737439811,
		},
	})

	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if len(result.Subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(result.Subtitles))
	}

	subtitle := result.Subtitles[0]
	if subtitle.Language != "hu" {
		t.Errorf("Expected language %q, got %q", "hu", subtitle.Language)
	}
	// The name should be only the episode title, extracted from the eredeti (original) title
	// which contains the pattern "Show - SxEE - Episode Title (Release Info)"
	expectedName := "A Hundred Thousand Angels"
	if subtitle.Name != expectedName {
		t.Errorf("Expected name %q, got %q", expectedName, subtitle.Name)
	}
	// Show name is extracted from eredeti, not magyar
	if subtitle.ShowName != "Outlander" {
		t.Errorf("Expected show name %q, got %q", "Outlander", subtitle.ShowName)
	}
	if subtitle.Season != 7 || subtitle.Episode != 16 {
		t.Errorf("Expected season 7 episode 16, got %d %d", subtitle.Season, subtitle.Episode)
	}
	if subtitle.IsSeasonPack {
		t.Errorf("Expected IsSeasonPack false")
	}

	// Verify URL components instead of exact string (parameter order may vary due to normalization)
	parsedURL, err := url.Parse(subtitle.DownloadURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	if parsedURL.Scheme != "https" || parsedURL.Host != "feliratok.eu" || parsedURL.Path != "/index.php" {
		t.Errorf("Invalid URL structure: %q", subtitle.DownloadURL)
	}
	params := parsedURL.Query()
	if params.Get("action") != "letolt" {
		t.Errorf("Expected action=letolt, got %q", params.Get("action"))
	}
	if params.Get("felirat") != "1737439811" {
		t.Errorf("Expected felirat=1737439811, got %q", params.Get("felirat"))
	}
	if params.Get("fnev") != "outlander.s07e16.srt" {
		t.Errorf("Expected fnev=outlander.s07e16.srt, got %q", params.Get("fnev"))
	}
	if subtitle.ID != 1737439811 {
		t.Errorf("Expected ID %d, got %d", 1737439811, subtitle.ID)
	}
	if subtitle.Filename != "outlander.s07e16.srt" {
		t.Errorf("Expected filename %q, got %q", "outlander.s07e16.srt", subtitle.Filename)
	}
	if subtitle.Uploader != "kissoreg" {
		t.Errorf("Expected uploader %q, got %q", "kissoreg", subtitle.Uploader)
	}

	expectedDate := time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC)
	if !subtitle.UploadedAt.Equal(expectedDate) {
		t.Errorf("Expected uploaded date %v, got %v", expectedDate, subtitle.UploadedAt)
	}

	expectedQualities := []models.Quality{models.Quality720p, models.Quality1080p}
	if !reflect.DeepEqual(subtitle.Qualities, expectedQualities) {
		t.Errorf("Expected qualities %v, got %v", expectedQualities, subtitle.Qualities)
	}

	expectedGroups := []string{"FLUX", "SuccessfulCrab"}
	if !reflect.DeepEqual(subtitle.ReleaseGroups, expectedGroups) {
		t.Errorf("Expected release groups %v, got %v", expectedGroups, subtitle.ReleaseGroups)
	}

	if subtitle.Release != "AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab" {
		t.Errorf("Expected release info %q, got %q", "AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab", subtitle.Release)
	}

	if result.CurrentPage != 1 || result.TotalPages != 1 || result.HasNextPage {
		t.Errorf("Expected pagination 1/1 with no next page, got %d/%d next=%v", result.CurrentPage, result.TotalPages, result.HasNextPage)
	}
}

func TestSubtitleParser_ParseHtmlWithPagination_SeasonPack(t *testing.T) {
	// Generate proper HTML for a season pack
	htmlContent := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
		{
			Language:         "Magyar",
			FlagImage:        "hungary.gif",
			MagyarTitle:      "Billy the Kid (11. évad)",
			EredetiTitle:     "Billy the Kid (Season 2) (WEB.720p-EDITH, AMZN.WEB-DL.2160p-RAWR)",
			Uploader:         "gricsi",
			UploaderBold:     false,
			UploadDate:       "2024-09-14",
			DownloadAction:   "letolt",
			DownloadFilename: "billy.s02.zip",
			SubtitleID:       1726325505,
		},
	})

	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if len(result.Subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(result.Subtitles))
	}

	subtitle := result.Subtitles[0]
	if subtitle.Language != "hu" {
		t.Errorf("Expected language %q, got %q", "hu", subtitle.Language)
	}
	if subtitle.ShowName != "Billy the Kid" {
		t.Errorf("Expected show name %q, got %q", "Billy the Kid", subtitle.ShowName)
	}
	if subtitle.Season != 2 || subtitle.Episode != -1 {
		t.Errorf("Expected season 2 episode -1, got %d %d", subtitle.Season, subtitle.Episode)
	}
	if !subtitle.IsSeasonPack {
		t.Errorf("Expected IsSeasonPack true")
	}

	expectedQualities := []models.Quality{models.Quality720p, models.Quality2160p}
	if !reflect.DeepEqual(subtitle.Qualities, expectedQualities) {
		t.Errorf("Expected qualities %v, got %v", expectedQualities, subtitle.Qualities)
	}

	expectedGroups := []string{"EDITH", "RAWR"}
	if !reflect.DeepEqual(subtitle.ReleaseGroups, expectedGroups) {
		t.Errorf("Expected release groups %v, got %v", expectedGroups, subtitle.ReleaseGroups)
	}

	if subtitle.Filename != "billy.s02.zip" {
		t.Errorf("Expected filename %q, got %q", "billy.s02.zip", subtitle.Filename)
	}
}

func TestSubtitleParser_ParseHtmlWithPagination_OldalPagination(t *testing.T) {
	// Generate proper HTML with oldal-based pagination
	htmlContent := testutil.GenerateSubtitleTableHTMLWithPagination(nil, 1, 3, true)

	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if result.CurrentPage != 1 || result.TotalPages != 3 || !result.HasNextPage {
		t.Errorf("Expected pagination 1/3 with next page, got %d/%d next=%v", result.CurrentPage, result.TotalPages, result.HasNextPage)
	}
}

func TestSubtitleParser_ParseHtml_ReturnsSubtitlesOnly(t *testing.T) {
	// Generate proper HTML content
	htmlContent := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
		{
			Language:         "Angol",
			FlagImage:        "uk.gif",
			MagyarTitle:      "Outlander - Az idegen - 7x15",
			EredetiTitle:     "Outlander - 7x15 - Written in My Own Heart's Blood (AMZN.WEB-DL.720p-NTb)",
			Uploader:         "J1GG4",
			UploaderBold:     false,
			UploadDate:       "2025-01-17",
			DownloadAction:   "letolt",
			DownloadFilename: "outlander.s07e15.srt",
			SubtitleID:       1737139076,
		},
	})

	parser := NewSubtitleParser("https://feliratok.eu")
	subtitles, err := parser.ParseHtml(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtml failed: %v", err)
	}

	if len(subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(subtitles))
	}
}

func TestSubtitleParser_ExtractFilenameFromDownloadLink_URLEncoded(t *testing.T) {
	parser := NewSubtitleParser("https://feliratok.eu")

	tests := []struct {
		name     string
		link     string
		expected string
	}{
		{
			name:     "URL encoded filename with spaces and special chars",
			link:     "/index.php?action=letolt&fnev=Billy%20The%20Kid%20-%2003x04%20-%20The%20Shepherds%20Hut.EDITH.English.C.orig.Addic7ed.com.srt&felirat=1760949698",
			expected: "Billy The Kid - 03x04 - The Shepherds Hut.EDITH.English.C.orig.Addic7ed.com.srt",
		},
		{
			name:     "URL encoded with parentheses",
			link:     "/index.php?action=letolt&fnev=Show%20Name%20%282024%29.srt&felirat=123456",
			expected: "Show Name (2024).srt",
		},
		{
			name:     "Simple filename without encoding",
			link:     "/index.php?action=letolt&fnev=outlander.s07e16.srt&felirat=1737439811",
			expected: "outlander.s07e16.srt",
		},
		{
			name:     "No fnev parameter",
			link:     "/index.php?action=letolt&felirat=123456",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractFilenameFromDownloadLink(tt.link)
			if result != tt.expected {
				t.Errorf("Expected filename %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertLanguageToISO(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Hungarian language names
		{name: "Hungarian - Magyar", input: "Magyar", expected: "hu"},
		{name: "Hungarian - magyar lowercase", input: "magyar", expected: "hu"},
		{name: "English - Angol", input: "Angol", expected: "en"},
		{name: "English - angol lowercase", input: "angol", expected: "en"},
		{name: "German - Német", input: "Német", expected: "de"},
		{name: "French - Francia", input: "Francia", expected: "fr"},
		{name: "Spanish - Spanyol", input: "Spanyol", expected: "es"},
		{name: "Italian - Olasz", input: "Olasz", expected: "it"},
		{name: "Russian - Orosz", input: "Orosz", expected: "ru"},
		{name: "Portuguese - Portugál", input: "Portugál", expected: "pt"},
		{name: "Dutch - Holland", input: "Holland", expected: "nl"},
		{name: "Polish - Lengyel", input: "Lengyel", expected: "pl"},
		{name: "Turkish - Török", input: "Török", expected: "tr"},
		{name: "Arabic - Arab", input: "Arab", expected: "ar"},
		{name: "Hebrew - Héber", input: "Héber", expected: "he"},
		{name: "Japanese - Japán", input: "Japán", expected: "ja"},
		{name: "Chinese - Kínai", input: "Kínai", expected: "zh"},
		{name: "Korean - Koreai", input: "Koreai", expected: "ko"},
		{name: "Czech - Cseh", input: "Cseh", expected: "cs"},
		{name: "Danish - Dán", input: "Dán", expected: "da"},
		{name: "Finnish - Finn", input: "Finn", expected: "fi"},
		{name: "Greek - Görög", input: "Görög", expected: "el"},
		{name: "Norwegian - Norvég", input: "Norvég", expected: "no"},
		{name: "Swedish - Svéd", input: "Svéd", expected: "sv"},
		{name: "Romanian - Román", input: "Román", expected: "ro"},
		{name: "Serbian - Szerb", input: "Szerb", expected: "sr"},
		{name: "Croatian - Horvát", input: "Horvát", expected: "hr"},
		{name: "Bulgarian - Bolgár", input: "Bolgár", expected: "bg"},
		{name: "Ukrainian - Ukrán", input: "Ukrán", expected: "uk"},
		{name: "Thai - Thai", input: "Thai", expected: "th"},
		{name: "Vietnamese - Vietnámi", input: "Vietnámi", expected: "vi"},
		{name: "Indonesian - Indonéz", input: "Indonéz", expected: "id"},
		{name: "Hindi - Hindi", input: "Hindi", expected: "hi"},
		{name: "Persian - Perzsa", input: "Perzsa", expected: "fa"},
		{name: "Brazilian - Brazil", input: "Brazil", expected: "pt"},

		// English language names (fallback)
		{name: "English name - Hungarian", input: "Hungarian", expected: "hu"},
		{name: "English name - English", input: "English", expected: "en"},
		{name: "English name - German", input: "German", expected: "de"},
		{name: "English name - French", input: "French", expected: "fr"},
		{name: "English name - Spanish", input: "Spanish", expected: "es"},
		{name: "English name - Portuguese", input: "Portuguese", expected: "pt"},

		// Edge cases
		{name: "Empty string", input: "", expected: ""},
		{name: "Whitespace only", input: "   ", expected: ""},
		{name: "Already ISO code - en", input: "en", expected: "en"},
		{name: "Already ISO code - hu", input: "hu", expected: "hu"},
		{name: "Already ISO code - uppercase", input: "EN", expected: "en"},
		{name: "Mixed case", input: "MaGyAr", expected: "hu"},
		{name: "With leading/trailing spaces", input: "  Angol  ", expected: "en"},

		// Unknown language (should return original)
		{name: "Unknown language", input: "Klingon", expected: "Klingon"},
		{name: "Numeric input", input: "12345", expected: "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLanguageToISO(tt.input)
			if result != tt.expected {
				t.Errorf("convertLanguageToISO(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveParentheticalContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Episode with release info",
			input:    "The Copenhagen Test - 1x04 - Obsidian (WEB.720p-SYLiX, AMZN.WEB-DL.720p-Kitsune, AMZN.WEB-DL.720p-RAWR, PCOK.WEB-DL.720p-playWEB, WEB.1080p-ETHEL, AMZN.WEB-DL.1080p-Kitsune, AMZN.WEB-DL.1080p-RAWR, PCOK.WEB-DL.1080p-BLOOM, PCOK.WEB-DL.1080p-playWEB, WEB.2160p-ETHEL, AMZN.WEB-DL.2160p-RAWR, PCOK.WEB-DL.2160p-playWEB, PCOK.WEB-DL.2160p-RAWR)",
			expected: "The Copenhagen Test - 1x04 - Obsidian",
		},
		{
			name:     "Season pack with release info",
			input:    "Pocoyo (Season 4) (NF.WEBRip)",
			expected: "Pocoyo",
		},
		{
			name:     "Single parenthetical content",
			input:    "Billy the Kid (Season 2) (WEB.720p-EDITH, AMZN.WEB-DL.2160p-RAWR)",
			expected: "Billy the Kid",
		},
		{
			name:     "Outlander example",
			input:    "Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab)",
			expected: "Outlander - 7x16 - A Hundred Thousand Angels",
		},
		{
			name:     "No parentheses",
			input:    "Show Name - 1x01 - Episode Title",
			expected: "Show Name - 1x01 - Episode Title",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only parentheses",
			input:    "(Release Info)",
			expected: "",
		},
		{
			name:     "Multiple separate parentheses",
			input:    "Show (Year) - Episode (Release)",
			expected: "Show - Episode",
		},
		{
			name:     "Trailing dash after removal",
			input:    "Show Name -",
			expected: "Show Name",
		},
		{
			name:     "Trailing dash and space after parentheses",
			input:    "Show Name - (Release Info)",
			expected: "Show Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeParentheticalContent(tt.input)
			if result != tt.expected {
				t.Errorf("removeParentheticalContent(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractEpisodeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Outlander episode",
			input:    "Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab)",
			expected: "A Hundred Thousand Angels",
		},
		{
			name:     "Outlander episode with duplicate SxEE",
			input:    "Outlander - Az idegen - 7x16 Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab)",
			expected: "A Hundred Thousand Angels",
		},
		{
			name:     "The Copenhagen Test",
			input:    "The Copenhagen Test - 1x04 - Obsidian (WEB.720p-SYLiX, AMZN.WEB-DL.720p-Kitsune)",
			expected: "Obsidian",
		},
		{
			name:     "Episode with multiple dashes in title",
			input:    "Show - 2x05 - Title With - Many - Dashes (Release)",
			expected: "Title With - Many - Dashes",
		},
		{
			name:     "Episode with parentheses in title",
			input:    "Show - 1x01 - Episode (Part 1) (Release Info)",
			expected: "Episode",
		},
		{
			name:     "Season pack",
			input:    "Billy the Kid (Season 2) (WEB.720p-EDITH, AMZN.WEB-DL.2160p-RAWR)",
			expected: "",
		},
		{
			name:     "No dashes, just text",
			input:    "Simple Episode (Release)",
			expected: "Simple Episode",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only parentheses",
			input:    "(Release Info)",
			expected: "",
		},
		{
			name:     "Ends with dash",
			input:    "Show - 1x01 - (Release Info)",
			expected: "",
		},
		{
			name:     "No parentheses",
			input:    "Show - 1x01 - Perfect Episode Name",
			expected: "Perfect Episode Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEpisodeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("extractEpisodeTitle(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSubtitleParser_ExtractShowIDFromHTML(t *testing.T) {
	// Test that show ID is correctly extracted from the main page HTML
	htmlContent := testutil.GenerateSubtitleTableHTML([]testutil.SubtitleRowOptions{
		{
			ShowID:           13051,
			Language:         "Magyar",
			FlagImage:        "hungary.gif",
			MagyarTitle:      "The Copenhagen Test - 1x04 (SubRip)",
			EredetiTitle:     "The Copenhagen Test - 1x04 - Obsidian (WEB.720p-SYLiX)",
			Uploader:         "Anonymus",
			UploaderBold:     false,
			UploadDate:       "2026-02-09",
			DownloadAction:   "letolt",
			DownloadFilename: "The.Copenhagen.Test.S01E04.srt",
			SubtitleID:       1770617276,
		},
		{
			ShowID:           11930,
			Language:         "Magyar",
			FlagImage:        "hungary.gif",
			MagyarTitle:      "Három hónap jegyesség - a másik út - 7x18 (SubRip)",
			EredetiTitle:     "90 Day Fiancé: The Other Way - 7x18 - Adios (HMAX.WEBRip)",
			Uploader:         "Anonymus",
			UploaderBold:     false,
			UploadDate:       "2026-02-08",
			DownloadAction:   "letolt",
			DownloadFilename: "90.Day.Fiance.The.Other.Way.S07E18.srt",
			SubtitleID:       1770577432,
		},
	})

	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if len(result.Subtitles) != 2 {
		t.Fatalf("Expected 2 subtitles, got %d", len(result.Subtitles))
	}

	// Test first subtitle
	subtitle1 := result.Subtitles[0]
	if subtitle1.ShowID != 13051 {
		t.Errorf("Expected ShowID %d, got %d", 13051, subtitle1.ShowID)
	}
	if subtitle1.ID != 1770617276 {
		t.Errorf("Expected ID %d, got %d", 1770617276, subtitle1.ID)
	}

	// Test second subtitle
	subtitle2 := result.Subtitles[1]
	if subtitle2.ShowID != 11930 {
		t.Errorf("Expected ShowID %d, got %d", 11930, subtitle2.ShowID)
	}
	if subtitle2.ID != 1770577432 {
		t.Errorf("Expected ID %d, got %d", 1770577432, subtitle2.ID)
	}
}
