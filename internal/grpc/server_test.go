package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/Belphemur/SuperSubtitles/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// mockClient implements client.Client for testing
type mockClient struct {
	getShowListFunc        func(ctx context.Context) ([]models.Show, error)
	getSubtitlesFunc       func(ctx context.Context, showID int) (*models.SubtitleCollection, error)
	getShowSubtitlesFunc   func(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error)
	checkForUpdatesFunc    func(ctx context.Context, contentID string) (*models.UpdateCheckResult, error)
	downloadSubtitleFunc   func(ctx context.Context, downloadURL string, req models.DownloadRequest) (*models.DownloadResult, error)
	getRecentSubtitlesFunc func(ctx context.Context, sinceID int) ([]models.ShowSubtitles, error)
}

func (m *mockClient) GetShowList(ctx context.Context) ([]models.Show, error) {
	if m.getShowListFunc != nil {
		return m.getShowListFunc(ctx)
	}
	return []models.Show{}, nil
}

func (m *mockClient) GetSubtitles(ctx context.Context, showID int) (*models.SubtitleCollection, error) {
	if m.getSubtitlesFunc != nil {
		return m.getSubtitlesFunc(ctx, showID)
	}
	return &models.SubtitleCollection{}, nil
}

func (m *mockClient) GetShowSubtitles(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error) {
	if m.getShowSubtitlesFunc != nil {
		return m.getShowSubtitlesFunc(ctx, shows)
	}
	return []models.ShowSubtitles{}, nil
}

func (m *mockClient) CheckForUpdates(ctx context.Context, contentID string) (*models.UpdateCheckResult, error) {
	if m.checkForUpdatesFunc != nil {
		return m.checkForUpdatesFunc(ctx, contentID)
	}
	return &models.UpdateCheckResult{}, nil
}

func (m *mockClient) DownloadSubtitle(ctx context.Context, downloadURL string, req models.DownloadRequest) (*models.DownloadResult, error) {
	if m.downloadSubtitleFunc != nil {
		return m.downloadSubtitleFunc(ctx, downloadURL, req)
	}
	return &models.DownloadResult{}, nil
}

func (m *mockClient) GetRecentSubtitles(ctx context.Context, sinceID int) ([]models.ShowSubtitles, error) {
	if m.getRecentSubtitlesFunc != nil {
		return m.getRecentSubtitlesFunc(ctx, sinceID)
	}
	return []models.ShowSubtitles{}, nil
}

