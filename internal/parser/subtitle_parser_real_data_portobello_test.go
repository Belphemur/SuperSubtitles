package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
)

func TestSubtitleParser_ParseHtmlWithPagination_RealData_Portobello(t *testing.T) {
	t.Parallel()

	type expectedSubtitle struct {
		id       int
		language string
		season   int
		episode  int
	}

	episodeRows := []struct {
		episode int
		title   string
	}{
		{episode: 4, title: "Column of Infamy"},
		{episode: 3, title: "Infallible"},
		{episode: 2, title: "Jail"},
		{episode: 1, title: "28 Million Viewers"},
	}

	rows := make([]testutil.SubtitleRowOptions, 0, len(episodeRows)*3)
	expected := make([]expectedSubtitle, 0, len(episodeRows)*3)
	nextID := 1773431967

	for _, episodeRow := range episodeRows {
		for _, language := range []struct {
			name    string
			isoCode string
			suffix  string
		}{
			{name: "Magyar", isoCode: "hu", suffix: "hu"},
			{name: "Olasz", isoCode: "it", suffix: "it"},
			{name: "Angol", isoCode: "en", suffix: "en"},
		} {
			rows = append(rows, testutil.SubtitleRowOptions{
				ShowID:           13108,
				Language:         language.name,
				MagyarTitle:      "Portobello - 1x" + fmt.Sprintf("%02d", episodeRow.episode),
				EredetiTitle:     "Portobello - 1x" + fmt.Sprintf("%02d", episodeRow.episode) + " - " + episodeRow.title + " (AMZN.WEB-DL.720p-playWEB, AMZN.WEB-DL.720p-RAWR, WEB.1080p-EDITH, AMZN.WEB-DL.1080p-playWEB, AMZN.WEB-DL.1080p-RAWR, WEB.2160p-EDITH, HMAX.WEB-DL.2160p-playWEB, HMAX.WEB-DL.2160p-RAWR)",
				Uploader:         "J1GG4",
				UploadDate:       "2026-03-13",
				DownloadAction:   "letolt",
				DownloadFilename: "Portobello.S01E" + fmt.Sprintf("%02d", episodeRow.episode) + "." + strings.ReplaceAll(episodeRow.title, " ", ".") + ".HMAX.WEB-DL." + language.suffix + ".srt",
				SubtitleID:       nextID,
			})

			expected = append(expected, expectedSubtitle{
				id:       nextID,
				language: language.isoCode,
				season:   1,
				episode:  episodeRow.episode,
			})

			nextID--
		}
	}

	htmlContent := testutil.GenerateSubtitleTableHTML(rows)
	parser := NewSubtitleParser("https://feliratok.eu")
	result, err := parser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseHtmlWithPagination failed: %v", err)
	}

	if len(result.Subtitles) != len(expected) {
		t.Fatalf("Expected %d subtitles, got %d", len(expected), len(result.Subtitles))
	}

	for i, subtitle := range result.Subtitles {
		exp := expected[i]

		if subtitle.ID != exp.id {
			t.Errorf("subtitle %d: expected ID %d, got %d", i, exp.id, subtitle.ID)
		}
		if subtitle.ShowID != 13108 {
			t.Errorf("subtitle %d: expected ShowID 13108, got %d", i, subtitle.ShowID)
		}
		if subtitle.ShowName != "Portobello" {
			t.Errorf("subtitle %d: expected show name %q, got %q", i, "Portobello", subtitle.ShowName)
		}
		if subtitle.Language != exp.language {
			t.Errorf("subtitle %d: expected language %q, got %q", i, exp.language, subtitle.Language)
		}
		if subtitle.Season != exp.season || subtitle.Episode != exp.episode {
			t.Errorf(
				"subtitle %d: expected season/episode %d/%d, got %d/%d",
				i,
				exp.season,
				exp.episode,
				subtitle.Season,
				subtitle.Episode,
			)
		}
		if subtitle.IsSeasonPack {
			t.Errorf("subtitle %d: expected IsSeasonPack=false, got true", i)
		}
	}
}
