package services

import (
	"context"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// SubtitleDownloader defines the interface for downloading subtitles
type SubtitleDownloader interface {
	// DownloadSubtitle downloads a subtitle, optionally extracting a specific episode from a season pack
	DownloadSubtitle(ctx context.Context, downloadURL string, req models.DownloadRequest) (*models.DownloadResult, error)
}
