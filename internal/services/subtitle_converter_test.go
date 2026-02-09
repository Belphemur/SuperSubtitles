package services

import (
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_BasicConversion(t *testing.T) {
	converter := NewSubtitleConverter()

	superSub := &models.SuperSubtitle{
		SubtitleID:   "1737417600",
		Name:         "Outlander - Az idegen - 7x16 (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab, AMZN.WEB-DL.1080p-FLUX)",
		Language:     "Magyar",
		Season:       "7",
		Episode:      "16",
		BaseLink:     "https://feliratok.eu",
		Uploader:     "kissoreg",
		IsSeasonPack: "0",
	}

	result := converter.ConvertSuperSubtitle(superSub)

	if result.ID != "1737417600" {
		t.Errorf("expected subtitle ID 1737417600, got %s", result.ID)
	}

	if result.Language != "hu" {
		t.Errorf("expected language 'hu', got '%s'", result.Language)
	}

	if result.Season != 7 {
		t.Errorf("expected season 7, got %d", result.Season)
	}

	if result.Episode != 16 {
		t.Errorf("expected episode 16, got %d", result.Episode)
	}

	if result.Uploader != "kissoreg" {
		t.Errorf("expected uploader 'kissoreg', got '%s'", result.Uploader)
	}

	if result.IsSeasonPack != false {
		t.Errorf("expected IsSeasonPack false, got %v", result.IsSeasonPack)
	}

	// Check that release groups are extracted (not just full name)
	if len(result.ReleaseGroups) == 0 {
		t.Errorf("expected release groups to be extracted, got empty")
	}

	releaseGroupMap := make(map[string]bool)
	for _, group := range result.ReleaseGroups {
		releaseGroupMap[group] = true
	}

	expectedGroups := []string{"AMZN", "FLUX", "SuccessfulCrab", "WEB-DL"}
	for _, expected := range expectedGroups {
		if !releaseGroupMap[expected] {
			t.Errorf("expected release group '%s' not found in %v", expected, result.ReleaseGroups)
		}
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_QualityExtraction(t *testing.T) {
	converter := NewSubtitleConverter()

	tests := []struct {
		name              string
		subtitleName      string
		expectedQualities []models.Quality
	}{
		{
			name:              "single 720p quality",
			subtitleName:      "Test Show - 1x01 (720p)",
			expectedQualities: []models.Quality{models.Quality720p},
		},
		{
			name:              "single 1080p quality",
			subtitleName:      "Test Show - 1x01 (1080p)",
			expectedQualities: []models.Quality{models.Quality1080p},
		},
		{
			name:              "multiple qualities in order",
			subtitleName:      "Outlander - 7x16 (720p, 1080p, 2160p)",
			expectedQualities: []models.Quality{models.Quality720p, models.Quality1080p, models.Quality2160p},
		},
		{
			name:              "4k quality",
			subtitleName:      "Billy the Kid - 3x07 (4K)",
			expectedQualities: []models.Quality{models.Quality2160p},
		},
		{
			name:              "no quality",
			subtitleName:      "Test Show - 1x01",
			expectedQualities: nil,
		},
		{
			name:              "duplicate qualities",
			subtitleName:      "Test Show - 1x01 (720p, 1080p, 720p)",
			expectedQualities: []models.Quality{models.Quality720p, models.Quality1080p},
		},
		{
			name:              "real world example with multiple releases",
			subtitleName:      "Billy the Kid - 3x07 (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR, AMZN.WEB-DL.2160p-RAWR)",
			expectedQualities: []models.Quality{models.Quality720p, models.Quality1080p, models.Quality2160p},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			superSub := &models.SuperSubtitle{
				SubtitleID:   "1234567890",
				Name:         tt.subtitleName,
				Language:     "Magyar",
				Season:       "1",
				Episode:      "1",
				BaseLink:     "https://feliratok.eu",
				Uploader:     "testuser",
				IsSeasonPack: "0",
			}

			result := converter.ConvertSuperSubtitle(superSub)

			if len(result.Qualities) != len(tt.expectedQualities) {
				t.Errorf("expected %d qualities, got %d: %v",
					len(tt.expectedQualities), len(result.Qualities), result.Qualities)
			}

			for i, expected := range tt.expectedQualities {
				if i >= len(result.Qualities) {
					t.Errorf("missing quality at index %d", i)
					continue
				}
				if result.Qualities[i] != expected {
					t.Errorf("quality at index %d: expected %v, got %v", i, expected, result.Qualities[i])
				}
			}
		})
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_ShowNameExtraction(t *testing.T) {
	converter := NewSubtitleConverter()

	tests := []struct {
		name         string
		subtitleName string
		expectedShow string
	}{
		{
			name:         "standard format with episode",
			subtitleName: "Outlander - Az idegen - 7x16 (720p)",
			expectedShow: "Outlander",
		},
		{
			name:         "show with dash in name",
			subtitleName: "Billy the Kid - 3x07 (720p)",
			expectedShow: "Billy the Kid",
		},
		{
			name:         "season pack format",
			subtitleName: "Billy the Kid (Season 1) (720p)",
			expectedShow: "Billy the Kid",
		},
		{
			name:         "season pack No episode",
			subtitleName: "Test Show (Season 2) (720p)",
			expectedShow: "Test Show",
		},
		{
			name:         "simple name no dash",
			subtitleName: "TestShow",
			expectedShow: "TestShow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			superSub := &models.SuperSubtitle{
				SubtitleID:   "1234567890",
				Name:         tt.subtitleName,
				Language:     "Magyar",
				Season:       "1",
				Episode:      "1",
				BaseLink:     "https://feliratok.eu",
				Uploader:     "testuser",
				IsSeasonPack: "0",
			}

			result := converter.ConvertSuperSubtitle(superSub)

			if result.ShowName != tt.expectedShow {
				t.Errorf("expected show name '%s', got '%s'", tt.expectedShow, result.ShowName)
			}
		})
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_LanguageConversion(t *testing.T) {
	converter := NewSubtitleConverter()

	tests := []struct {
		language string
		expected string
	}{
		{"Magyar", "hu"},
		{"Angol", "en"},
		{"Francia", "fr"},
		{"Német", "de"},
		{"Spanyol", "es"},
		{"Olasz", "it"},
		{"Portugál", "pt"},
		{"Brazíliai portugál", "pt"},
		{"Orosz", "ru"},
		{"Görög", "el"},
		{"Héber", "he"},
		{"Koreai", "ko"},
		{"Cseh", "cs"},
		{"Lengyel", "pl"},
		{"Horvát", "hr"},
		{"Szerb", "sr"},
		{"Román", "ro"},
		{"Szlovák", "sk"},
		{"Szlovén", "sl"},
		{"Török", "tr"},
		{"Dán", "da"},
		{"Svéd", "sv"},
		{"Norvég", "no"},
		{"Finn", "fi"},
		{"Flamand", "nl"},
		{"Holland", "nl"},
		{"Arab", "ar"},
		{"Albán", "sq"},
		{"Bolgár", "bg"},
		{"UnknownLanguage", "unknownlanguage"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			superSub := &models.SuperSubtitle{
				SubtitleID:   "1234567890",
				Name:         "Test Show - 1x01",
				Language:     tt.language,
				Season:       "1",
				Episode:      "1",
				BaseLink:     "https://feliratok.eu",
				Uploader:     "testuser",
				IsSeasonPack: "0",
			}

			result := converter.ConvertSuperSubtitle(superSub)

			if result.Language != tt.expected {
				t.Errorf("language '%s': expected '%s', got '%s'", tt.language, tt.expected, result.Language)
			}
		})
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_SeasonEpisodeConversion(t *testing.T) {
	converter := NewSubtitleConverter()

	tests := []struct {
		season          string
		episode         string
		expectedSeason  int
		expectedEpisode int
	}{
		{"1", "1", 1, 1},
		{"7", "16", 7, 16},
		{"10", "255", 10, 255},
		{"-1", "-1", -1, -1},
		{"-1", "1", -1, 1},
		{"0", "0", 0, 0},
	}

	for _, tt := range tests {
		superSub := &models.SuperSubtitle{
			SubtitleID:   "1234567890",
			Name:         "Test Show - 1x01",
			Language:     "Magyar",
			Season:       tt.season,
			Episode:      tt.episode,
			BaseLink:     "https://feliratok.eu",
			Uploader:     "testuser",
			IsSeasonPack: "0",
		}

		result := converter.ConvertSuperSubtitle(superSub)

		if result.Season != tt.expectedSeason {
			t.Errorf("season '%s': expected %d, got %d", tt.season, tt.expectedSeason, result.Season)
		}

		if result.Episode != tt.expectedEpisode {
			t.Errorf("episode '%s': expected %d, got %d", tt.episode, tt.expectedEpisode, result.Episode)
		}
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_SeasonPackDetection(t *testing.T) {
	converter := NewSubtitleConverter()

	tests := []struct {
		isSeasonPack string
		expected     bool
	}{
		{"0", false},
		{"1", true},
		{"anything else", false},
	}

	for _, tt := range tests {
		superSub := &models.SuperSubtitle{
			SubtitleID:   "1234567890",
			Name:         "Test Show",
			Language:     "Magyar",
			Season:       "1",
			Episode:      "1",
			BaseLink:     "https://feliratok.eu",
			Uploader:     "testuser",
			IsSeasonPack: tt.isSeasonPack,
		}

		result := converter.ConvertSuperSubtitle(superSub)

		if result.IsSeasonPack != tt.expected {
			t.Errorf("isSeasonPack '%s': expected %v, got %v", tt.isSeasonPack, tt.expected, result.IsSeasonPack)
		}
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_DownloadURL(t *testing.T) {
	converter := NewSubtitleConverter()

	tests := []struct {
		name     string
		baseLink string
		subID    string
		expected string
	}{
		{
			name:     "base link without /index.php",
			baseLink: "https://feliratok.eu",
			subID:    "1234567890",
			expected: "https://feliratok.eu/index.php?action=letolt&felirat=1234567890",
		},
		{
			name:     "base link with /index.php",
			baseLink: "https://feliratok.eu/index.php",
			subID:    "9876543210",
			expected: "https://feliratok.eu/index.php?action=letolt&felirat=9876543210",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			superSub := &models.SuperSubtitle{
				SubtitleID:   tt.subID,
				Name:         "Test Show - 1x01",
				Language:     "Magyar",
				Season:       "1",
				Episode:      "1",
				BaseLink:     tt.baseLink,
				Uploader:     "testuser",
				IsSeasonPack: "0",
			}

			result := converter.ConvertSuperSubtitle(superSub)

			if result.DownloadURL != tt.expected {
				t.Errorf("expected URL '%s', got '%s'", tt.expected, result.DownloadURL)
			}
		})
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_UploadTime(t *testing.T) {
	converter := NewSubtitleConverter()

	// Unix timestamp for 2025-01-21 00:00:00 UTC
	timestampStr := "1737417600"
	expectedTime := time.Unix(1737417600, 0)

	superSub := &models.SuperSubtitle{
		SubtitleID:   timestampStr,
		Name:         "Test Show - 1x01",
		Language:     "Magyar",
		Season:       "1",
		Episode:      "1",
		BaseLink:     "https://feliratok.eu",
		Uploader:     "testuser",
		IsSeasonPack: "0",
	}

	result := converter.ConvertSuperSubtitle(superSub)

	if !result.UploadedAt.Equal(expectedTime) {
		t.Errorf("expected upload time %v, got %v", expectedTime, result.UploadedAt)
	}
}

func TestDefaultSubtitleConverter_ConvertSuperSubtitle_UploadTimeInvalid(t *testing.T) {
	converter := NewSubtitleConverter()

	superSub := &models.SuperSubtitle{
		SubtitleID:   "not-a-timestamp",
		Name:         "Test Show - 1x01",
		Language:     "Magyar",
		Season:       "1",
		Episode:      "1",
		BaseLink:     "https://feliratok.eu",
		Uploader:     "testuser",
		IsSeasonPack: "0",
	}

	result := converter.ConvertSuperSubtitle(superSub)

	if !result.UploadedAt.IsZero() {
		t.Errorf("expected zero time for invalid timestamp, got %v", result.UploadedAt)
	}
}

func TestDefaultSubtitleConverter_ConvertResponse_Single(t *testing.T) {
	converter := NewSubtitleConverter()

	response := models.SuperSubtitleResponse{
		"1234567890": {
			SubtitleID:   "1234567890",
			Name:         "Outlander - Az idegen - 7x16 (720p)",
			Language:     "Magyar",
			Season:       "7",
			Episode:      "16",
			BaseLink:     "https://feliratok.eu",
			Uploader:     "kissoreg",
			IsSeasonPack: "0",
		},
	}

	result := converter.ConvertResponse(response)

	if result.Total != 1 {
		t.Errorf("expected 1 subtitle, got %d", result.Total)
	}

	if result.ShowName != "Outlander" {
		t.Errorf("expected show name 'Outlander', got '%s'", result.ShowName)
	}

	if len(result.Subtitles) != 1 {
		t.Errorf("expected 1 subtitle in collection, got %d", len(result.Subtitles))
	}

	if result.Subtitles[0].Language != "hu" {
		t.Errorf("expected language 'hu', got '%s'", result.Subtitles[0].Language)
	}
}

func TestDefaultSubtitleConverter_ConvertResponse_Multiple(t *testing.T) {
	converter := NewSubtitleConverter()

	response := models.SuperSubtitleResponse{
		"1234567890": {
			SubtitleID:   "1234567890",
			Name:         "Billy the Kid - 3x07 (720p, 1080p)",
			Language:     "Magyar",
			Season:       "3",
			Episode:      "7",
			BaseLink:     "https://feliratok.eu",
			Uploader:     "gricsi",
			IsSeasonPack: "0",
		},
		"9876543210": {
			SubtitleID:   "9876543210",
			Name:         "Billy the Kid - 3x07 (1080p, 2160p)",
			Language:     "Angol",
			Season:       "3",
			Episode:      "7",
			BaseLink:     "https://feliratok.eu",
			Uploader:     "J1GG4",
			IsSeasonPack: "0",
		},
	}

	result := converter.ConvertResponse(response)

	if result.Total != 2 {
		t.Errorf("expected 2 subtitles, got %d", result.Total)
	}

	if result.ShowName == "" {
		t.Errorf("expected show name to be populated, got empty string")
	}

	if len(result.Subtitles) != 2 {
		t.Errorf("expected 2 subtitles in collection, got %d", len(result.Subtitles))
	}

	// Check both languages are present
	languages := make(map[string]bool)
	for _, sub := range result.Subtitles {
		languages[sub.Language] = true
	}

	if !languages["hu"] || !languages["en"] {
		t.Errorf("expected both Hungarian and English subtitles")
	}
}

func TestDefaultSubtitleConverter_ConvertResponse_Empty(t *testing.T) {
	converter := NewSubtitleConverter()

	response := models.SuperSubtitleResponse{}

	result := converter.ConvertResponse(response)

	if result.Total != 0 {
		t.Errorf("expected 0 subtitles, got %d", result.Total)
	}

	if result.ShowName != "" {
		t.Errorf("expected empty show name, got '%s'", result.ShowName)
	}

	if len(result.Subtitles) != 0 {
		t.Errorf("expected 0 subtitles in collection, got %d", len(result.Subtitles))
	}
}

func TestDefaultSubtitleConverter_ExtractShowName(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard format",
			input:    "Outlander - Az idegen - 7x16",
			expected: "Outlander",
		},
		{
			name:     "format with quality",
			input:    "Billy the Kid - 3x07 (720p, 1080p)",
			expected: "Billy the Kid",
		},
		{
			name:     "season pack format",
			input:    "Billy the Kid (Season 2) (720p)",
			expected: "Billy the Kid",
		},
		{
			name:     "season pack no quality",
			input:    "Test Show (Season 1)",
			expected: "Test Show",
		},
		{
			name:     "simple name",
			input:    "SimpleShow",
			expected: "SimpleShow",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "name with multiple dashes",
			input:    "Show - With - Dashes - 1x01",
			expected: "Show",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.extractShowName(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDefaultSubtitleConverter_ExtractQualities(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		name     string
		input    string
		expected []models.Quality
	}{
		{
			name:     "single 720p",
			input:    "Show - 1x01 (720p)",
			expected: []models.Quality{models.Quality720p},
		},
		{
			name:     "single 1080p",
			input:    "Show - 1x01 (1080p)",
			expected: []models.Quality{models.Quality1080p},
		},
		{
			name:     "multiple qualities",
			input:    "Show - 1x01 (720p, 1080p, 2160p)",
			expected: []models.Quality{models.Quality720p, models.Quality1080p, models.Quality2160p},
		},
		{
			name:     "4k notation",
			input:    "Show - 1x01 (4k, 1080p)",
			expected: []models.Quality{models.Quality2160p, models.Quality1080p},
		},
		{
			name:     "no quality",
			input:    "Show - 1x01",
			expected: nil,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "real world example",
			input:    "Billy the Kid - 3x07 (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR, AMZN.WEB-DL.2160p-RAWR)",
			expected: []models.Quality{models.Quality720p, models.Quality1080p, models.Quality2160p},
		},
		{
			name:     "case insensitive",
			input:    "Show - 1x01 (720P, 1080P)",
			expected: []models.Quality{models.Quality720p, models.Quality1080p},
		},
		{
			name:     "duplicate removal",
			input:    "Show - 1x01 (720p, 720p, 1080p, 720p)",
			expected: []models.Quality{models.Quality720p, models.Quality1080p},
		},
		{
			name:     "all quality levels",
			input:    "Show (360p, 480p, 720p, 1080p, 2160p)",
			expected: []models.Quality{models.Quality360p, models.Quality480p, models.Quality720p, models.Quality1080p, models.Quality2160p},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.extractQualities(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d qualities, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("quality at index %d: expected %v, got %v", i, expected, result[i])
				}
			}
		})
	}
}

func TestDefaultSubtitleConverter_ConvertLanguageToISO(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		language string
		expected string
	}{
		{"Magyar", "hu"},
		{"Angol", "en"},
		{"Francia", "fr"},
		{"Német", "de"},
		{"Spanyol", "es"},
		{"UnknownLanguage", "unknownlanguage"},
		{"", ""},
	}

	for _, tt := range tests {
		result := converter.convertLanguageToISO(tt.language)
		if result != tt.expected {
			t.Errorf("language '%s': expected '%s', got '%s'", tt.language, tt.expected, result)
		}
	}
}

func TestDefaultSubtitleConverter_BuildDownloadURL(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		name     string
		baseLink string
		subID    string
		expected string
	}{
		{
			name:     "base URL",
			baseLink: "https://feliratok.eu",
			subID:    "123",
			expected: "https://feliratok.eu/index.php?action=letolt&felirat=123",
		},
		{
			name:     "URL with /index.php",
			baseLink: "https://feliratok.eu/index.php",
			subID:    "456",
			expected: "https://feliratok.eu/index.php?action=letolt&felirat=456",
		},
		{
			name:     "URL with trailing slash",
			baseLink: "https://feliratok.eu/",
			subID:    "789",
			expected: "https://feliratok.eu/index.php?action=letolt&felirat=789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.buildDownloadURL(tt.baseLink, tt.subID)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDefaultSubtitleConverter_ConvertSeasonNumber(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"7", 7},
		{"100", 100},
		{"-1", -1},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		result := converter.convertSeasonNumber(tt.input)
		if result != tt.expected {
			t.Errorf("season '%s': expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}

func TestDefaultSubtitleConverter_ConvertEpisodeNumber(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"16", 16},
		{"255", 255},
		{"-1", -1},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		result := converter.convertEpisodeNumber(tt.input)
		if result != tt.expected {
			t.Errorf("episode '%s': expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}

func TestDefaultSubtitleConverter_ConvertIsSeasonPack(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		input    string
		expected bool
	}{
		{"1", true},
		{"0", false},
		{"true", false},
		{"false", false},
		{"", false},
	}

	for _, tt := range tests {
		result := converter.convertIsSeasonPack(tt.input)
		if result != tt.expected {
			t.Errorf("isSeasonPack '%s': expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestDefaultSubtitleConverter_ExtractReleaseGroups(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		name                string
		input               string
		expectedContains    []string // at least these should be present
		expectedNotContains []string // these should not be present
	}{
		{
			name:             "outlander example",
			input:            "Outlander - Az idegen - 7x16 (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab, AMZN.WEB-DL.1080p-FLUX)",
			expectedContains: []string{"AMZN", "WEB-DL", "FLUX", "SuccessfulCrab", "WEB"},
		},
		{
			name:             "billy the kid example",
			input:            "Billy the Kid - 3x07 (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR, AMZN.WEB-DL.2160p-RAWR)",
			expectedContains: []string{"AMZN", "WEB-DL", "RAWR", "EDITH", "WEB"},
		},
		{
			name:                "simple release group",
			input:               "Show - 1x01 (720p-GROUP)",
			expectedContains:    []string{"GROUP"},
			expectedNotContains: []string{"720p"},
		},
		{
			name:                "source and group",
			input:               "Show - 1x01 (WEB.720p-RELEASEGRP)",
			expectedContains:    []string{"WEB", "RELEASEGRP"},
			expectedNotContains: []string{"720p"},
		},
		{
			name:             "no release groups",
			input:            "Show - 1x01",
			expectedContains: nil,
		},
		{
			name:             "subrip format",
			input:            "Show - 1x01 (SubRip)",
			expectedContains: nil,
		},
		{
			name:                "multiple sources with dashes",
			input:               "Show - 1x01 (WEB-DL.720p-GROUP1, AMZN.WEB-DL.1080p-GROUP2)",
			expectedContains:    []string{"WEB-DL", "GROUP1", "AMZN", "GROUP2"},
			expectedNotContains: []string{"720p", "1080p"},
		},
		{
			name:             "single dot-separated source",
			input:            "Show - 1x01 (NF.WEB-DL.720p-GROUP)",
			expectedContains: []string{"NF", "WEB-DL", "GROUP"},
		},
		{
			name:             "empty parentheses",
			input:            "Show - 1x01 ()",
			expectedContains: nil,
		},
		{
			name:                "only quality in parentheses",
			input:               "Show - 1x01 (720p)",
			expectedContains:    nil,
			expectedNotContains: []string{"720p"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.extractReleaseGroups(tt.input)

			if tt.expectedContains == nil {
				if len(result) > 0 {
					t.Errorf("expected no release groups, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected release groups %v, got nil", tt.expectedContains)
				return
			}

			// Check that all expected groups are present
			resultMap := make(map[string]bool)
			for _, group := range result {
				resultMap[group] = true
			}

			for _, expected := range tt.expectedContains {
				if !resultMap[expected] {
					t.Errorf("expected release group '%s' not found in %v", expected, result)
				}
			}

			// Check that unwanted groups are not present
			for _, notExpected := range tt.expectedNotContains {
				if resultMap[notExpected] {
					t.Errorf("unexpected release group '%s' found in %v", notExpected, result)
				}
			}
		})
	}
}

func TestDefaultSubtitleConverter_ExtractReleaseGroups_RealWorldExamples(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		name             string
		input            string
		expectedContains []string
	}{
		{
			name:             "netflix release",
			input:            "Hightown - S03E01 (NF.WEB-DL.720p-CAKES)",
			expectedContains: []string{"NF", "WEB-DL", "CAKES"},
		},
		{
			name:             "amazon prime",
			input:            "The Boys - 2x01 (AMZN.WEB-DL.1080p-FLUX)",
			expectedContains: []string{"AMZN", "WEB-DL", "FLUX"},
		},
		{
			name:             "multi-source with multiple groups",
			input:            "Show - 1x01 (WEB.720p-A, WEB.1080p-B, AMZN.2160p-C)",
			expectedContains: []string{"WEB", "A", "B", "AMZN", "C"},
		},
		{
			name:             "season pack example",
			input:            "Show (Season 1) (WEB.720p-GLHF, AMZN.1080p-PECULATE)",
			expectedContains: []string{"WEB", "GLHF", "AMZN", "PECULATE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.extractReleaseGroups(tt.input)

			if result == nil && len(tt.expectedContains) > 0 {
				t.Errorf("expected release groups %v, got nil", tt.expectedContains)
				return
			}

			resultMap := make(map[string]bool)
			for _, group := range result {
				resultMap[group] = true
			}

			for _, expected := range tt.expectedContains {
				if !resultMap[expected] {
					t.Errorf("expected release group '%s' not found in %v", expected, result)
				}
			}
		})
	}
}

func TestDefaultSubtitleConverter_ConvertUploadTime(t *testing.T) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)

	tests := []struct {
		name          string
		subtitleID    string
		expectedValid bool
		expectedUnix  int64
	}{
		{
			name:          "valid unix timestamp",
			subtitleID:    "1737417600",
			expectedValid: true,
			expectedUnix:  1737417600,
		},
		{
			name:          "zero timestamp",
			subtitleID:    "0",
			expectedValid: true,
			expectedUnix:  0,
		},
		{
			name:          "invalid timestamp",
			subtitleID:    "not-a-number",
			expectedValid: false,
		},
		{
			name:          "empty string",
			subtitleID:    "",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertUploadTime(tt.subtitleID)

			if tt.expectedValid {
				expectedTime := time.Unix(tt.expectedUnix, 0)
				if !result.Equal(expectedTime) {
					t.Errorf("expected time %v, got %v", expectedTime, result)
				}
			} else {
				if !result.IsZero() {
					t.Errorf("expected zero time, got %v", result)
				}
			}
		})
	}
}

func BenchmarkDefaultSubtitleConverter_ConvertSuperSubtitle(b *testing.B) {
	converter := NewSubtitleConverter()
	superSub := &models.SuperSubtitle{
		SubtitleID:   "1737417600",
		Name:         "Outlander - Az idegen - 7x16 (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab, AMZN.WEB-DL.1080p-FLUX)",
		Language:     "Magyar",
		Season:       "7",
		Episode:      "16",
		BaseLink:     "https://feliratok.eu",
		Uploader:     "kissoreg",
		IsSeasonPack: "0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = converter.ConvertSuperSubtitle(superSub)
	}
}

func BenchmarkDefaultSubtitleConverter_ConvertResponse(b *testing.B) {
	converter := NewSubtitleConverter()
	response := models.SuperSubtitleResponse{
		"1234567890": {
			SubtitleID:   "1234567890",
			Name:         "Outlander - 7x16 (720p, 1080p)",
			Language:     "Magyar",
			Season:       "7",
			Episode:      "16",
			BaseLink:     "https://feliratok.eu",
			Uploader:     "kissoreg",
			IsSeasonPack: "0",
		},
		"9876543210": {
			SubtitleID:   "9876543210",
			Name:         "Billy the Kid - 3x07 (1080p, 2160p)",
			Language:     "Angol",
			Season:       "3",
			Episode:      "7",
			BaseLink:     "https://feliratok.eu",
			Uploader:     "J1GG4",
			IsSeasonPack: "0",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = converter.ConvertResponse(response)
	}
}

func BenchmarkDefaultSubtitleConverter_ExtractQualities(b *testing.B) {
	converter := NewSubtitleConverter().(*DefaultSubtitleConverter)
	name := "Billy the Kid - 3x07 (AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-RAWR, AMZN.WEB-DL.2160p-RAWR)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = converter.extractQualities(name)
	}
}
