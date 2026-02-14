package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// GetRecentSubtitles fetches recent subtitles from the main show page, filtered by subtitle ID
// Returns only subtitles with ID greater than sinceID
func (c *client) GetRecentSubtitles(ctx context.Context, sinceID int) ([]models.Subtitle, error) {
	var subtitles []models.Subtitle
	for result := range c.StreamRecentSubtitles(ctx, sinceID) {
		if result.Err != nil {
			return nil, result.Err
		}
		subtitles = append(subtitles, result.Value)
	}
	return subtitles, nil
}

// StreamRecentSubtitles streams recently uploaded subtitles as they are parsed.
// Fetches the main page, filters by sinceID, and streams each subtitle individually.
// Each Subtitle contains ShowID and ShowName for client-side grouping.
func (c *client) StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan StreamResult[models.Subtitle] {
	ch := make(chan StreamResult[models.Subtitle])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Int("sinceID", sinceID).Msg("Streaming recent subtitles from main page")

		// Fetch the main show page
		endpoint := fmt.Sprintf("%s/index.php?tab=sorozat", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			sendResult(ctx, ch, StreamResult[models.Subtitle]{Err: fmt.Errorf("failed to create request: %w", err)})
			return
		}
		req.Header.Set("User-Agent", config.GetUserAgent())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			sendResult(ctx, ch, StreamResult[models.Subtitle]{Err: fmt.Errorf("failed to fetch main page: %w", err)})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			sendResult(ctx, ch, StreamResult[models.Subtitle]{Err: fmt.Errorf("main page returned status %d", resp.StatusCode)})
			return
		}

		// Parse the HTML to extract subtitles
		subtitles, err := c.subtitleParser.ParseHtml(resp.Body)
		if err != nil {
			sendResult(ctx, ch, StreamResult[models.Subtitle]{Err: fmt.Errorf("failed to parse main page: %w", err)})
			return
		}

		logger.Info().Int("totalSubtitles", len(subtitles)).Msg("Parsed subtitles from main page")

		// Stream subtitles filtered by ID (only those with ID > sinceID)
		count := 0
		for _, subtitle := range subtitles {
			if sinceID == 0 || subtitle.ID > sinceID {
				select {
				case ch <- StreamResult[models.Subtitle]{Value: subtitle}:
					count++
				case <-ctx.Done():
					return
				}
			}
		}

		logger.Info().Int("streamedSubtitles", count).Msg("Finished streaming recent subtitles")
	}()

	return ch
}
