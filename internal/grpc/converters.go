package grpc

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Belphemur/SuperSubtitles/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// convertShowToProto converts a models.Show to a proto Show message
func convertShowToProto(show models.Show) *pb.Show {
	return &pb.Show{
		Name:     show.Name,
		Id:       int64(show.ID),
		Year:     int32(show.Year),
		ImageUrl: show.ImageURL,
	}
}

// convertShowFromProto converts a proto Show message to a models.Show
func convertShowFromProto(pbShow *pb.Show) models.Show {
	if pbShow == nil {
		return models.Show{}
	}
	return models.Show{
		Name:     pbShow.Name,
		ID:       int(pbShow.Id),
		Year:     int(pbShow.Year),
		ImageURL: pbShow.ImageUrl,
	}
}

// convertThirdPartyIdsToProto converts models.ThirdPartyIds to proto ThirdPartyIds message
func convertThirdPartyIdsToProto(ids models.ThirdPartyIds) *pb.ThirdPartyIds {
	return &pb.ThirdPartyIds{
		ImdbId:   ids.IMDBID,
		TvdbId:   int64(ids.TVDBID),
		TvMazeId: int64(ids.TVMazeID),
		TraktId:  int64(ids.TraktID),
	}
}

// convertQualityToProto converts a models.Quality to a proto Quality enum
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

// convertSubtitleToProto converts a models.Subtitle to a proto Subtitle message
func convertSubtitleToProto(subtitle models.Subtitle) *pb.Subtitle {
	qualities := make([]pb.Quality, len(subtitle.Qualities))
	for i, q := range subtitle.Qualities {
		qualities[i] = convertQualityToProto(q)
	}

	var uploadedAt *timestamppb.Timestamp
	// Only set timestamp if UploadedAt is not zero
	// This prevents serializing invalid dates (year 0001-01-01) to clients
	if !subtitle.UploadedAt.IsZero() {
		uploadedAt = timestamppb.New(subtitle.UploadedAt)
	}

	return &pb.Subtitle{
		Id:            int64(subtitle.ID),
		ShowId:        int64(subtitle.ShowID),
		ShowName:      subtitle.ShowName,
		Name:          subtitle.Name,
		Language:      subtitle.Language,
		Season:        int32(subtitle.Season),
		Episode:       int32(subtitle.Episode),
		Filename:      subtitle.Filename,
		DownloadUrl:   subtitle.DownloadURL,
		Uploader:      subtitle.Uploader,
		UploadedAt:    uploadedAt,
		Qualities:     qualities,
		ReleaseGroups: subtitle.ReleaseGroups,
		Release:       subtitle.Release,
		IsSeasonPack:  subtitle.IsSeasonPack,
	}
}

// convertSubtitleCollectionToProto converts a models.SubtitleCollection to a proto SubtitleCollection message
func convertSubtitleCollectionToProto(collection models.SubtitleCollection) *pb.SubtitleCollection {
	subtitles := make([]*pb.Subtitle, len(collection.Subtitles))
	for i, subtitle := range collection.Subtitles {
		subtitles[i] = convertSubtitleToProto(subtitle)
	}

	return &pb.SubtitleCollection{
		ShowName:  collection.ShowName,
		Subtitles: subtitles,
		Total:     int64(collection.Total),
	}
}

// convertShowSubtitlesToProto converts a models.ShowSubtitles to a proto ShowSubtitles message
func convertShowSubtitlesToProto(ss models.ShowSubtitles) *pb.ShowSubtitles {
	return &pb.ShowSubtitles{
		Show:               convertShowToProto(ss.Show),
		ThirdPartyIds:      convertThirdPartyIdsToProto(ss.ThirdPartyIds),
		SubtitleCollection: convertSubtitleCollectionToProto(ss.SubtitleCollection),
	}
}
