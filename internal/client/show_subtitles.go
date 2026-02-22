package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
)

// StreamShowSubtitles streams complete ShowSubtitles (show info + all subtitles) for multiple shows.
// For each show, it accumulates all subtitles, fetches third-party IDs, then sends the complete collection.
// Shows are processed in batches of 20 to limit concurrency.
func (c *client) StreamShowSubtitles(ctx context.Context, shows []models.Show) <-chan models.StreamResult[models.ShowSubtitles] {
	ch := make(chan models.StreamResult[models.ShowSubtitles])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Int("showCount", len(shows)).Msg("Streaming show subtitles in batches")

		const batchSize = 20
		var allErrors []error
		successCount := 0

		for i := 0; i < len(shows); i += batchSize {
			end := i + batchSize
			if end > len(shows) {
				end = len(shows)
			}

			batch := shows[i:end]
			logger.Info().Int("batchStart", i).Int("batchEnd", end-1).Int("batchSize", len(batch)).Msg("Processing batch of shows")

			batchErrors := c.streamShowBatch(ctx, batch, ch)
			successCount += len(batch) - len(batchErrors)
			allErrors = append(allErrors, batchErrors...)
		}

		if successCount == 0 && len(allErrors) > 0 {
			sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("all shows failed processing: %v", errors.Join(allErrors...))})
		} else if len(allErrors) > 0 {
			logger.Warn().Err(errors.Join(allErrors...)).Int("successfulShows", successCount).Int("totalShows", len(shows)).Msg("Partial success processing shows")
		} else {
			logger.Info().Int("totalShows", successCount).Msg("Successfully processed all shows")
		}
	}()

	return ch
}

// streamShowBatch processes a batch of shows concurrently, streaming results to the channel.
// Returns a list of errors for shows that failed.
func (c *client) streamShowBatch(ctx context.Context, shows []models.Show, ch chan<- models.StreamResult[models.ShowSubtitles]) []error {
	logger := config.GetLogger()

	var errorsMu sync.Mutex
	var errs []error
	var wg sync.WaitGroup
	wg.Add(len(shows))

	for _, show := range shows {
		show := show
		go func() {
			defer wg.Done()

			// Accumulate all subtitles for this show
			var subtitles []models.Subtitle
			var firstValidSubtitleID int

			for result := range c.StreamSubtitles(ctx, show.ID) {
				if result.Err != nil {
					logger.Warn().Err(result.Err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to stream subtitles for show")
					errorsMu.Lock()
					errs = append(errs, fmt.Errorf("failed to stream subtitles for show %d: %w", show.ID, result.Err))
					errorsMu.Unlock()
					return
				}
				// Log error and skip subtitle if ID is invalid
				if result.Value.ID <= 0 {
					logger.Error().
						Int("showID", show.ID).
						Str("showName", show.Name).
						Int("subtitleID", result.Value.ID).
						Str("subtitleName", result.Value.Name).
						Int("season", result.Value.Season).
						Int("episode", result.Value.Episode).
						Str("language", result.Value.Language).
						Str("filename", result.Value.Filename).
						Msg("Received subtitle with invalid ID, discarding")
					continue
				}

				if firstValidSubtitleID == 0 {
					firstValidSubtitleID = result.Value.ID
				}
				subtitles = append(subtitles, result.Value)
			}

			// Fetch third-party IDs using first valid subtitle ID
			var thirdPartyIds models.ThirdPartyIds
			if firstValidSubtitleID > 0 {
				thirdPartyIds = c.fetchThirdPartyIds(ctx, show, firstValidSubtitleID)
				foundThirdPartyIds := thirdPartyIds.IMDBID != "" || thirdPartyIds.TVDBID != 0
				logger.Debug().
					Int("showID", show.ID).
					Str("showName", show.Name).
					Str("imdbId", thirdPartyIds.IMDBID).
					Int("tvdbId", thirdPartyIds.TVDBID).
					Bool("foundThirdPartyIds", foundThirdPartyIds).
					Msg("Fetched third-party IDs")
			} else {
				logger.Warn().Int("showID", show.ID).Str("showName", show.Name).Msg("No valid subtitle ID found, sending with empty third-party IDs")
			}

			// Build show name from subtitles if available
			showName := show.Name
			if len(subtitles) > 0 {
				showName = subtitles[0].ShowName
			}

			// Send complete ShowSubtitles
			showSubtitles := models.ShowSubtitles{
				Show:          show,
				ThirdPartyIds: thirdPartyIds,
				SubtitleCollection: models.SubtitleCollection{
					ShowName:  showName,
					Subtitles: subtitles,
					Total:     len(subtitles),
				},
			}

			select {
			case ch <- models.StreamResult[models.ShowSubtitles]{Value: showSubtitles}:
			case <-ctx.Done():
				return
			}
		}()
	}

	wg.Wait()
	return errs
}

// fetchThirdPartyIds fetches third-party IDs for a show using the given episode ID.
// Returns empty ThirdPartyIds on error (logs warning but doesn't fail).
func (c *client) fetchThirdPartyIds(ctx context.Context, show models.Show, episodeID int) models.ThirdPartyIds {
	logger := config.GetLogger()

	// Construct detail page URL
	detailURL := fmt.Sprintf("%s/index.php?tipus=adatlap&azon=a_%d", c.baseURL, episodeID)

	// Fetch detail page HTML
	req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
	if err != nil {
		logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to create detail page request")
		return models.ThirdPartyIds{}
	}
	req.Header.Set("User-Agent", config.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Str("detailURL", detailURL).Msg("Failed to fetch detail page")
		return models.ThirdPartyIds{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn().Int("statusCode", resp.StatusCode).Int("showID", show.ID).Str("showName", show.Name).Str("detailURL", detailURL).Msg("Detail page returned non-OK status")
		return models.ThirdPartyIds{}
	}

	// Parse third-party IDs from HTML
	ids, err := c.thirdPartyParser.ParseHtml(resp.Body)
	if err != nil {
		logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to parse third-party IDs")
		return models.ThirdPartyIds{}
	}

	return ids
}
