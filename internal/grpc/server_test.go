package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "github.com/Belphemur/SuperSubtitles/v2/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
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

func (m *mockClient) Close() error {
	return nil
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestDownloadSubtitle_EpisodeNotFoundInZip tests that ErrSubtitleNotFoundInZip results in a NotFound gRPC status
func TestDownloadSubtitle_EpisodeNotFoundInZip(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		downloadSubtitleFunc: func(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error) {
			return nil, fmt.Errorf("failed to extract episode %d from ZIP: %w", *episode, &apperrors.ErrSubtitleNotFoundInZip{Episode: *episode, FileCount: 3})
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	req := &pb.DownloadSubtitleRequest{
		SubtitleId: "101",
		Episode:    proto.Int32(5),
	}

	_, err := srv.DownloadSubtitle(ctx, req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("Expected codes.NotFound, got %v", st.Code())
	}
}

// TestDownloadSubtitle_ResourceNotFound tests that ErrSubtitleResourceNotFound (HTTP 404) results in a NotFound gRPC status
func TestDownloadSubtitle_ResourceNotFound(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		downloadSubtitleFunc: func(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error) {
			return nil, fmt.Errorf("failed to download subtitle: %w", &apperrors.ErrSubtitleResourceNotFound{URL: "http://example.com/download/101"})
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	req := &pb.DownloadSubtitleRequest{SubtitleId: "101"}

	_, err := srv.DownloadSubtitle(ctx, req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("Expected codes.NotFound, got %v", st.Code())
	}
}

// TestGetSubtitles_ShowNotFound tests that ErrNotFound results in a NotFound gRPC status
func TestGetSubtitles_ShowNotFound(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		streamSubtitlesFunc: func(ctx context.Context, showID int) <-chan models.StreamResult[models.Subtitle] {
			ch := make(chan models.StreamResult[models.Subtitle], 1)
			ch <- models.StreamResult[models.Subtitle]{Err: apperrors.NewNotFoundError("show", showID)}
			close(ch)
			return ch
		},
	}

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.Subtitle]()

	err := srv.GetSubtitles(&pb.GetSubtitlesRequest{ShowId: 999}, stream)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("Expected codes.NotFound, got %v", st.Code())
	}
}
func TestGetRecentSubtitles_Success(t *testing.T) {
	t.Parallel()
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
	stream := newMockServerStream[pb.ShowSubtitlesCollection]()

	err := srv.GetRecentSubtitles(&pb.GetRecentSubtitlesRequest{SinceId: 100}, stream)
	if err != nil {
		t.Fatalf("GetRecentSubtitles returned error: %v", err)
	}

	// Expect 1 item: 1 ShowSubtitlesCollection containing ShowInfo + 2 Subtitles
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
	if showInfo.Show.Id != 1 {
		t.Errorf("Expected show ID 1, got %d", showInfo.Show.Id)
	}
	if showInfo.ThirdPartyIds.ImdbId != "tt0903747" {
		t.Errorf("Expected IMDB ID 'tt0903747', got '%s'", showInfo.ThirdPartyIds.ImdbId)
	}
	if showInfo.ThirdPartyIds.TvdbId != 81189 {
		t.Errorf("Expected TVDB ID 81189, got %d", showInfo.ThirdPartyIds.TvdbId)
	}

	// Verify Subtitles
	if len(collection.Subtitles) != 2 {
		t.Fatalf("Expected 2 subtitles, got %d", len(collection.Subtitles))
	}
	if collection.Subtitles[0].Id != 101 {
		t.Errorf("Expected first subtitle ID 101, got %d", collection.Subtitles[0].Id)
	}
	if collection.Subtitles[0].ShowId != 1 {
		t.Errorf("Expected show ID 1, got %d", collection.Subtitles[0].ShowId)
	}
	if collection.Subtitles[1].Id != 102 {
		t.Errorf("Expected second subtitle ID 102, got %d", collection.Subtitles[1].Id)
	}
	if collection.Subtitles[1].Language != "eng" {
		t.Errorf("Expected language 'eng', got '%s'", collection.Subtitles[1].Language)
	}
}

