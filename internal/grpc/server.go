package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Belphemur/SuperSubtitles/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/internal/client"
	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
	"github.com/rs/zerolog"
)

// server implements the SuperSubtitlesServiceServer interface
type server struct {
	pb.UnimplementedSuperSubtitlesServiceServer
	client client.Client
	logger zerolog.Logger
}

// NewServer creates a new gRPC server instance
func NewServer(c client.Client) pb.SuperSubtitlesServiceServer {
	return &server{
		client: c,
		logger: config.GetLogger(),
	}
}

// GetShowList implements SuperSubtitlesServiceServer.GetShowList
func (s *server) GetShowList(ctx context.Context, req *pb.GetShowListRequest) (*pb.GetShowListResponse, error) {
	s.logger.Debug().Msg("GetShowList called")

	shows, err := s.client.GetShowList(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get show list")
		return nil, status.Errorf(codes.Internal, "failed to get show list: %v", err)
	}

	pbShows := make([]*pb.Show, len(shows))
	for i, show := range shows {
		pbShows[i] = convertShowToProto(show)
	}

	s.logger.Debug().Int("count", len(pbShows)).Msg("GetShowList completed")
	return &pb.GetShowListResponse{Shows: pbShows}, nil
}

// GetSubtitles implements SuperSubtitlesServiceServer.GetSubtitles
func (s *server) GetSubtitles(ctx context.Context, req *pb.GetSubtitlesRequest) (*pb.GetSubtitlesResponse, error) {
	s.logger.Debug().Int32("show_id", req.ShowId).Msg("GetSubtitles called")

	collection, err := s.client.GetSubtitles(ctx, int(req.ShowId))
	if err != nil {
		s.logger.Error().Err(err).Int32("show_id", req.ShowId).Msg("Failed to get subtitles")
		return nil, status.Errorf(codes.Internal, "failed to get subtitles: %v", err)
	}

	s.logger.Debug().Int32("show_id", req.ShowId).Int("count", len(collection.Subtitles)).Msg("GetSubtitles completed")
	return &pb.GetSubtitlesResponse{
		SubtitleCollection: convertSubtitleCollectionToProto(*collection),
	}, nil
}

// GetShowSubtitles implements SuperSubtitlesServiceServer.GetShowSubtitles
func (s *server) GetShowSubtitles(ctx context.Context, req *pb.GetShowSubtitlesRequest) (*pb.GetShowSubtitlesResponse, error) {
	s.logger.Debug().Int("show_count", len(req.Shows)).Msg("GetShowSubtitles called")

	shows := make([]models.Show, len(req.Shows))
	for i, pbShow := range req.Shows {
		shows[i] = convertShowFromProto(pbShow)
	}

	showSubtitles, err := s.client.GetShowSubtitles(ctx, shows)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get show subtitles")
		return nil, status.Errorf(codes.Internal, "failed to get show subtitles: %v", err)
	}

	pbShowSubtitles := make([]*pb.ShowSubtitles, len(showSubtitles))
	for i, ss := range showSubtitles {
		pbShowSubtitles[i] = convertShowSubtitlesToProto(ss)
	}

	s.logger.Debug().Int("count", len(pbShowSubtitles)).Msg("GetShowSubtitles completed")
	return &pb.GetShowSubtitlesResponse{ShowSubtitles: pbShowSubtitles}, nil
}

// CheckForUpdates implements SuperSubtitlesServiceServer.CheckForUpdates
func (s *server) CheckForUpdates(ctx context.Context, req *pb.CheckForUpdatesRequest) (*pb.CheckForUpdatesResponse, error) {
	s.logger.Debug().Str("content_id", req.ContentId).Msg("CheckForUpdates called")

	result, err := s.client.CheckForUpdates(ctx, req.ContentId)
	if err != nil {
		s.logger.Error().Err(err).Str("content_id", req.ContentId).Msg("Failed to check for updates")
		return nil, status.Errorf(codes.Internal, "failed to check for updates: %v", err)
	}

	s.logger.Debug().
		Str("content_id", req.ContentId).
		Int("film_count", result.FilmCount).
		Int("series_count", result.SeriesCount).
		Bool("has_updates", result.HasUpdates).
		Msg("CheckForUpdates completed")

	return &pb.CheckForUpdatesResponse{
		FilmCount:   int32(result.FilmCount),
		SeriesCount: int32(result.SeriesCount),
		HasUpdates:  result.HasUpdates,
	}, nil
}

// DownloadSubtitle implements SuperSubtitlesServiceServer.DownloadSubtitle
func (s *server) DownloadSubtitle(ctx context.Context, req *pb.DownloadSubtitleRequest) (*pb.DownloadSubtitleResponse, error) {
	s.logger.Debug().
		Str("download_url", req.DownloadUrl).
		Str("subtitle_id", req.SubtitleId).
		Int32("episode", req.Episode).
		Msg("DownloadSubtitle called")

	downloadReq := models.DownloadRequest{
		SubtitleID: req.SubtitleId,
		Episode:    int(req.Episode),
	}

	result, err := s.client.DownloadSubtitle(ctx, req.DownloadUrl, downloadReq)
	if err != nil {
		s.logger.Error().Err(err).Str("subtitle_id", req.SubtitleId).Msg("Failed to download subtitle")
		return nil, status.Errorf(codes.Internal, "failed to download subtitle: %v", err)
	}

	s.logger.Debug().
		Str("subtitle_id", req.SubtitleId).
		Str("filename", result.Filename).
		Int("size", len(result.Content)).
		Msg("DownloadSubtitle completed")

	return &pb.DownloadSubtitleResponse{
		Filename:    result.Filename,
		Content:     result.Content,
		ContentType: result.ContentType,
	}, nil
}

