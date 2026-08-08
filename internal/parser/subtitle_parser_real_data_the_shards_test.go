package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
)

func TestSubtitleParser_ParseHtmlWithPagination_RealData_TheShards(t *testing.T) {
	t.Parallel()

	const (
		theShardsShowID   = 13340
		theShardsShowName = "The Shards"
		theShardsStatus   = "fordítás alatt (SubRip)"
	)

	type expectedSubtitle struct {
		id       int
		language string
		season   int
		episode  int
	}

	episodeRows := []struct {
		episode      int
		episodeTitle string
		filenameBase string
	}{
		{
			episode:      2,
			episodeTitle: "Don't You Want Me",
			filenameBase: "The.Shards.S01E02.Dont.You.Want.Me.DSNP.WEB-DL",
		},
		{
			episode:      1,
			episodeTitle: "Pilot",
			filenameBase: "The.Shards.S01E01.Pilot.DSNP.WEB-DL",
		},
	}

	languages := []struct {
		name           string
		isoCode        string
		uploader       string
		filenameSuffix string
		magyarSuffix   string
		subtitleIDs    map[int]int
	}{
		{
			name:           "Magyar",
			isoCode:        "hu",
			uploader:       "Anonymus",
			filenameSuffix: "hun",
			magyarSuffix:   " (SubRip)",
			subtitleIDs:    map[int]int{1: 1786002352, 2: 1786002369},
		},
		{
			name:           "Angol",
			isoCode:        "en",
			uploader:       "Kai",
			filenameSuffix: "eng",
			subtitleIDs:    map[int]int{1: 1786002343, 2: 1786002361},
		},
	}

	rows := make([]testutil.SubtitleRowOptions, 0, len(episodeRows)*len(languages))
	expected := make([]expectedSubtitle, 0, len(episodeRows)*len(languages))

	releaseGroupSuffix := " (DSNP.WEB-DL.720p-playWEB, DSNP.WEB-DL.720p-RAWR, DSNP.WEB-DL.1080p-FLUX, DSNP.WEB-DL.1080p-playWEB, DSNP.WEB-DL.1080p-RAWR)"

	for _, episodeRow := range episodeRows {
		episodeLabel := fmt.Sprintf("1x%02d", episodeRow.episode)
		eredetiTitle := theShardsShowName + " - " + episodeLabel + " - " + episodeRow.episodeTitle + releaseGroupSuffix

		for _, language := range languages {
			subtitleID := language.subtitleIDs[episodeRow.episode]

			rows = append(rows, testutil.SubtitleRowOptions{
				ShowID:           theShardsShowID,
				Language:         language.name,
				MagyarTitle:      "A szilánkok - " + episodeLabel + language.magyarSuffix,
				EredetiTitle:     eredetiTitle,
				Uploader:         language.uploader,
				UploadDate:       "2026-08-06",
				DownloadAction:   "letolt",
				DownloadFilename: episodeRow.filenameBase + "." + language.filenameSuffix + ".srt",
				SubtitleID:       subtitleID,
				Status:           theShardsStatus,
			})

			expected = append(expected, expectedSubtitle{
				id:       subtitleID,
				language: language.isoCode,
				season:   1,
				episode:  episodeRow.episode,
			})
		}
	}

	htmlContent := testutil.GenerateSubtitleTableHTML(rows)
	subtitleParser := NewSubtitleParser("https://feliratok.eu")
	result, err := subtitleParser.ParseHtmlWithPagination(strings.NewReader(htmlContent))
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
		if subtitle.ShowID != theShardsShowID {
			t.Errorf("subtitle %d: expected ShowID %d, got %d", i, theShardsShowID, subtitle.ShowID)
		}
		if subtitle.ShowName != theShardsShowName {
			t.Errorf("subtitle %d: expected show name %q, got %q", i, theShardsShowName, subtitle.ShowName)
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

	// Core assertion: episodes 1 and 2 of season 1 must both be present,
	// each with both Hungarian and English subtitle variants.
	type seasonEpisode struct {
		season  int
		episode int
	}

	languagesByEpisode := make(map[seasonEpisode]map[string]bool)
	for _, subtitle := range result.Subtitles {
		key := seasonEpisode{season: subtitle.Season, episode: subtitle.Episode}
		if languagesByEpisode[key] == nil {
			languagesByEpisode[key] = make(map[string]bool)
		}
		languagesByEpisode[key][subtitle.Language] = true
	}

	if len(languagesByEpisode) != 2 {
		t.Fatalf("Expected exactly 2 distinct season/episode pairs, got %d: %v", len(languagesByEpisode), languagesByEpisode)
	}

	for _, episode := range []int{1, 2} {
		key := seasonEpisode{season: 1, episode: episode}
		languages, found := languagesByEpisode[key]
		if !found {
			t.Errorf("Expected season 1 episode %d to be parsed, but it was not found", episode)
			continue
		}
		for _, isoCode := range []string{"hu", "en"} {
			if !languages[isoCode] {
				t.Errorf("Expected season 1 episode %d to have language %q variant", episode, isoCode)
			}
		}
	}
}
