package services

import (
	"testing"
	"time"

	"SuperSubtitles/internal/models"
)

func TestNewSubtitleConverter(t *testing.T) {
	converter := NewSubtitleConverter()
	if converter == nil {
		t.Fatal("NewSubtitleConverter should return a non-nil converter")
	}

	// Verify it implements the interface
	var _ SubtitleConverter = converter //nolint:staticcheck // explicit interface compliance check
}

func TestConvertSuperSubtitle(t *testing.T) {
	converter := NewSubtitleConverter()

	tests := []struct {
		name     string
		input    models.SuperSubtitle
		expected models.Subtitle
	}{
		{
			name: "basic conversion",
			input: models.SuperSubtitle{
				Language:     "Angol",
				Name:         "Outlander (Season 1) (1080p)",
				BaseLink:     "https://feliratok.eu/index.php",
				Filename:     "Outlander.S01.HDTV.720p.1080p.ENG.zip",
				SubtitleID:   "1435431909",
				Season:       "1",
				Episode:      "1",
				Uploader:     "J1GG4",
				ExactMatch:   "111",
				IsSeasonPack: "0",
			},
			expected: models.Subtitle{
				ID:           "1435431909",
				ShowName:     "Outlander",
				Language:     "en",
				Season:       1,
				Episode:      1,
				Filename:     "Outlander.S01.HDTV.720p.1080p.ENG.zip",
				DownloadURL:  "https://feliratok.eu/index.php/index.php?action=letolt&felirat=1435431909",
				Uploader:     "J1GG4",
				UploadedAt:   time.Unix(1435431909, 0),
				Quality:      models.Quality1080p,
				ReleaseGroup: "Outlander (Season 1) (1080p)",
				Source:       "Outlander (Season 1) (1080p)",
				IsSeasonPack: false,
				ExactMatch:   111,
			},
		},
		{
			name: "season pack conversion",
			input: models.SuperSubtitle{
				Language:     "Magyar",
				Name:         "Outlander - 7x01 (AMZN.WEB-DL.720p-NTb)",
				BaseLink:     "https://feliratok.eu/index.php",
				Filename:     "Outlander.S07E01.srt",
				SubtitleID:   "1686999476",
				Season:       "-1",
				Episode:      "-1",
				Uploader:     "kissoreg",
				ExactMatch:   "010",
				IsSeasonPack: "1",
			},
			expected: models.Subtitle{
				ID:           "1686999476",
				ShowName:     "Outlander",
				Language:     "hu",
				Season:       -1,
				Episode:      -1,
				Filename:     "Outlander.S07E01.srt",
				DownloadURL:  "https://feliratok.eu/index.php/index.php?action=letolt&felirat=1686999476",
				Uploader:     "kissoreg",
				UploadedAt:   time.Unix(1686999476, 0),
				Quality:      models.Quality720p,
				ReleaseGroup: "Outlander - 7x01 (AMZN.WEB-DL.720p-NTb)",
				Source:       "Outlander - 7x01 (AMZN.WEB-DL.720p-NTb)",
				IsSeasonPack: true,
				ExactMatch:   10,
			},
		},
		{
			name: "unknown quality and language",
			input: models.SuperSubtitle{
				Language:     "Unknown Language",
				Name:         "Some Movie",
				BaseLink:     "https://test.com",
				Filename:     "movie.srt",
				SubtitleID:   "123456789",
				Season:       "0",
				Episode:      "0",
				Uploader:     "test",
				ExactMatch:   "0",
				IsSeasonPack: "0",
			},
			expected: models.Subtitle{
				ID:           "123456789",
				ShowName:     "Some Movie",
				Language:     "unknown language",
				Season:       0,
				Episode:      0,
				Filename:     "movie.srt",
				DownloadURL:  "https://test.com/index.php?action=letolt&felirat=123456789",
				Uploader:     "test",
				UploadedAt:   time.Unix(123456789, 0),
				Quality:      models.QualityUnknown,
				ReleaseGroup: "Some Movie",
				Source:       "Some Movie",
				IsSeasonPack: false,
				ExactMatch:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertSuperSubtitle(&tt.input)

			if result.ID != tt.expected.ID {
				t.Errorf("ID: expected %s, got %s", tt.expected.ID, result.ID)
			}
			if result.ShowName != tt.expected.ShowName {
				t.Errorf("ShowName: expected %s, got %s", tt.expected.ShowName, result.ShowName)
			}
			if result.Language != tt.expected.Language {
				t.Errorf("Language: expected %s, got %s", tt.expected.Language, result.Language)
			}
			if result.Season != tt.expected.Season {
				t.Errorf("Season: expected %d, got %d", tt.expected.Season, result.Season)
			}
			if result.Episode != tt.expected.Episode {
				t.Errorf("Episode: expected %d, got %d", tt.expected.Episode, result.Episode)
			}
			if result.Quality != tt.expected.Quality {
				t.Errorf("Quality: expected %v, got %v", tt.expected.Quality, result.Quality)
			}
			if result.IsSeasonPack != tt.expected.IsSeasonPack {
				t.Errorf("IsSeasonPack: expected %t, got %t", tt.expected.IsSeasonPack, result.IsSeasonPack)
			}
			if result.ExactMatch != tt.expected.ExactMatch {
				t.Errorf("ExactMatch: expected %d, got %d", tt.expected.ExactMatch, result.ExactMatch)
			}
		})
	}
}

