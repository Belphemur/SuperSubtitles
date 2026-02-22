package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
)

// StreamRecentSubtitles streams recently uploaded subtitles grouped by show as ShowSubtitles entries.
// Each streamed item contains a show's info (with third-party IDs) and all its recent subtitles.
// ShowInfo is fetched once per unique show_id using an in-memory cache.
func (c *client) StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles] {
	ch := make(chan models.StreamResult[models.ShowSubtitles])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Int("sinceID", sinceID).Msg("Streaming recent subtitles from main page")

		// Fetch the main show page
		endpoint := fmt.Sprintf("%s/index.php?tab=sorozat", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("failed to create request: %w", err)})
			return
		}
		req.Header.Set("User-Agent", config.GetUserAgent())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("failed to fetch main page: %w", err)})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("main page returned status %d", resp.StatusCode)})
			return
		}

		// Parse the HTML to extract subtitles
		subtitles, err := c.subtitleParser.ParseHtml(resp.Body)
		if err != nil {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("failed to parse main page: %w", err)})
			return
		}

		logger.Info().Int("totalSubtitles", len(subtitles)).Msg("Parsed subtitles from main page")

		// Group subtitles by show, preserving encounter order
		type showData struct {
			subtitles       []models.Subtitle
			firstValidSubID int
			showName        string
		}
		showDataMap := make(map[int]*showData)
		var showOrder []int

		for _, subtitle := range subtitles {
			if sinceID != 0 && subtitle.ID <= sinceID {
				continue
			}

			showID := subtitle.ShowID

			// Skip subtitles without a valid show ID
			if showID == 0 {
				logger.Warn().Int("subtitleID", subtitle.ID).Str("showName", subtitle.ShowName).Msg("Skipping subtitle with missing show_id")
				continue
			}

			sd, exists := showDataMap[showID]
			if !exists {
				sd = &showData{showName: subtitle.ShowName}
				showDataMap[showID] = sd
				showOrder = append(showOrder, showID)
			}

			if sd.firstValidSubID == 0 && subtitle.ID > 0 {
				sd.firstValidSubID = subtitle.ID
			}
			sd.subtitles = append(sd.subtitles, subtitle)
		}

		// Stream each show's complete data
		for _, showID := range showOrder {
			sd := showDataMap[showID]

			show := models.Show{
				ID:   showID,
				Name: sd.showName,
			}

			// Fetch third-party IDs using first valid subtitle ID
			var thirdPartyIds models.ThirdPartyIds
			if sd.firstValidSubID > 0 {
				thirdPartyIds = c.fetchThirdPartyIds(ctx, show, sd.firstValidSubID)
			} else {
				logger.Warn().Int("showID", showID).Msg("No valid subtitle ID to fetch third-party IDs")
			}

			showSubtitles := models.ShowSubtitles{
				Show:          show,
				ThirdPartyIds: thirdPartyIds,
				SubtitleCollection: models.SubtitleCollection{
					ShowName:  sd.showName,
					Subtitles: sd.subtitles,
					Total:     len(sd.subtitles),
				},
			}

			select {
			case ch <- models.StreamResult[models.ShowSubtitles]{Value: showSubtitles}:
			case <-ctx.Done():
				return
			}
		}

		logger.Info().Int("uniqueShows", len(showOrder)).Msg("Finished streaming recent subtitles")
	}()

	return ch
}
