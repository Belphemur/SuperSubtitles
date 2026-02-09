package client

import (
	"context"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// DownloadSubtitle downloads a subtitle file, with support for extracting specific episodes from season packs
func (c *client) DownloadSubtitle(ctx context.Context, downloadURL string, req models.DownloadRequest) (*models.DownloadResult, error) {
	return c.subtitleDownloader.DownloadSubtitle(ctx, downloadURL, req)
}
