package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	pb "github.com/Belphemur/SuperSubtitles/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// mockClient implements client.Client for testing
type mockClient struct {
	getShowListFunc        func(ctx context.Context) ([]models.Show, error)
	getSubtitlesFunc       func(ctx context.Context, showID int) (*models.SubtitleCollection, error)
	getShowSubtitlesFunc   func(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error)
	checkForUpdatesFunc    func(ctx context.Context, contentID int64) (*models.UpdateCheckResult, error)
	downloadSubtitleFunc   func(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error)
	getRecentSubtitlesFunc func(ctx context.Context, sinceID int) ([]models.ShowSubtitles, error)

	streamShowListFunc        func(ctx context.Context) <-chan models.StreamResult[models.Show]
	streamSubtitlesFunc       func(ctx context.Context, showID int) <-chan models.StreamResult[models.Subtitle]
	streamShowSubtitlesFunc   func(ctx context.Context, shows []models.Show) <-chan models.StreamResult[models.ShowSubtitles]
	streamRecentSubtitlesFunc func(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles]
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

func (m *mockClient) CheckForUpdates(ctx context.Context, contentID int64) (*models.UpdateCheckResult, error) {
	if m.checkForUpdatesFunc != nil {
		return m.checkForUpdatesFunc(ctx, contentID)
	}
	return &models.UpdateCheckResult{}, nil
}

func (m *mockClient) DownloadSubtitle(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error) {
	if m.downloadSubtitleFunc != nil {
		return m.downloadSubtitleFunc(ctx, subtitleID, episode)
	}
	return &models.DownloadResult{}, nil
}

func (m *mockClient) GetRecentSubtitles(ctx context.Context, sinceID int) ([]models.ShowSubtitles, error) {
	if m.getRecentSubtitlesFunc != nil {
		return m.getRecentSubtitlesFunc(ctx, sinceID)
	}
	return []models.ShowSubtitles{}, nil
}

func (m *mockClient) StreamShowList(ctx context.Context) <-chan models.StreamResult[models.Show] {
	if m.streamShowListFunc != nil {
		return m.streamShowListFunc(ctx)
	}
	ch := make(chan models.StreamResult[models.Show])
	go func() {
		defer close(ch)
		shows, err := m.GetShowList(ctx)
		if err != nil {
			ch <- models.StreamResult[models.Show]{Err: err}
			return
		}
		for _, show := range shows {
			ch <- models.StreamResult[models.Show]{Value: show}
		}
	}()
	return ch
}

func (m *mockClient) StreamSubtitles(ctx context.Context, showID int) <-chan models.StreamResult[models.Subtitle] {
	if m.streamSubtitlesFunc != nil {
		return m.streamSubtitlesFunc(ctx, showID)
	}
	ch := make(chan models.StreamResult[models.Subtitle])
	go func() {
		defer close(ch)
		collection, err := m.GetSubtitles(ctx, showID)
		if err != nil {
			ch <- models.StreamResult[models.Subtitle]{Err: err}
			return
		}
		for _, subtitle := range collection.Subtitles {
			ch <- models.StreamResult[models.Subtitle]{Value: subtitle}
		}
	}()
	return ch
}

func (m *mockClient) StreamShowSubtitles(ctx context.Context, shows []models.Show) <-chan models.StreamResult[models.ShowSubtitles] {
	if m.streamShowSubtitlesFunc != nil {
		return m.streamShowSubtitlesFunc(ctx, shows)
	}
	ch := make(chan models.StreamResult[models.ShowSubtitles])
	go func() {
		defer close(ch)
		showSubtitles, err := m.GetShowSubtitles(ctx, shows)
		if err != nil {
			ch <- models.StreamResult[models.ShowSubtitles]{Err: err}
			return
		}
		for _, ss := range showSubtitles {
			ch <- models.StreamResult[models.ShowSubtitles]{Value: ss}
		}
	}()
	return ch
}

func (m *mockClient) StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles] {
	if m.streamRecentSubtitlesFunc != nil {
		return m.streamRecentSubtitlesFunc(ctx, sinceID)
	}
	ch := make(chan models.StreamResult[models.ShowSubtitles])
	go func() {
		defer close(ch)
		showSubtitles, err := m.GetRecentSubtitles(ctx, sinceID)
		if err != nil {
			ch <- models.StreamResult[models.ShowSubtitles]{Err: err}
			return
		}
		for _, ss := range showSubtitles {
			ch <- models.StreamResult[models.ShowSubtitles]{Value: ss}
		}
	}()
	return ch
}

// mockServerStream implements grpc.ServerStreamingServer for testing streaming RPCs
type mockServerStream[T any] struct {
	grpc.ServerStream
	ctx   context.Context
	items []*T
}

func newMockServerStream[T any]() *mockServerStream[T] {
	return &mockServerStream[T]{ctx: context.Background()}
}

func (m *mockServerStream[T]) Send(item *T) error {
	m.items = append(m.items, item)
	return nil
}

