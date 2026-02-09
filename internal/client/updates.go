package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// CheckForUpdates checks if there are any updates available since a specific content ID
func (c *client) CheckForUpdates(ctx context.Context, contentID string) (*models.UpdateCheckResult, error) {
	logger := config.GetLogger()

	// Clean the content ID - remove "a_" prefix if present
	cleanContentID := contentID
	if strings.HasPrefix(contentID, "a_") {
		cleanContentID = strings.TrimPrefix(contentID, "a_")
	}

	logger.Info().Str("contentID", contentID).Str("cleanContentID", cleanContentID).Msg("Checking for updates since content ID")

	// Construct the URL for checking updates
	endpoint := fmt.Sprintf("%s/index.php?action=recheck&azon=%s", c.baseURL, cleanContentID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", config.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse JSON response
	var updateResponse models.UpdateCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	result := &models.UpdateCheckResult{
		FilmCount:   updateResponse.Film,
		SeriesCount: updateResponse.Sorozat,
		HasUpdates:  updateResponse.Film > 0 || updateResponse.Sorozat > 0,
	}

	logger.Info().
		Int("filmCount", result.FilmCount).
		Int("seriesCount", result.SeriesCount).
		Bool("hasUpdates", result.HasUpdates).
		Msg("Successfully checked for updates")

	return result, nil
}
