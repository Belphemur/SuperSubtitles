package grpc

import (
	"math"
	"testing"
	"time"

	pb "github.com/Belphemur/SuperSubtitles/v2/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
)

// TestQualityConversion tests quality enum conversion
func TestQualityConversion(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	result := convertShowFromProto(nil)
	if result.ID != 0 || result.Name != "" {
		t.Errorf("Expected zero value Show, got %+v", result)
	}
}

// TestConvertShowFromProto tests proto Show to model conversion
func TestConvertShowFromProto(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestSanitizeUTF8_ValidString tests that valid UTF-8 strings pass through unchanged
func TestSanitizeUTF8_ValidString(t *testing.T) {
	t.Parallel()
	testCases := []string{
		"Breaking Bad",
		"Magyar felirat",
		"日本語",
		"Émile Zola",
		"Test 123 !@#$",
	}

	for _, tc := range testCases {
		result := sanitizeUTF8(tc)
		if result != tc {
			t.Errorf("Expected valid UTF-8 string to remain unchanged: %q, got %q", tc, result)
		}
	}
}

// TestSanitizeUTF8_InvalidString tests that invalid UTF-8 sequences are replaced
func TestSanitizeUTF8_InvalidString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid byte at start",
			input:    "\xffHello World",
			expected: "�Hello World",
		},
		{
			name:     "invalid byte in middle",
			input:    "Hello\xffWorld",
			expected: "Hello�World",
		},
		{
			name:     "invalid byte at end",
			input:    "Hello World\xff",
			expected: "Hello World�",
		},
		{
			name:     "multiple invalid bytes",
			input:    "\xffHello\xfe\xfdWorld\xfc",
			expected: "�Hello�World�",
		},
		{
			name:     "incomplete UTF-8 sequence",
			input:    "Test\xc3",
			expected: "Test�",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizeUTF8(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestSanitizeUTF8Slice_ValidStrings tests that valid UTF-8 strings in slices pass through unchanged
func TestSanitizeUTF8Slice_ValidStrings(t *testing.T) {
	t.Parallel()
	input := []string{"DIMENSION", "LOL", "NTb"}
	result := sanitizeUTF8Slice(input)

	if len(result) != len(input) {
		t.Errorf("Expected slice length %d, got %d", len(input), len(result))
	}

	for i, s := range input {
		if result[i] != s {
			t.Errorf("Expected string at index %d to be %q, got %q", i, s, result[i])
		}
	}
}

// TestSanitizeUTF8Slice_InvalidStrings tests that invalid UTF-8 sequences in slices are sanitized
func TestSanitizeUTF8Slice_InvalidStrings(t *testing.T) {
	t.Parallel()
	input := []string{
		"Valid",
		"\xffInvalid",
		"Also\xfeValid",
	}
	expected := []string{
		"Valid",
		"�Invalid",
		"Also�Valid",
	}

	result := sanitizeUTF8Slice(input)

	if len(result) != len(expected) {
		t.Fatalf("Expected slice length %d, got %d", len(expected), len(result))
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Expected string at index %d to be %q, got %q", i, exp, result[i])
		}
	}
}

// TestSanitizeUTF8Slice_EmptySlice tests that empty slices are handled correctly
func TestSanitizeUTF8Slice_EmptySlice(t *testing.T) {
	t.Parallel()
	input := []string{}
	result := sanitizeUTF8Slice(input)

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(result))
	}
}

// TestConvertShowToProto_InvalidUTF8 tests that invalid UTF-8 in show fields is sanitized
func TestConvertShowToProto_InvalidUTF8(t *testing.T) {
	t.Parallel()
	show := models.Show{
		Name:     "Breaking\xffBad",
		ID:       42,
		Year:     2008,
		ImageURL: "http://example.com/image\xfe.jpg",
	}

	result := convertShowToProto(show)

	if result.Name != "Breaking�Bad" {
		t.Errorf("Expected sanitized name 'Breaking�Bad', got '%s'", result.Name)
	}
	if result.ImageUrl != "http://example.com/image�.jpg" {
		t.Errorf("Expected sanitized image URL 'http://example.com/image�.jpg', got '%s'", result.ImageUrl)
	}
}

