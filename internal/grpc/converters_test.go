package grpc

import (
	"testing"
	"time"

	pb "github.com/Belphemur/SuperSubtitles/v2/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
)

// TestQualityConversion tests quality enum conversion
func TestQualityConversion(t *testing.T) {
	testCases := []struct {
		modelQuality models.Quality
		protoQuality pb.Quality
	}{
		{models.Quality360p, pb.Quality_QUALITY_360P},
		{models.Quality480p, pb.Quality_QUALITY_480P},
		{models.Quality720p, pb.Quality_QUALITY_720P},
		{models.Quality1080p, pb.Quality_QUALITY_1080P},
		{models.Quality2160p, pb.Quality_QUALITY_2160P},
		{models.QualityUnknown, pb.Quality_QUALITY_UNSPECIFIED},
	}

	for _, tc := range testCases {
		result := convertQualityToProto(tc.modelQuality)
		if result != tc.protoQuality {
			t.Errorf("Quality conversion failed: expected %v, got %v", tc.protoQuality, result)
		}
	}
}

// TestConvertShowToProto tests Show model to proto conversion
func TestConvertShowToProto(t *testing.T) {
	show := models.Show{
		Name:     "Breaking Bad",
		ID:       42,
		Year:     2008,
		ImageURL: "http://example.com/image.jpg",
	}

	result := convertShowToProto(show)

	if result.Name != "Breaking Bad" {
		t.Errorf("Expected name 'Breaking Bad', got '%s'", result.Name)
	}
	if result.Id != 42 {
		t.Errorf("Expected ID 42, got %d", result.Id)
	}
	if result.Year != 2008 {
		t.Errorf("Expected year 2008, got %d", result.Year)
	}
	if result.ImageUrl != "http://example.com/image.jpg" {
		t.Errorf("Expected image URL 'http://example.com/image.jpg', got '%s'", result.ImageUrl)
	}
}

// TestConvertShowFromProto_NilShow tests nil handling in show conversion
func TestConvertShowFromProto_NilShow(t *testing.T) {
	result := convertShowFromProto(nil)
	if result.ID != 0 || result.Name != "" {
		t.Errorf("Expected zero value Show, got %+v", result)
	}
}

// TestConvertShowFromProto tests proto Show to model conversion
func TestConvertShowFromProto(t *testing.T) {
	pbShow := &pb.Show{
		Name:     "Game of Thrones",
		Id:       123,
		Year:     2011,
		ImageUrl: "http://example.com/got.jpg",
	}

	result := convertShowFromProto(pbShow)

	if result.Name != "Game of Thrones" {
		t.Errorf("Expected name 'Game of Thrones', got '%s'", result.Name)
	}
	if result.ID != 123 {
		t.Errorf("Expected ID 123, got %d", result.ID)
	}
	if result.Year != 2011 {
		t.Errorf("Expected year 2011, got %d", result.Year)
	}
	if result.ImageURL != "http://example.com/got.jpg" {
		t.Errorf("Expected image URL 'http://example.com/got.jpg', got '%s'", result.ImageURL)
	}
}

// TestConvertThirdPartyIdsToProto tests ThirdPartyIds conversion
func TestConvertThirdPartyIdsToProto(t *testing.T) {
	ids := models.ThirdPartyIds{
		IMDBID:   "tt0903747",
		TVDBID:   81189,
		TVMazeID: 169,
		TraktID:  1388,
	}

	result := convertThirdPartyIdsToProto(ids)

	if result.ImdbId != "tt0903747" {
		t.Errorf("Expected IMDB ID 'tt0903747', got '%s'", result.ImdbId)
	}
	if result.TvdbId != 81189 {
		t.Errorf("Expected TVDB ID 81189, got %d", result.TvdbId)
	}
	if result.TvMazeId != 169 {
		t.Errorf("Expected TVMaze ID 169, got %d", result.TvMazeId)
	}
	if result.TraktId != 1388 {
		t.Errorf("Expected Trakt ID 1388, got %d", result.TraktId)
	}
}

// TestConvertSubtitleToProto tests subtitle conversion with valid timestamp
func TestConvertSubtitleToProto(t *testing.T) {
	uploadTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)
	subtitle := models.Subtitle{
		ID:            101,
		ShowID:        1,
		ShowName:      "Breaking Bad",
		Name:          "S01E01",
		Language:      "hun",
		Season:        1,
		Episode:       1,
		Filename:      "breaking.bad.s01e01.srt",
		DownloadURL:   "http://example.com/download/101",
		Uploader:      "testuser",
		UploadedAt:    uploadTime,
		Qualities:     []models.Quality{models.Quality720p, models.Quality1080p},
		ReleaseGroups: []string{"DIMENSION", "LOL"},
		Release:       "720p/1080p",
		IsSeasonPack:  false,
	}

	result := convertSubtitleToProto(subtitle)

	if result.Id != 101 {
		t.Errorf("Expected ID 101, got %d", result.Id)
	}
	if result.ShowId != 1 {
		t.Errorf("Expected ShowID 1, got %d", result.ShowId)
	}
	if result.Language != "hun" {
		t.Errorf("Expected language 'hun', got '%s'", result.Language)
	}
	if result.Season != 1 {
		t.Errorf("Expected season 1, got %d", result.Season)
	}
	if result.Episode != 1 {
		t.Errorf("Expected episode 1, got %d", result.Episode)
	}
	if result.UploadedAt == nil {
		t.Error("Expected non-nil UploadedAt")
	} else if !result.UploadedAt.AsTime().Equal(uploadTime) {
		t.Errorf("Expected upload time %v, got %v", uploadTime, result.UploadedAt.AsTime())
	}
	if len(result.Qualities) != 2 {
		t.Errorf("Expected 2 qualities, got %d", len(result.Qualities))
	}
	if result.Qualities[0] != pb.Quality_QUALITY_720P {
		t.Errorf("Expected first quality 720p, got %v", result.Qualities[0])
	}
	if result.Qualities[1] != pb.Quality_QUALITY_1080P {
		t.Errorf("Expected second quality 1080p, got %v", result.Qualities[1])
	}
	if len(result.ReleaseGroups) != 2 {
		t.Errorf("Expected 2 release groups, got %d", len(result.ReleaseGroups))
	}
	if result.IsSeasonPack {
		t.Error("Expected IsSeasonPack to be false")
	}
}