// GetRecentSubtitles implements SuperSubtitlesServiceServer.GetRecentSubtitles
func (s *server) GetRecentSubtitles(ctx context.Context, req *pb.GetRecentSubtitlesRequest) (*pb.GetRecentSubtitlesResponse, error) {
	s.logger.Debug().Int32("since_id", req.SinceId).Msg("GetRecentSubtitles called")

	showSubtitles, err := s.client.GetRecentSubtitles(ctx, int(req.SinceId))
	if err != nil {
		s.logger.Error().Err(err).Int32("since_id", req.SinceId).Msg("Failed to get recent subtitles")
		return nil, status.Errorf(codes.Internal, "failed to get recent subtitles: %v", err)
	}

	pbShowSubtitles := make([]*pb.ShowSubtitles, len(showSubtitles))
	for i, ss := range showSubtitles {
		pbShowSubtitles[i] = convertShowSubtitlesToProto(ss)
	}

	s.logger.Debug().Int32("since_id", req.SinceId).Int("count", len(pbShowSubtitles)).Msg("GetRecentSubtitles completed")
	return &pb.GetRecentSubtitlesResponse{ShowSubtitles: pbShowSubtitles}, nil
}

// Conversion functions

func convertShowToProto(show models.Show) *pb.Show {
	return &pb.Show{
		Name:     show.Name,
		Id:       int32(show.ID),
		Year:     int32(show.Year),
		ImageUrl: show.ImageURL,
	}
}

func convertShowFromProto(pbShow *pb.Show) models.Show {
	return models.Show{
		Name:     pbShow.Name,
		ID:       int(pbShow.Id),
		Year:     int(pbShow.Year),
		ImageURL: pbShow.ImageUrl,
	}
}

func convertThirdPartyIdsToProto(ids models.ThirdPartyIds) *pb.ThirdPartyIds {
	return &pb.ThirdPartyIds{
		ImdbId:   ids.IMDBID,
		TvdbId:   int32(ids.TVDBID),
		TvMazeId: int32(ids.TVMazeID),
		TraktId:  int32(ids.TraktID),
	}
}

func convertQualityToProto(quality models.Quality) pb.Quality {
	switch quality {
	case models.Quality360p:
		return pb.Quality_QUALITY_360P
	case models.Quality480p:
		return pb.Quality_QUALITY_480P
	case models.Quality720p:
		return pb.Quality_QUALITY_720P
	case models.Quality1080p:
		return pb.Quality_QUALITY_1080P
	case models.Quality2160p:
		return pb.Quality_QUALITY_2160P
	default:
		return pb.Quality_QUALITY_UNSPECIFIED
	}
}

func convertSubtitleToProto(subtitle models.Subtitle) *pb.Subtitle {
	qualities := make([]pb.Quality, len(subtitle.Qualities))
	for i, q := range subtitle.Qualities {
		qualities[i] = convertQualityToProto(q)
	}

	return &pb.Subtitle{
		Id:            int32(subtitle.ID),
		ShowId:        int32(subtitle.ShowID),
		ShowName:      subtitle.ShowName,
		Name:          subtitle.Name,
		Language:      subtitle.Language,
		Season:        int32(subtitle.Season),
		Episode:       int32(subtitle.Episode),
		Filename:      subtitle.Filename,
		DownloadUrl:   subtitle.DownloadURL,
		Uploader:      subtitle.Uploader,
		UploadedAt:    timestamppb.New(subtitle.UploadedAt),
		Qualities:     qualities,
		ReleaseGroups: subtitle.ReleaseGroups,
		Release:       subtitle.Release,
		IsSeasonPack:  subtitle.IsSeasonPack,
	}
}

func convertSubtitleCollectionToProto(collection models.SubtitleCollection) *pb.SubtitleCollection {
	subtitles := make([]*pb.Subtitle, len(collection.Subtitles))
	for i, subtitle := range collection.Subtitles {
		subtitles[i] = convertSubtitleToProto(subtitle)
	}

	return &pb.SubtitleCollection{
		ShowName:  collection.ShowName,
		Subtitles: subtitles,
		Total:     int32(collection.Total),
	}
}

func convertShowSubtitlesToProto(ss models.ShowSubtitles) *pb.ShowSubtitles {
	return &pb.ShowSubtitles{
		Show:               convertShowToProto(ss.Show),
		ThirdPartyIds:      convertThirdPartyIdsToProto(ss.ThirdPartyIds),
		SubtitleCollection: convertSubtitleCollectionToProto(ss.SubtitleCollection),
	}
}