// TestGetShowList_Success tests successful show list retrieval
func TestGetShowList_Success(t *testing.T) {
	mockShows := []models.Show{
		{Name: "Breaking Bad", ID: 1, Year: 2008, ImageURL: "http://example.com/image1.jpg"},
		{Name: "Game of Thrones", ID: 2, Year: 2011, ImageURL: "http://example.com/image2.jpg"},
	}

	mock := &mockClient{
		getShowListFunc: func(ctx context.Context) ([]models.Show, error) {
			return mockShows, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	resp, err := srv.GetShowList(ctx, &pb.GetShowListRequest{})
	if err != nil {
		t.Fatalf("GetShowList returned error: %v", err)
	}

	if len(resp.Shows) != 2 {
		t.Fatalf("Expected 2 shows, got %d", len(resp.Shows))
	}

	if resp.Shows[0].Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", resp.Shows[0].Name)
	}
	if resp.Shows[0].Id != 1 {
		t.Errorf("Expected show ID 1, got %d", resp.Shows[0].Id)
	}
	if resp.Shows[1].Name != "Game of Thrones" {
		t.Errorf("Expected show name 'Game of Thrones', got '%s'", resp.Shows[1].Name)
	}
}

// TestGetShowList_Error tests error handling in show list retrieval
func TestGetShowList_Error(t *testing.T) {
	mock := &mockClient{
		getShowListFunc: func(ctx context.Context) ([]models.Show, error) {
			return nil, errors.New("network error")
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	_, err := srv.GetShowList(ctx, &pb.GetShowListRequest{})
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

// TestGetSubtitles_Success tests successful subtitle retrieval
func TestGetSubtitles_Success(t *testing.T) {
	uploadTime := time.Now()
	mockCollection := &models.SubtitleCollection{
		ShowName: "Breaking Bad",
		Total:    2,
		Subtitles: []models.Subtitle{
			{
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
				ReleaseGroups: []string{"DIMENSION"},
				Release:       "720p/1080p",
				IsSeasonPack:  false,
			},
		},
	}

	mock := &mockClient{
		getSubtitlesFunc: func(ctx context.Context, showID int) (*models.SubtitleCollection, error) {
			if showID != 1 {
				t.Errorf("Expected showID 1, got %d", showID)
			}
			return mockCollection, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	resp, err := srv.GetSubtitles(ctx, &pb.GetSubtitlesRequest{ShowId: 1})
	if err != nil {
		t.Fatalf("GetSubtitles returned error: %v", err)
	}

	if resp.SubtitleCollection.ShowName != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", resp.SubtitleCollection.ShowName)
	}
	if resp.SubtitleCollection.Total != 2 {
		t.Errorf("Expected total 2, got %d", resp.SubtitleCollection.Total)
	}
	if len(resp.SubtitleCollection.Subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(resp.SubtitleCollection.Subtitles))
	}

	subtitle := resp.SubtitleCollection.Subtitles[0]
	if subtitle.Id != 101 {
		t.Errorf("Expected subtitle ID 101, got %d", subtitle.Id)
	}
	if subtitle.Language != "hun" {
		t.Errorf("Expected language 'hun', got '%s'", subtitle.Language)
	}
	if len(subtitle.Qualities) != 2 {
		t.Errorf("Expected 2 qualities, got %d", len(subtitle.Qualities))
	}
	if subtitle.Qualities[0] != pb.Quality_QUALITY_720P {
		t.Errorf("Expected quality 720p, got %v", subtitle.Qualities[0])
	}
}

// TestGetShowSubtitles_Success tests successful show subtitles retrieval
func TestGetShowSubtitles_Success(t *testing.T) {
	mockShowSubtitles := []models.ShowSubtitles{
		{
			Show: models.Show{Name: "Breaking Bad", ID: 1, Year: 2008, ImageURL: "http://example.com/image.jpg"},
			ThirdPartyIds: models.ThirdPartyIds{
				IMDBID:   "tt0903747",
				TVDBID:   81189,
				TVMazeID: 169,
				TraktID:  1388,
			},
			SubtitleCollection: models.SubtitleCollection{
				ShowName: "Breaking Bad",
				Total:    1,
				Subtitles: []models.Subtitle{
					{
						ID:       101,
						ShowID:   1,
						ShowName: "Breaking Bad",
						Language: "hun",
					},
				},
			},
		},
	}

	mock := &mockClient{
		getShowSubtitlesFunc: func(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error) {
			if len(shows) != 1 {
				t.Errorf("Expected 1 show, got %d", len(shows))
			}
			return mockShowSubtitles, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	req := &pb.GetShowSubtitlesRequest{
		Shows: []*pb.Show{
			{Name: "Breaking Bad", Id: 1, Year: 2008, ImageUrl: "http://example.com/image.jpg"},
		},
	}

	resp, err := srv.GetShowSubtitles(ctx, req)
	if err != nil {
		t.Fatalf("GetShowSubtitles returned error: %v", err)
	}

	if len(resp.ShowSubtitles) != 1 {
		t.Fatalf("Expected 1 show subtitle, got %d", len(resp.ShowSubtitles))
	}

	ss := resp.ShowSubtitles[0]
	if ss.Show.Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", ss.Show.Name)
	}
	if ss.ThirdPartyIds.ImdbId != "tt0903747" {
		t.Errorf("Expected IMDB ID 'tt0903747', got '%s'", ss.ThirdPartyIds.ImdbId)
	}
	if ss.ThirdPartyIds.TvdbId != 81189 {
		t.Errorf("Expected TVDB ID 81189, got %d", ss.ThirdPartyIds.TvdbId)
	}
}

// TestCheckForUpdates_Success tests successful update check
func TestCheckForUpdates_Success(t *testing.T) {
	mockResult := &models.UpdateCheckResult{
		FilmCount:   5,
		SeriesCount: 10,
		HasUpdates:  true,
	}

	mock := &mockClient{
		checkForUpdatesFunc: func(ctx context.Context, contentID string) (*models.UpdateCheckResult, error) {
			if contentID != "12345" {
				t.Errorf("Expected content ID '12345', got '%s'", contentID)
			}
			return mockResult, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	resp, err := srv.CheckForUpdates(ctx, &pb.CheckForUpdatesRequest{ContentId: "12345"})
	if err != nil {
		t.Fatalf("CheckForUpdates returned error: %v", err)
	}

	if resp.FilmCount != 5 {
		t.Errorf("Expected film count 5, got %d", resp.FilmCount)
	}
	if resp.SeriesCount != 10 {
		t.Errorf("Expected series count 10, got %d", resp.SeriesCount)
	}
	if !resp.HasUpdates {
		t.Error("Expected HasUpdates to be true")
	}
}

// TestDownloadSubtitle_Success tests successful subtitle download
func TestDownloadSubtitle_Success(t *testing.T) {
	mockResult := &models.DownloadResult{
		Filename:    "breaking.bad.s01e01.srt",
		Content:     []byte("subtitle content"),
		ContentType: "application/x-subrip",
	}

	mock := &mockClient{
		downloadSubtitleFunc: func(ctx context.Context, downloadURL string, req models.DownloadRequest) (*models.DownloadResult, error) {
			if downloadURL != "http://example.com/download" {
				t.Errorf("Expected download URL 'http://example.com/download', got '%s'", downloadURL)
			}
			if req.SubtitleID != "101" {
				t.Errorf("Expected subtitle ID '101', got '%s'", req.SubtitleID)
			}
			if req.Episode != 1 {
				t.Errorf("Expected episode 1, got %d", req.Episode)
			}
			return mockResult, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	req := &pb.DownloadSubtitleRequest{
		DownloadUrl: "http://example.com/download",
		SubtitleId:  "101",
		Episode:     1,
	}

	resp, err := srv.DownloadSubtitle(ctx, req)
	if err != nil {
		t.Fatalf("DownloadSubtitle returned error: %v", err)
	}

	if resp.Filename != "breaking.bad.s01e01.srt" {
		t.Errorf("Expected filename 'breaking.bad.s01e01.srt', got '%s'", resp.Filename)
	}
	if string(resp.Content) != "subtitle content" {
		t.Errorf("Expected content 'subtitle content', got '%s'", string(resp.Content))
	}
	if resp.ContentType != "application/x-subrip" {
		t.Errorf("Expected content type 'application/x-subrip', got '%s'", resp.ContentType)
	}
}

// TestGetRecentSubtitles_Success tests successful recent subtitles retrieval
func TestGetRecentSubtitles_Success(t *testing.T) {
	mockShowSubtitles := []models.ShowSubtitles{
		{
			Show: models.Show{Name: "Breaking Bad", ID: 1, Year: 2008},
			SubtitleCollection: models.SubtitleCollection{
				ShowName: "Breaking Bad",
				Total:    1,
				Subtitles: []models.Subtitle{
					{ID: 101, ShowID: 1, Language: "hun"},
				},
			},
		},
	}

	mock := &mockClient{
		getRecentSubtitlesFunc: func(ctx context.Context, sinceID int) ([]models.ShowSubtitles, error) {
			if sinceID != 100 {
				t.Errorf("Expected since ID 100, got %d", sinceID)
			}
			return mockShowSubtitles, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	resp, err := srv.GetRecentSubtitles(ctx, &pb.GetRecentSubtitlesRequest{SinceId: 100})
	if err != nil {
		t.Fatalf("GetRecentSubtitles returned error: %v", err)
	}

	if len(resp.ShowSubtitles) != 1 {
		t.Fatalf("Expected 1 show subtitle, got %d", len(resp.ShowSubtitles))
	}

	ss := resp.ShowSubtitles[0]
	if ss.Show.Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", ss.Show.Name)
	}
}
