package grpc

import (
	"context"

	pb "github.com/Belphemur/SuperSubtitles/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/internal/client"
	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	s.logger.Debug().Int64("show_id", req.ShowId).Msg("GetSubtitles called")

	collection, err := s.client.GetSubtitles(ctx, int(req.ShowId))
	if err != nil {
		s.logger.Error().Err(err).Int64("show_id", req.ShowId).Msg("Failed to get subtitles")
		return nil, status.Errorf(codes.Internal, "failed to get subtitles: %v", err)
	}

	s.logger.Debug().Int64("show_id", req.ShowId).Int("count", len(collection.Subtitles)).Msg("GetSubtitles completed")
	return &pb.GetSubtitlesResponse{
		SubtitleCollection: convertSubtitleCollectionToProto(*collection),
	}, nil
}

// GetShowSubtitles implements SuperSubtitlesServiceServer.GetShowSubtitles
func (s *server) GetShowSubtitles(ctx context.Context, req *pb.GetShowSubtitlesRequest) (*pb.GetShowSubtitlesResponse, error) {
	s.logger.Debug().Int("show_count", len(req.Shows)).Msg("GetShowSubtitles called")

	// Filter out nil entries and convert proto shows to models
	shows := make([]models.Show, 0, len(req.Shows))
	for _, pbShow := range req.Shows {
		if pbShow == nil {
			s.logger.Warn().Msg("Skipping nil show entry in request")
			continue
		}
		shows = append(shows, convertShowFromProto(pbShow))
	}

	if len(shows) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no valid shows provided")
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
		Str("subtitle_id", req.SubtitleId).
		Int32("episode", req.Episode).
		Msg("DownloadSubtitle called")

	result, err := s.client.DownloadSubtitle(ctx, req.SubtitleId, int(req.Episode))
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
	s.logger.Debug().Int64("since_id", req.SinceId).Msg("GetRecentSubtitles called")

	showSubtitles, err := s.client.GetRecentSubtitles(ctx, int(req.SinceId))
	if err != nil {
		s.logger.Error().Err(err).Int64("since_id", req.SinceId).Msg("Failed to get recent subtitles")
		return nil, status.Errorf(codes.Internal, "failed to get recent subtitles: %v", err)
	}

	pbShowSubtitles := make([]*pb.ShowSubtitles, len(showSubtitles))
	for i, ss := range showSubtitles {
		pbShowSubtitles[i] = convertShowSubtitlesToProto(ss)
	}

	s.logger.Debug().Int64("since_id", req.SinceId).Int("count", len(pbShowSubtitles)).Msg("GetRecentSubtitles completed")
	return &pb.GetRecentSubtitlesResponse{ShowSubtitles: pbShowSubtitles}, nil
}
