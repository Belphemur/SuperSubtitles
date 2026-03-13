package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
)

func TestSubtitleParser_ParseHtmlWithPagination_RealData_ThePitt(t *testing.T) {
	t.Parallel()

	type expectedSubtitle struct {
		id           int
		language     string
		season       int
		episode      int
		isSeasonPack bool
	}

	episodeRows := []struct {
		episode   int
		timeLabel string
		filename  string
	}{
		{episode: 9, timeLabel: "3:00 P.M.", filename: "The.Pitt.S02E09.300.P.M.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
		{episode: 8, timeLabel: "2:00 P.M.", filename: "The.Pitt.S02E08.200.P.M.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
		{episode: 7, timeLabel: "1:00 P.M.", filename: "The.Pitt.S02E07.100.P.M.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
		{episode: 6, timeLabel: "12:00 P.M.", filename: "The.Pitt.S02E06.1200.P.M.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
		{episode: 5, timeLabel: "11:00 A.M.", filename: "The.Pitt.S02E05.1100.A.M.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
		{episode: 4, timeLabel: "10:00 A.M.", filename: "The.Pitt.S02E04.1000.A.M.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
		{episode: 3, timeLabel: "9:00 A.M.", filename: "The.Pitt.S02E03.900.A.M.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
		{episode: 2, timeLabel: "8:00 A.M.", filename: "The.Pitt.S02E02.8.00.A.M.720p.AMZN.WEB-DL.DD+5.1.H.264-playWEB"},
		{episode: 1, timeLabel: "7:00 A.M.", filename: "The.Pitt.S02E01.700.A.M.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb"},
	}

	rows := make([]testutil.SubtitleRowOptions, 0, len(episodeRows)*2+2)
	expected := make([]expectedSubtitle, 0, len(episodeRows)*2+2)
	nextID := 1772814700

	for _, episodeRow := range episodeRows {
		for _, language := range []struct {
			name    string
			isoCode string
			suffix  string
		}{
			{name: "Magyar", isoCode: "hu", suffix: "hun"},
			{name: "Angol", isoCode: "en", suffix: "eng"},
		} {
			rows = append(rows, testutil.SubtitleRowOptions{
				ShowID:           11989,
				Language:         language.name,
				MagyarTitle:      "Vészhelyzet Pittsburghben - 2x" + fmt.Sprintf("%02d", episodeRow.episode),
				EredetiTitle:     "The Pitt - 2x" + fmt.Sprintf("%02d", episodeRow.episode) + " - " + episodeRow.timeLabel + " (AMZN.WEB-DL.720p-FLUX, AMZN.WEB-DL.1080p-NTb)",
				Uploader:         "Anonymus",
				UploadDate:       "2026-03-06",
				DownloadAction:   "letolt",
				DownloadFilename: episodeRow.filename + "." + language.suffix + ".srt",
				SubtitleID:       nextID,
			})

			expected = append(expected, expectedSubtitle{
				id:           nextID,
				language:     language.isoCode,
				season:       2,
				episode:      episodeRow.episode,
				isSeasonPack: false,
			})

			nextID++
		}
	}

	for _, language := range []struct {
		name    string
		isoCode string
		suffix  string
	}{
		{name: "Magyar", isoCode: "hu", suffix: "HUN"},
		{name: "Angol", isoCode: "en", suffix: "ENG"},
	} {
		rows = append(rows, testutil.SubtitleRowOptions{
			ShowID:           11989,
			Language:         language.name,
			MagyarTitle:      "Vészhelyzet Pittsburghben (1. évad)",
			EredetiTitle:     "The Pitt (Season 1) (AMZN.WEB-DL.720p-FLUX, AMZN.WEB-DL.1080p-NTb)",
			Uploader:         "J1GG4",
			UploadDate:       "2025-07-05",
			DownloadAction:   "letolt",
			DownloadFilename: "The.Pitt.S01.720p.AMZN.WEB-DL.DDP5.1.H.264-NTb." + language.suffix + ".zip",
			SubtitleID:       nextID,
		})

		expected = append(expected, expectedSubtitle{
			id:           nextID,
			language:     language.isoCode,
			season:       1,
			episode:      -1,
			isSeasonPack: true,
		})

		nextID++
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
		if subtitle.ShowID != 11989 {
			t.Errorf("subtitle %d: expected ShowID 11989, got %d", i, subtitle.ShowID)
		}
		if subtitle.ShowName != "The Pitt" {
			t.Errorf("subtitle %d: expected show name %q, got %q", i, "The Pitt", subtitle.ShowName)
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
		if subtitle.IsSeasonPack != exp.isSeasonPack {
			t.Errorf("subtitle %d: expected IsSeasonPack=%v, got %v", i, exp.isSeasonPack, subtitle.IsSeasonPack)
		}
	}
}
