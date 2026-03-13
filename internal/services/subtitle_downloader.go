package services

import (
	"context"

	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
)

// SubtitleDownloader defines the interface for downloading subtitles
type SubtitleDownloader interface {
	// DownloadSubtitle downloads a subtitle, optionally extracting a specific episode from a season pack.
	// If episode is nil, whole-archive downloads may be normalized before returning.
	// Returns apperrors.ErrSubtitleNotFoundInZip if the requested episode is not found in a season-pack archive.
	// Returns apperrors.ErrSubtitleResourceNotFound if the subtitle URL returns HTTP 404.
	DownloadSubtitle(ctx context.Context, downloadURL string, episode *int) (*models.DownloadResult, error)

	// Close releases any resources held by the downloader (e.g., cache connections).
	Close() error
}