func (m *mockServerStream[T]) SetHeader(metadata.MD) error  { return nil }
func (m *mockServerStream[T]) SendHeader(metadata.MD) error { return nil }
func (m *mockServerStream[T]) SetTrailer(metadata.MD)       {}
func (m *mockServerStream[T]) Context() context.Context     { return m.ctx }
func (m *mockServerStream[T]) SendMsg(msg any) error        { return nil }
func (m *mockServerStream[T]) RecvMsg(msg any) error        { return nil }

// TestGetShowList_Success tests successful show list streaming
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

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.Show]()

	err := srv.GetShowList(&pb.GetShowListRequest{}, stream)
	if err != nil {
		t.Fatalf("GetShowList returned error: %v", err)
	}

	if len(stream.items) != 2 {
		t.Fatalf("Expected 2 shows streamed, got %d", len(stream.items))
	}

	if stream.items[0].Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", stream.items[0].Name)
	}
	if stream.items[0].Id != 1 {
		t.Errorf("Expected show ID 1, got %d", stream.items[0].Id)
	}
	if stream.items[1].Name != "Game of Thrones" {
		t.Errorf("Expected show name 'Game of Thrones', got '%s'", stream.items[1].Name)
	}
}

// TestGetShowList_Error tests error handling in show list streaming
func TestGetShowList_Error(t *testing.T) {
	mock := &mockClient{
		getShowListFunc: func(ctx context.Context) ([]models.Show, error) {
			return nil, errors.New("network error")
		},
	}

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.Show]()

	err := srv.GetShowList(&pb.GetShowListRequest{}, stream)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

// TestGetSubtitles_Success tests successful subtitle streaming
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

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.Subtitle]()

	err := srv.GetSubtitles(&pb.GetSubtitlesRequest{ShowId: 1}, stream)
	if err != nil {
		t.Fatalf("GetSubtitles returned error: %v", err)
	}

	if len(stream.items) != 1 {
		t.Fatalf("Expected 1 subtitle streamed, got %d", len(stream.items))
	}

	subtitle := stream.items[0]
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

// TestGetShowSubtitles_Success tests successful show subtitles streaming
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

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.ShowSubtitlesCollection]()

	req := &pb.GetShowSubtitlesRequest{
		Shows: []*pb.Show{
			{Name: "Breaking Bad", Id: 1, Year: 2008, ImageUrl: "http://example.com/image.jpg"},
		},
	}

	err := srv.GetShowSubtitles(req, stream)
	if err != nil {
		t.Fatalf("GetShowSubtitles returned error: %v", err)
	}

	// Expect 1 item: 1 ShowSubtitlesCollection containing ShowInfo + 1 Subtitle
	if len(stream.items) != 1 {
		t.Fatalf("Expected 1 streamed item, got %d", len(stream.items))
	}

	collection := stream.items[0]

	// Verify ShowInfo
	showInfo := collection.GetShowInfo()
	if showInfo == nil {
		t.Fatal("Expected collection to have ShowInfo")
	}
	if showInfo.Show.Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", showInfo.Show.Name)
	}
	if showInfo.ThirdPartyIds.ImdbId != "tt0903747" {
		t.Errorf("Expected IMDB ID 'tt0903747', got '%s'", showInfo.ThirdPartyIds.ImdbId)
	}
	if showInfo.ThirdPartyIds.TvdbId != 81189 {
		t.Errorf("Expected TVDB ID 81189, got %d", showInfo.ThirdPartyIds.TvdbId)
	}

	// Verify Subtitles
	if len(collection.Subtitles) != 1 {
		t.Fatalf("Expected 1 subtitle, got %d", len(collection.Subtitles))
	}
	if collection.Subtitles[0].Id != 101 {
		t.Errorf("Expected subtitle ID 101, got %d", collection.Subtitles[0].Id)
	}
	if collection.Subtitles[0].ShowId != 1 {
		t.Errorf("Expected show ID 1, got %d", collection.Subtitles[0].ShowId)
	}
}