func TestConvertResponse(t *testing.T) {
	converter := NewSubtitleConverter()

	response := models.SuperSubtitleResponse{
		"1": models.SuperSubtitle{
			Language:     "Angol",
			Name:         "Outlander (Season 1) (1080p)",
			BaseLink:     "https://feliratok.eu/index.php",
			Filename:     "Outlander.S01.HDTV.720p.1080p.ENG.zip",
			SubtitleID:   "1435431909",
			Season:       "1",
			Episode:      "1",
			Uploader:     "J1GG4",
			ExactMatch:   "111",
			IsSeasonPack: "0",
		},
		"2": models.SuperSubtitle{
			Language:     "Magyar",
			Name:         "Outlander (Season 1) (720p)",
			BaseLink:     "https://feliratok.eu/index.php",
			Filename:     "Outlander.S01.HDTV.720p.HUN.zip",
			SubtitleID:   "1435431932",
			Season:       "1",
			Episode:      "-1",
			Uploader:     "BCsilla",
			ExactMatch:   "111",
			IsSeasonPack: "1",
		},
	}

	result := converter.ConvertResponse(response)

	if result.Total != 2 {
		t.Errorf("Total: expected 2, got %d", result.Total)
	}
	if len(result.Subtitles) != 2 {
		t.Errorf("Subtitles length: expected 2, got %d", len(result.Subtitles))
	}
	if result.ShowName != "Outlander" {
		t.Errorf("ShowName: expected 'Outlander', got %s", result.ShowName)
	}

	// Build a map of subtitles by language for order-independent assertions
	// (SuperSubtitleResponse is a map so iteration order is non-deterministic)
	subtitlesByLang := make(map[string]models.Subtitle)
	for _, s := range result.Subtitles {
		subtitlesByLang[s.Language] = s
	}

	if en, ok := subtitlesByLang["en"]; !ok {
		t.Error("Expected English subtitle not found")
	} else {
		if en.Quality != models.Quality1080p {
			t.Errorf("English subtitle quality: expected Quality1080p, got %v", en.Quality)
		}
	}
}

// Benchmark tests
func BenchmarkConvertSuperSubtitle(b *testing.B) {
	converter := NewSubtitleConverter()
	superSub := &models.SuperSubtitle{
		Language:     "Angol",
		Name:         "Outlander (Season 1) (1080p)",
		BaseLink:     "https://feliratok.eu/index.php",
		Filename:     "Outlander.S01.HDTV.720p.1080p.ENG.zip",
		SubtitleID:   "1435431909",
		Season:       "1",
		Episode:      "1",
		Uploader:     "J1GG4",
		ExactMatch:   "111",
		IsSeasonPack: "0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.ConvertSuperSubtitle(superSub)
	}
}