// TestConvertSubtitleToProto_ZeroTimestamp tests zero timestamp handling
func TestConvertSubtitleToProto_ZeroTimestamp(t *testing.T) {
	subtitle := models.Subtitle{
		ID:         101,
		ShowID:     1,
		Language:   "hun",
		UploadedAt: time.Time{}, // Zero value
	}

	result := convertSubtitleToProto(subtitle)
	if result.UploadedAt != nil {
		t.Error("Expected nil UploadedAt for zero time, got non-nil")
	}
}

func TestConvertShowSubtitlesToProto(t *testing.T) {
	uploadTime := time.Date(2024, 2, 5, 8, 15, 0, 0, time.UTC)
	ss := models.ShowSubtitles{
		Show: models.Show{
			Name:     "The Expanse",
			ID:       204,
			Year:     2015,
			ImageURL: "http://example.com/expanse.jpg",
		},
		ThirdPartyIds: models.ThirdPartyIds{
			IMDBID:   "tt3230854",
			TVDBID:   281620,
			TVMazeID: 151,
			TraktID:  11463,
		},
		SubtitleCollection: models.SubtitleCollection{
			ShowName: "The Expanse",
			Total:    2,
			Subtitles: []models.Subtitle{
				{
					ID:           3001,
					ShowID:       204,
					ShowName:     "The Expanse",
					Name:         "S03E05",
					Language:     "hun",
					Season:       3,
					Episode:      5,
					Filename:     "the.expanse.s03e05.srt",
					DownloadURL:  "http://example.com/download/3001",
					Uploader:     "subtitlefan",
					UploadedAt:   uploadTime,
					Qualities:    []models.Quality{models.Quality720p},
					Release:      "WEB-DL",
					IsSeasonPack: false,
				},
				{
					ID:       3002,
					ShowID:   204,
					ShowName: "The Expanse",
					Language: "eng",
					Season:   3,
					Episode:  5,
				},
			},
		},
	}

	result := convertShowSubtitlesToProto(ss)

	// Verify ShowInfo
	if result.ShowInfo == nil {
		t.Fatal("Expected ShowInfo to be set")
	}
	if result.ShowInfo.Show == nil {
		t.Fatal("Expected ShowInfo.Show to be set")
	}
	if result.ShowInfo.Show.Name != "The Expanse" {
		t.Errorf("Expected show name 'The Expanse', got '%s'", result.ShowInfo.Show.Name)
	}
	if result.ShowInfo.Show.Id != 204 {
		t.Errorf("Expected show ID 204, got %d", result.ShowInfo.Show.Id)
	}
	if result.ShowInfo.Show.Year != 2015 {
		t.Errorf("Expected show year 2015, got %d", result.ShowInfo.Show.Year)
	}
	if result.ShowInfo.ThirdPartyIds == nil {
		t.Fatal("Expected ThirdPartyIds to be set")
	}
	if result.ShowInfo.ThirdPartyIds.ImdbId != "tt3230854" {
		t.Errorf("Expected IMDB ID 'tt3230854', got '%s'", result.ShowInfo.ThirdPartyIds.ImdbId)
	}
	if result.ShowInfo.ThirdPartyIds.TvdbId != 281620 {
		t.Errorf("Expected TVDB ID 281620, got %d", result.ShowInfo.ThirdPartyIds.TvdbId)
	}

	// Verify Subtitles
	if len(result.Subtitles) != 2 {
		t.Fatalf("Expected 2 subtitles, got %d", len(result.Subtitles))
	}
	if result.Subtitles[0].Id != 3001 {
		t.Errorf("Expected first subtitle ID 3001, got %d", result.Subtitles[0].Id)
	}
	if result.Subtitles[0].Language != "hun" {
		t.Errorf("Expected language 'hun', got '%s'", result.Subtitles[0].Language)
	}
	if result.Subtitles[0].UploadedAt == nil {
		t.Fatal("Expected UploadedAt to be set")
	}
	if !result.Subtitles[0].UploadedAt.AsTime().Equal(uploadTime) {
		t.Errorf("Expected upload time %v, got %v", uploadTime, result.Subtitles[0].UploadedAt.AsTime())
	}
	if result.Subtitles[1].Id != 3002 {
		t.Errorf("Expected second subtitle ID 3002, got %d", result.Subtitles[1].Id)
	}
	if result.Subtitles[1].Language != "eng" {
		t.Errorf("Expected language 'eng', got '%s'", result.Subtitles[1].Language)
	}
}
