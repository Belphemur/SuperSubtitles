package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// GetShowSubtitles retrieves third-party IDs and subtitle collections for multiple shows in parallel
// Processes shows in batches of 20 to avoid overwhelming the server
func (c *client) GetShowSubtitles(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error) {
	logger := config.GetLogger()
	logger.Info().Int("showCount", len(shows)).Msg("Fetching third-party IDs and subtitles for shows in batches")

	const batchSize = 20
	var allShowSubtitles []models.ShowSubtitles
	var allErrors []error

	// Process shows in batches
	for i := 0; i < len(shows); i += batchSize {
		end := i + batchSize
		if end > len(shows) {
			end = len(shows)
		}

		batch := shows[i:end]
		logger.Info().Int("batchStart", i).Int("batchEnd", end-1).Int("batchSize", len(batch)).Msg("Processing batch of shows")

		batchResults, batchErrors := c.processShowBatch(ctx, batch)

		allShowSubtitles = append(allShowSubtitles, batchResults...)
		allErrors = append(allErrors, batchErrors...)
	}

	if len(allShowSubtitles) == 0 && len(allErrors) > 0 {
		// All shows failed
		return nil, fmt.Errorf("all shows failed processing: %v", errors.Join(allErrors...))
	}

	if len(allErrors) > 0 {
		// Partial success - log aggregated error but still return data
		logger.Warn().Err(errors.Join(allErrors...)).Int("successfulShows", len(allShowSubtitles)).Int("totalShows", len(shows)).Msg("Partial success processing shows")
	} else {
		logger.Info().Int("totalShows", len(allShowSubtitles)).Msg("Successfully processed all shows")
	}

	return allShowSubtitles, nil
}

// processShowBatch processes a batch of shows concurrently (up to batch size)
func (c *client) processShowBatch(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, []error) {
	logger := config.GetLogger()

	type showResult struct {
		showSubtitles models.ShowSubtitles
		err           error
	}

	results := make([]showResult, len(shows))
	var wg sync.WaitGroup
	wg.Add(len(shows))

	for i, show := range shows {
		i, show := i, show // Capture loop variables
		go func() {
			defer wg.Done()

			// Get subtitles for this show
			subtitles, err := c.GetSubtitles(ctx, show.ID)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to fetch subtitles for show")
				results[i] = showResult{err: fmt.Errorf("failed to get subtitles for show %d: %w", show.ID, err)}
				return
			}

			// Find an episode ID from the subtitles (use first subtitle's ID)
			var episodeID int
			if len(subtitles.Subtitles) > 0 {
				episodeID = subtitles.Subtitles[0].ID
			} else {
				logger.Warn().Int("showID", show.ID).Str("showName", show.Name).Msg("No subtitles found, cannot fetch third-party IDs")
				// Create ShowSubtitles without third-party IDs
				results[i] = showResult{
					showSubtitles: models.ShowSubtitles{
						Show:               show,
						ThirdPartyIds:      models.ThirdPartyIds{}, // Empty
						SubtitleCollection: *subtitles,
					},
				}
				return
			}

			// Construct detail page URL
			detailURL := fmt.Sprintf("%s/index.php?tipus=adatlap&azon=a_%d", c.baseURL, episodeID)

			// Fetch detail page HTML
			req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to create detail page request")
				results[i] = showResult{err: fmt.Errorf("failed to create detail page request for show %d: %w", show.ID, err)}
				return
			}
			req.Header.Set("User-Agent", config.GetUserAgent())

			resp, err := c.httpClient.Do(req)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Str("detailURL", detailURL).Msg("Failed to fetch detail page")
				results[i] = showResult{err: fmt.Errorf("failed to fetch detail page for show %d: %w", show.ID, err)}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				logger.Warn().Int("statusCode", resp.StatusCode).Int("showID", show.ID).Str("showName", show.Name).Str("detailURL", detailURL).Msg("Detail page returned non-OK status")
				results[i] = showResult{err: fmt.Errorf("detail page for show %d returned status %d", show.ID, resp.StatusCode)}
				return
			}

			// Parse third-party IDs from HTML
			thirdPartyIds, err := c.thirdPartyParser.ParseHtml(resp.Body)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to parse third-party IDs")
				// Don't fail completely, just log and continue with empty third-party IDs
				thirdPartyIds = models.ThirdPartyIds{}
			}

			// Create ShowSubtitles object
			showSubtitles := models.ShowSubtitles{
				Show:               show,
				ThirdPartyIds:      thirdPartyIds,
				SubtitleCollection: *subtitles,
			}

			logger.Debug().Int("showID", show.ID).Str("showName", show.Name).Str("imdbId", thirdPartyIds.IMDBID).Int("tvdbId", thirdPartyIds.TVDBID).Msg("Successfully processed show")
			results[i] = showResult{showSubtitles: showSubtitles}
		}()
	}

	wg.Wait()

	// Collect successful results and errors
	var showSubtitlesList []models.ShowSubtitles
	var errs []error
	for _, result := range results {
		if result.err != nil {
			errs = append(errs, result.err)
		} else {
			showSubtitlesList = append(showSubtitlesList, result.showSubtitles)
		}
	}

	return showSubtitlesList, errs
}