// TestConvertSubtitleToProto_InvalidUTF8 tests that invalid UTF-8 in subtitle fields is sanitized
func TestConvertSubtitleToProto_InvalidUTF8(t *testing.T) {
	t.Parallel()
	subtitle := models.Subtitle{
		ID:            101,
		ShowID:        1,
		ShowName:      "Breaking\xffBad",
		Name:          "S01\xfeE01",
		Language:      "hun\xfd",
		Filename:      "file\xfc.srt",
		DownloadURL:   "http://example.com/\xfb",
		Uploader:      "user\xfa123",
		ReleaseGroups: []string{"DIM\xffENSION", "L\xfeOL"},
		Release:       "720p\xff",
	}

	result := convertSubtitleToProto(subtitle)

	if result.ShowName != "Breaking�Bad" {
		t.Errorf("Expected sanitized ShowName 'Breaking�Bad', got '%s'", result.ShowName)
	}
	if result.Name != "S01�E01" {
		t.Errorf("Expected sanitized Name 'S01�E01', got '%s'", result.Name)
	}
	if result.Language != "hun�" {
		t.Errorf("Expected sanitized Language 'hun�', got '%s'", result.Language)
	}
	if result.Filename != "file�.srt" {
		t.Errorf("Expected sanitized Filename 'file�.srt', got '%s'", result.Filename)
	}
	if result.DownloadUrl != "http://example.com/�" {
		t.Errorf("Expected sanitized DownloadUrl 'http://example.com/�', got '%s'", result.DownloadUrl)
	}
	if result.Uploader != "user�123" {
		t.Errorf("Expected sanitized Uploader 'user�123', got '%s'", result.Uploader)
	}
	if result.Release != "720p�" {
		t.Errorf("Expected sanitized Release '720p�', got '%s'", result.Release)
	}
	if len(result.ReleaseGroups) != 2 {
		t.Fatalf("Expected 2 release groups, got %d", len(result.ReleaseGroups))
	}
	if result.ReleaseGroups[0] != "DIM�ENSION" {
		t.Errorf("Expected sanitized release group 'DIM�ENSION', got '%s'", result.ReleaseGroups[0])
	}
	if result.ReleaseGroups[1] != "L�OL" {
		t.Errorf("Expected sanitized release group 'L�OL', got '%s'", result.ReleaseGroups[1])
	}
}

// TestConvertThirdPartyIdsToProto_InvalidUTF8 tests that invalid UTF-8 in IMDB ID is sanitized
func TestConvertThirdPartyIdsToProto_InvalidUTF8(t *testing.T) {
	t.Parallel()
	ids := models.ThirdPartyIds{
		IMDBID:   "tt09\xff03747",
		TVDBID:   81189,
		TVMazeID: 169,
		TraktID:  1388,
	}

	result := convertThirdPartyIdsToProto(ids)

	if result.ImdbId != "tt09�03747" {
		t.Errorf("Expected sanitized IMDB ID 'tt09�03747', got '%s'", result.ImdbId)
	}
}

// TestSafeInt32_OverflowValues tests that safeInt32 clamps values exceeding int32 bounds
func TestSafeInt32_OverflowValues(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    int
		expected int32
	}{
		{"positive overflow", math.MaxInt32 + 1, math.MaxInt32},
		{"large positive overflow", math.MaxInt32 + 1000, math.MaxInt32},
		{"negative overflow", math.MinInt32 - 1, math.MinInt32},
		{"large negative overflow", math.MinInt32 - 1000, math.MinInt32},
		{"max int32", math.MaxInt32, math.MaxInt32},
		{"min int32", math.MinInt32, math.MinInt32},
		{"zero", 0, 0},
		{"positive within range", 42, 42},
		{"negative within range", -42, -42},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := safeInt32(tc.input)
			if result != tc.expected {
				t.Errorf("safeInt32(%d) = %d, expected %d", tc.input, result, tc.expected)
			}
		})
	}
}