// TestGetShowSubtitles_NoValidShows tests error when no valid shows are provided
func TestGetShowSubtitles_NoValidShows(t *testing.T) {
	srv := NewServer(&mockClient{}).(*server)
	stream := newMockServerStream[pb.ShowSubtitlesCollection]()

	req := &pb.GetShowSubtitlesRequest{
		Shows: []*pb.Show{nil},
	}

	err := srv.GetShowSubtitles(req, stream)
	if err == nil {
		t.Fatal("Expected error but got nil")
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
		checkForUpdatesFunc: func(ctx context.Context, contentID int64) (*models.UpdateCheckResult, error) {
			if contentID != 12345 {
				t.Errorf("Expected content ID 12345, got %d", contentID)
			}
			return mockResult, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	resp, err := srv.CheckForUpdates(ctx, &pb.CheckForUpdatesRequest{ContentId: 12345})
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
		downloadSubtitleFunc: func(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error) {
			if subtitleID != "101" {
				t.Errorf("Expected subtitle ID '101', got '%s'", subtitleID)
			}
			if episode == nil || *episode != 1 {
				t.Errorf("Expected episode 1, got %v", episode)
			}
			return mockResult, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	req := &pb.DownloadSubtitleRequest{
		SubtitleId: "101",
		Episode:    proto.Int32(1),
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

// TestDownloadSubtitle_NoEpisode tests subtitle download without specifying an episode
func TestDownloadSubtitle_NoEpisode(t *testing.T) {
	mockResult := &models.DownloadResult{
		Filename:    "breaking.bad.season.01.srt",
		Content:     []byte("season pack content"),
		ContentType: "application/zip",
	}

	mock := &mockClient{
		downloadSubtitleFunc: func(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error) {
			if subtitleID != "999" {
				t.Errorf("Expected subtitle ID '999', got '%s'", subtitleID)
			}
			if episode != nil {
				t.Errorf("Expected episode to be nil, got %v", episode)
			}
			return mockResult, nil
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	// Request without episode - Episode field is nil
	req := &pb.DownloadSubtitleRequest{
		SubtitleId: "999",
		Episode:    nil,
	}

	resp, err := srv.DownloadSubtitle(ctx, req)
	if err != nil {
		t.Fatalf("DownloadSubtitle returned error: %v", err)
	}

	if resp.Filename != "breaking.bad.season.01.srt" {
		t.Errorf("Expected filename 'breaking.bad.season.01.srt', got '%s'", resp.Filename)
	}
	if string(resp.Content) != "season pack content" {
		t.Errorf("Expected content 'season pack content', got '%s'", string(resp.Content))
	}
	if resp.ContentType != "application/zip" {
		t.Errorf("Expected content type 'application/zip', got '%s'", resp.ContentType)
	}
}

// TestGetRecentSubtitles_Success tests successful recent subtitles streaming
func TestGetRecentSubtitles_Success(t *testing.T) {
	mockShowSubtitles := []models.ShowSubtitles{
		{
			Show: models.Show{Name: "Breaking Bad", ID: 1, Year: 2008},
			ThirdPartyIds: models.ThirdPartyIds{
				IMDBID: "tt0903747",
				TVDBID: 81189,
			},
			SubtitleCollection: models.SubtitleCollection{
				ShowName: "Breaking Bad",
				Total:    2,
				Subtitles: []models.Subtitle{
					{ID: 101, ShowID: 1, ShowName: "Breaking Bad", Language: "hun"},
					{ID: 102, ShowID: 1, ShowName: "Breaking Bad", Language: "eng"},
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

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.ShowSubtitleItem]()

	err := srv.GetRecentSubtitles(&pb.GetRecentSubtitlesRequest{SinceId: 100}, stream)
	if err != nil {
		t.Fatalf("GetRecentSubtitles returned error: %v", err)
	}

	// Expect 3 items: 1 ShowInfo + 2 Subtitles
	if len(stream.items) != 3 {
		t.Fatalf("Expected 3 streamed items, got %d", len(stream.items))
	}

	// First item should be ShowInfo with show name and third-party IDs
	showInfoItem := stream.items[0].GetShowInfo()
	if showInfoItem == nil {
		t.Fatal("Expected first item to be ShowInfo")
	}
	if showInfoItem.Show.Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", showInfoItem.Show.Name)
	}
	if showInfoItem.Show.Id != 1 {
		t.Errorf("Expected show ID 1, got %d", showInfoItem.Show.Id)
	}
	if showInfoItem.ThirdPartyIds.ImdbId != "tt0903747" {
		t.Errorf("Expected IMDB ID 'tt0903747', got '%s'", showInfoItem.ThirdPartyIds.ImdbId)
	}
	if showInfoItem.ThirdPartyIds.TvdbId != 81189 {
		t.Errorf("Expected TVDB ID 81189, got %d", showInfoItem.ThirdPartyIds.TvdbId)
	}

	// Second item should be first Subtitle
	subtitleItem := stream.items[1].GetSubtitle()
	if subtitleItem == nil {
		t.Fatal("Expected second item to be Subtitle")
	}
	if subtitleItem.Id != 101 {
		t.Errorf("Expected subtitle ID 101, got %d", subtitleItem.Id)
	}
	if subtitleItem.ShowId != 1 {
		t.Errorf("Expected show ID 1, got %d", subtitleItem.ShowId)
	}

	// Third item should be second Subtitle
	subtitleItem2 := stream.items[2].GetSubtitle()
	if subtitleItem2 == nil {
		t.Fatal("Expected third item to be Subtitle")
	}
	if subtitleItem2.Id != 102 {
		t.Errorf("Expected subtitle ID 102, got %d", subtitleItem2.Id)
	}
	if subtitleItem2.Language != "eng" {
		t.Errorf("Expected language 'eng', got '%s'", subtitleItem2.Language)
	}
}