// errorOnSendStream is a mock stream that always returns an error on Send
type errorOnSendStream[T any] struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *errorOnSendStream[T]) Send(item *T) error {
	return fmt.Errorf("send failed")
}
func (m *errorOnSendStream[T]) SetHeader(metadata.MD) error  { return nil }
func (m *errorOnSendStream[T]) SendHeader(metadata.MD) error { return nil }
func (m *errorOnSendStream[T]) SetTrailer(metadata.MD)       {}
func (m *errorOnSendStream[T]) Context() context.Context     { return m.ctx }
func (m *errorOnSendStream[T]) SendMsg(msg any) error        { return nil }
func (m *errorOnSendStream[T]) RecvMsg(msg any) error        { return nil }

// TestGetShowList_StreamSendError tests that a stream.Send error returns Internal status
func TestGetShowList_StreamSendError(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		getShowListFunc: func(ctx context.Context) ([]models.Show, error) {
			return []models.Show{{Name: "Breaking Bad", ID: 1}}, nil
		},
	}

	srv := NewServer(mock).(*server)
	stream := &errorOnSendStream[pb.Show]{ctx: context.Background()}

	err := srv.GetShowList(&pb.GetShowListRequest{}, stream)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}

// TestGetShowList_PartialSuccess tests that errors after successful sends are logged and streaming continues
func TestGetShowList_PartialSuccess(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		streamShowListFunc: func(ctx context.Context) <-chan models.StreamResult[models.Show] {
			ch := make(chan models.StreamResult[models.Show], 2)
			ch <- models.StreamResult[models.Show]{Value: models.Show{Name: "Breaking Bad", ID: 1}}
			ch <- models.StreamResult[models.Show]{Err: errors.New("page 2 failed")}
			close(ch)
			return ch
		},
	}

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.Show]()

	err := srv.GetShowList(&pb.GetShowListRequest{}, stream)
	if err != nil {
		t.Fatalf("Expected no error (partial success), got: %v", err)
	}

	if len(stream.items) != 1 {
		t.Fatalf("Expected 1 show streamed, got %d", len(stream.items))
	}
	if stream.items[0].Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", stream.items[0].Name)
	}
}

// TestGetSubtitles_GenericError tests that a non-NotFound error returns Internal status
func TestGetSubtitles_GenericError(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		streamSubtitlesFunc: func(ctx context.Context, showID int) <-chan models.StreamResult[models.Subtitle] {
			ch := make(chan models.StreamResult[models.Subtitle], 1)
			ch <- models.StreamResult[models.Subtitle]{Err: errors.New("database error")}
			close(ch)
			return ch
		},
	}

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.Subtitle]()

	err := srv.GetSubtitles(&pb.GetSubtitlesRequest{ShowId: 1}, stream)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}

// TestGetShowSubtitles_ErrorAfterPartialSuccess tests that errors after partial sends are logged and streaming continues
func TestGetShowSubtitles_ErrorAfterPartialSuccess(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		streamShowSubtitlesFunc: func(ctx context.Context, shows []models.Show) <-chan models.StreamResult[models.ShowSubtitles] {
			ch := make(chan models.StreamResult[models.ShowSubtitles], 2)
			ch <- models.StreamResult[models.ShowSubtitles]{
				Value: models.ShowSubtitles{
					Show: models.Show{Name: "Breaking Bad", ID: 1},
					SubtitleCollection: models.SubtitleCollection{
						ShowName:  "Breaking Bad",
						Subtitles: []models.Subtitle{{ID: 101, ShowID: 1}},
					},
				},
			}
			ch <- models.StreamResult[models.ShowSubtitles]{Err: errors.New("fetch failed for show 2")}
			close(ch)
			return ch
		},
	}

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.ShowSubtitlesCollection]()

	req := &pb.GetShowSubtitlesRequest{
		Shows: []*pb.Show{
			{Name: "Breaking Bad", Id: 1},
			{Name: "Game of Thrones", Id: 2},
		},
	}

	err := srv.GetShowSubtitles(req, stream)
	if err != nil {
		t.Fatalf("Expected no error (partial success), got: %v", err)
	}

	if len(stream.items) != 1 {
		t.Fatalf("Expected 1 streamed item, got %d", len(stream.items))
	}
	if stream.items[0].GetShowInfo().Show.Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", stream.items[0].GetShowInfo().Show.Name)
	}
}

