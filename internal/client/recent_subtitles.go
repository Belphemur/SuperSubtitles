package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// StreamRecentSubtitles streams recently uploaded subtitles as ShowSubtitles Items.
// For each new show encountered, a ShowInfo item is sent first (with third-party IDs),
// followed by individual Subtitle items. ShowInfo is only sent once per unique show_id
// within a single call using an in-memory cache.
func (c *client) StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitleItem] {
	ch := make(chan models.StreamResult[models.ShowSubtitleItem])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Int("sinceID", sinceID).Msg("Streaming recent subtitles from main page")

		// Fetch the main show page
		endpoint := fmt.Sprintf("%s/index.php?tab=sorozat", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("failed to create request: %w", err)})
			return
		}
		req.Header.Set("User-Agent", config.GetUserAgent())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("failed to fetch main page: %w", err)})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("main page returned status %d", resp.StatusCode)})
			return
		}

		// Parse the HTML to extract subtitles
		subtitles, err := c.subtitleParser.ParseHtml(resp.Body)
		if err != nil {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("failed to parse main page: %w", err)})
			return
		}

		logger.Info().Int("totalSubtitles", len(subtitles)).Msg("Parsed subtitles from main page")

		// Filter subtitles by ID and stream them with show info
		// Track which shows we've already sent ShowInfo for
		sentShowInfo := make(map[int]bool)
		count := 0

		for _, subtitle := range subtitles {
			if sinceID != 0 && subtitle.ID <= sinceID {
				continue
			}

			showID := subtitle.ShowID

			// Skip subtitles without a valid show ID to avoid orphaned items
			if showID == 0 {
				logger.Warn().Int("subtitleID", subtitle.ID).Str("showName", subtitle.ShowName).Msg("Skipping subtitle with missing show_id")
				continue
			}

			// If we haven't sent ShowInfo for this show yet, fetch and send it
			if !sentShowInfo[showID] {
				showInfo := c.fetchShowInfoForRecent(ctx, subtitle)
				select {
				case ch <- models.StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{ShowInfo: &showInfo}}:
					sentShowInfo[showID] = true
				case <-ctx.Done():
					return
				}
			}

			// Send the subtitle
			sub := subtitle
			select {
			case ch <- models.StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{Subtitle: &sub}}:
				count++
			case <-ctx.Done():
				return
			}
		}

		logger.Info().Int("streamedSubtitles", count).Int("uniqueShows", len(sentShowInfo)).Msg("Finished streaming recent subtitles")
	}()

	return ch
}

// fetchShowInfoForRecent builds a ShowInfo for a subtitle, fetching third-party IDs from the detail page.
func (c *client) fetchShowInfoForRecent(ctx context.Context, subtitle models.Subtitle) models.ShowInfo {
	logger := config.GetLogger()

	show := models.Show{
		ID:   subtitle.ShowID,
		Name: subtitle.ShowName,
	}

	var thirdPartyIds models.ThirdPartyIds

	if subtitle.ID <= 0 {
		logger.Warn().Int("showID", subtitle.ShowID).Msg("No valid subtitle ID to fetch third-party IDs")
		return models.ShowInfo{Show: show, ThirdPartyIds: thirdPartyIds}
	}

	// Construct detail page URL to get third-party IDs
	detailURL := fmt.Sprintf("%s/index.php?tipus=adatlap&azon=a_%d", c.baseURL, subtitle.ID)

	req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
	if err != nil {
		logger.Warn().Err(err).Int("showID", subtitle.ShowID).Str("detailURL", detailURL).Msg("Failed to create detail request")
		return models.ShowInfo{Show: show, ThirdPartyIds: thirdPartyIds}
	}
	req.Header.Set("User-Agent", config.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Warn().Err(err).Int("showID", subtitle.ShowID).Str("detailURL", detailURL).Msg("Failed to fetch detail page")
		return models.ShowInfo{Show: show, ThirdPartyIds: thirdPartyIds}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn().Int("statusCode", resp.StatusCode).Int("showID", subtitle.ShowID).Str("detailURL", detailURL).Msg("Detail page returned non-OK status")
		return models.ShowInfo{Show: show, ThirdPartyIds: thirdPartyIds}
	}

	ids, err := c.thirdPartyParser.ParseHtml(resp.Body)
	if err != nil {
		logger.Warn().Err(err).Int("showID", subtitle.ShowID).Msg("Failed to parse third-party IDs, continuing with empty IDs")
	} else {
		thirdPartyIds = ids
	}

	return models.ShowInfo{Show: show, ThirdPartyIds: thirdPartyIds}
}
