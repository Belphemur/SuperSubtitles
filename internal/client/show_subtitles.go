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
	// Collect streamed items and group by show
	showInfoMap := make(map[int]*models.ShowInfo)
	subtitlesByShow := make(map[int][]models.Subtitle)
	var showOrder []int

	for item := range c.StreamShowSubtitles(ctx, shows) {
		if item.Err != nil {
			// Log but continue — partial success
			logger.Warn().Err(item.Err).Msg("Error in show subtitle stream")
			continue
		}
		if item.Value.ShowInfo != nil {
			sid := item.Value.ShowInfo.Show.ID
			showInfoMap[sid] = item.Value.ShowInfo
			showOrder = append(showOrder, sid)
		}
		if item.Value.Subtitle != nil {
			subtitlesByShow[item.Value.Subtitle.ShowID] = append(subtitlesByShow[item.Value.Subtitle.ShowID], *item.Value.Subtitle)
		}
	}

	if len(showInfoMap) == 0 {
		return nil, fmt.Errorf("all shows failed processing")
	}

	// Build ShowSubtitles results in order
	var results []models.ShowSubtitles
	for _, sid := range showOrder {
		info := showInfoMap[sid]
		subs := subtitlesByShow[sid]
		showName := info.Show.Name
		if len(subs) > 0 {
			showName = subs[0].ShowName
		}
		results = append(results, models.ShowSubtitles{
			Show:          info.Show,
			ThirdPartyIds: info.ThirdPartyIds,
			SubtitleCollection: models.SubtitleCollection{
				ShowName:  showName,
				Subtitles: subs,
				Total:     len(subs),
			},
		})
	}

	return results, nil
}

// StreamShowSubtitles streams ShowSubtitleItems (ShowInfo and Subtitle) for multiple shows.
// For each show, it first sends a ShowInfo item, then streams each subtitle.
// Shows are processed in batches of 20 to limit concurrency.
func (c *client) StreamShowSubtitles(ctx context.Context, shows []models.Show) <-chan StreamResult[models.ShowSubtitleItem] {
	ch := make(chan StreamResult[models.ShowSubtitleItem])

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
			sendResult(ctx, ch, StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("all shows failed processing: %v", errors.Join(allErrors...))})
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
func (c *client) streamShowBatch(ctx context.Context, shows []models.Show, ch chan<- StreamResult[models.ShowSubtitleItem]) []error {
	logger := config.GetLogger()

	type showResult struct {
		showInfo  models.ShowInfo
		subtitles []models.Subtitle
		err       error
	}

	results := make([]showResult, len(shows))
	var wg sync.WaitGroup
	wg.Add(len(shows))

	for i, show := range shows {
		i, show := i, show
		go func() {
			defer wg.Done()

			// Get subtitles for this show
			subtitles, err := c.GetSubtitles(ctx, show.ID)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to fetch subtitles for show")
				results[i] = showResult{err: fmt.Errorf("failed to get subtitles for show %d: %w", show.ID, err)}
				return
			}

			// Find a valid episode ID from the subtitles (use first non-zero subtitle ID)
			var episodeID int
			for _, s := range subtitles.Subtitles {
				if s.ID > 0 {
					episodeID = s.ID
					break
				}
			}

			var thirdPartyIds models.ThirdPartyIds
			if episodeID == 0 {
				logger.Warn().Int("showID", show.ID).Str("showName", show.Name).Msg("No valid subtitle ID found, cannot fetch third-party IDs")
			} else {
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
				ids, err := c.thirdPartyParser.ParseHtml(resp.Body)
				if err != nil {
					logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to parse third-party IDs")
				} else {
					thirdPartyIds = ids
				}
			}

			logger.Debug().Int("showID", show.ID).Str("showName", show.Name).Str("imdbId", thirdPartyIds.IMDBID).Int("tvdbId", thirdPartyIds.TVDBID).Msg("Successfully processed show")
			results[i] = showResult{
				showInfo: models.ShowInfo{
					Show:          show,
					ThirdPartyIds: thirdPartyIds,
				},
				subtitles: subtitles.Subtitles,
			}
		}()
	}

	wg.Wait()

	// Stream results and collect errors
	var errs []error
	for _, result := range results {
		if result.err != nil {
			errs = append(errs, result.err)
			continue
		}

		// Send ShowInfo
		select {
		case ch <- StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{ShowInfo: &result.showInfo}}:
		case <-ctx.Done():
			return errs
		}

		// Send each subtitle
		for _, subtitle := range result.subtitles {
			subtitle := subtitle
			select {
			case ch <- StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{Subtitle: &subtitle}}:
			case <-ctx.Done():
				return errs
			}
		}
	}

	return errs
}