// TestGetShowSubtitles_StreamSendError tests that a stream.Send error returns Internal status
func TestGetShowSubtitles_StreamSendError(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		getShowSubtitlesFunc: func(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error) {
			return []models.ShowSubtitles{
				{
					Show:               models.Show{Name: "Breaking Bad", ID: 1},
					SubtitleCollection: models.SubtitleCollection{ShowName: "Breaking Bad"},
				},
			}, nil
		},
	}

	srv := NewServer(mock).(*server)
	stream := &errorOnSendStream[pb.ShowSubtitlesCollection]{ctx: context.Background()}

	req := &pb.GetShowSubtitlesRequest{
		Shows: []*pb.Show{{Name: "Breaking Bad", Id: 1}},
	}

	err := srv.GetShowSubtitles(req, stream)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}

// TestCheckForUpdates_Error tests that an error returns Internal status
func TestCheckForUpdates_Error(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		checkForUpdatesFunc: func(ctx context.Context, contentID int64) (*models.UpdateCheckResult, error) {
			return nil, errors.New("service unavailable")
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	_, err := srv.CheckForUpdates(ctx, &pb.CheckForUpdatesRequest{ContentId: 12345})
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}

// TestGetRecentSubtitles_ErrorAsFirstResult tests that an error as the first result returns Internal status
func TestGetRecentSubtitles_ErrorAsFirstResult(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		streamRecentSubtitlesFunc: func(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles] {
			ch := make(chan models.StreamResult[models.ShowSubtitles], 1)
			ch <- models.StreamResult[models.ShowSubtitles]{Err: errors.New("connection refused")}
			close(ch)
			return ch
		},
	}

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.ShowSubtitlesCollection]()

	err := srv.GetRecentSubtitles(&pb.GetRecentSubtitlesRequest{SinceId: 100}, stream)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}

// TestGetRecentSubtitles_StreamSendError tests that a stream.Send error returns Internal status
func TestGetRecentSubtitles_StreamSendError(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		getRecentSubtitlesFunc: func(ctx context.Context, sinceID int) ([]models.ShowSubtitles, error) {
			return []models.ShowSubtitles{
				{
					Show:               models.Show{Name: "Breaking Bad", ID: 1},
					SubtitleCollection: models.SubtitleCollection{ShowName: "Breaking Bad"},
				},
			}, nil
		},
	}

	srv := NewServer(mock).(*server)
	stream := &errorOnSendStream[pb.ShowSubtitlesCollection]{ctx: context.Background()}

	err := srv.GetRecentSubtitles(&pb.GetRecentSubtitlesRequest{SinceId: 100}, stream)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}

// TestGetRecentSubtitles_ErrorAfterPartialSuccess tests that errors after partial sends are logged and streaming continues
func TestGetRecentSubtitles_ErrorAfterPartialSuccess(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		streamRecentSubtitlesFunc: func(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles] {
			ch := make(chan models.StreamResult[models.ShowSubtitles], 2)
			ch <- models.StreamResult[models.ShowSubtitles]{
				Value: models.ShowSubtitles{
					Show: models.Show{Name: "Breaking Bad", ID: 1},
					SubtitleCollection: models.SubtitleCollection{
						ShowName:  "Breaking Bad",
						Subtitles: []models.Subtitle{{ID: 101, ShowID: 1}},
					},
				},
			}
			ch <- models.StreamResult[models.ShowSubtitles]{Err: errors.New("page 2 failed")}
			close(ch)
			return ch
		},
	}

	srv := NewServer(mock).(*server)
	stream := newMockServerStream[pb.ShowSubtitlesCollection]()

	err := srv.GetRecentSubtitles(&pb.GetRecentSubtitlesRequest{SinceId: 100}, stream)
	if err != nil {
		t.Fatalf("Expected no error (partial success), got: %v", err)
	}

	if len(stream.items) != 1 {
		t.Fatalf("Expected 1 streamed item, got %d", len(stream.items))
	}
	if stream.items[0].GetShowInfo().Show.Name != "Breaking Bad" {
		t.Errorf("Expected show name 'Breaking Bad', got '%s'", stream.items[0].GetShowInfo().Show.Name)
	}
}

// TestDownloadSubtitle_GenericError tests that a non-specific error returns Internal status
func TestDownloadSubtitle_GenericError(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		downloadSubtitleFunc: func(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error) {
			return nil, errors.New("unexpected server error")
		},
	}

	srv := NewServer(mock)
	ctx := context.Background()

	req := &pb.DownloadSubtitleRequest{SubtitleId: "101"}

	_, err := srv.DownloadSubtitle(ctx, req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", st.Code())
	}
}
