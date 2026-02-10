package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// DownloadSubtitle downloads a subtitle file, with support for extracting specific episodes from season packs.
// The download URL is derived from the subtitle ID.
// If episode is nil, the entire file is returned without extraction.
func (c *client) DownloadSubtitle(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error) {
	downloadURL, err := c.buildDownloadURL(subtitleID)
	if err != nil {
		return nil, err
	}

	return c.subtitleDownloader.DownloadSubtitle(ctx, downloadURL, episode)
}

func (c *client) buildDownloadURL(subtitleID string) (string, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/index.php"
	query := baseURL.Query()
	query.Set("action", "letolt")
	query.Set("felirat", subtitleID)
	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}
