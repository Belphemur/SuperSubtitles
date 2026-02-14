package grpc

import (
	"context"
	"fmt"

	pb "github.com/Belphemur/SuperSubtitles/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/internal/client"
	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
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

// GetShowList streams all available TV shows
func (s *server) GetShowList(req *pb.GetShowListRequest, stream grpc.ServerStreamingServer[pb.Show]) error {
	s.logger.Debug().Msg("GetShowList called")

	shows, err := s.client.GetShowList(stream.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get show list")
		return status.Errorf(codes.Internal, "failed to get show list: %v", err)
	}

	for _, show := range shows {
		if err := stream.Send(convertShowToProto(show)); err != nil {
			return status.Errorf(codes.Internal, "failed to stream show: %v", err)
		}
	}

	s.logger.Debug().Int("count", len(shows)).Msg("GetShowList completed")
	return nil
}

// GetSubtitles streams all subtitles for a specific show
func (s *server) GetSubtitles(req *pb.GetSubtitlesRequest, stream grpc.ServerStreamingServer[pb.Subtitle]) error {
	s.logger.Debug().Int64("show_id", req.ShowId).Msg("GetSubtitles called")

	collection, err := s.client.GetSubtitles(stream.Context(), int(req.ShowId))
	if err != nil {
		s.logger.Error().Err(err).Int64("show_id", req.ShowId).Msg("Failed to get subtitles")
		return status.Errorf(codes.Internal, "failed to get subtitles: %v", err)
	}

	for _, subtitle := range collection.Subtitles {
		if err := stream.Send(convertSubtitleToProto(subtitle)); err != nil {
			return status.Errorf(codes.Internal, "failed to stream subtitle: %v", err)
		}
	}

	s.logger.Debug().Int64("show_id", req.ShowId).Int("count", len(collection.Subtitles)).Msg("GetSubtitles completed")
	return nil
}

// GetShowSubtitles streams show information and subtitles for multiple shows
func (s *server) GetShowSubtitles(req *pb.GetShowSubtitlesRequest, stream grpc.ServerStreamingServer[pb.ShowSubtitleItem]) error {
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
		return status.Error(codes.InvalidArgument, "no valid shows provided")
	}

	showSubtitles, err := s.client.GetShowSubtitles(stream.Context(), shows)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get show subtitles")
		return status.Errorf(codes.Internal, "failed to get show subtitles: %v", err)
	}

	if err := streamShowSubtitleItems(showSubtitles, stream); err != nil {
		return err
	}

	s.logger.Debug().Int("count", len(showSubtitles)).Msg("GetShowSubtitles completed")
	return nil
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
	logEvent := s.logger.Debug().
		Str("subtitle_id", req.SubtitleId)
	if req.Episode != nil {
		logEvent = logEvent.Int32("episode", *req.Episode)
	}
	logEvent.Msg("DownloadSubtitle called")

	// Convert optional proto int32 to optional Go int
	var episode *int
	if req.Episode != nil {
		e := int(*req.Episode)
		episode = &e
	}

	result, err := s.client.DownloadSubtitle(ctx, req.SubtitleId, episode)
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

// GetRecentSubtitles streams recently uploaded subtitles grouped by show
func (s *server) GetRecentSubtitles(req *pb.GetRecentSubtitlesRequest, stream grpc.ServerStreamingServer[pb.ShowSubtitleItem]) error {
	s.logger.Debug().Int64("since_id", req.SinceId).Msg("GetRecentSubtitles called")

	showSubtitles, err := s.client.GetRecentSubtitles(stream.Context(), int(req.SinceId))
	if err != nil {
		s.logger.Error().Err(err).Int64("since_id", req.SinceId).Msg("Failed to get recent subtitles")
		return status.Errorf(codes.Internal, "failed to get recent subtitles: %v", err)
	}

	if err := streamShowSubtitleItems(showSubtitles, stream); err != nil {
		return err
	}

	s.logger.Debug().Int64("since_id", req.SinceId).Int("count", len(showSubtitles)).Msg("GetRecentSubtitles completed")
	return nil
}

// streamShowSubtitleItems streams ShowSubtitles data as ShowSubtitleItem messages.
// For each show, it first sends a ShowInfo item, then streams each subtitle individually.
func streamShowSubtitleItems(showSubtitles []models.ShowSubtitles, stream grpc.ServerStreamingServer[pb.ShowSubtitleItem]) error {
	for _, ss := range showSubtitles {
		// Send show info first
		showInfoItem := &pb.ShowSubtitleItem{
			Item: &pb.ShowSubtitleItem_ShowInfo{
				ShowInfo: &pb.ShowInfo{
					Show:          convertShowToProto(ss.Show),
					ThirdPartyIds: convertThirdPartyIdsToProto(ss.ThirdPartyIds),
				},
			},
		}
		if err := stream.Send(showInfoItem); err != nil {
			return status.Errorf(codes.Internal, "failed to stream show info: %v", err)
		}

		// Then stream each subtitle
		for _, subtitle := range ss.SubtitleCollection.Subtitles {
			subtitleItem := &pb.ShowSubtitleItem{
				Item: &pb.ShowSubtitleItem_Subtitle{
					Subtitle: convertSubtitleToProto(subtitle),
				},
			}
			if err := stream.Send(subtitleItem); err != nil {
				return status.Errorf(codes.Internal, fmt.Sprintf("failed to stream subtitle for show %d: %%v", ss.Show.ID), err)
			}
		}
	}
	return nil
}
