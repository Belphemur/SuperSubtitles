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

	var errorsMu sync.Mutex
	var errs []error
	var wg sync.WaitGroup
	wg.Add(len(shows))

	for _, show := range shows {
		show := show
		go func() {
			defer wg.Done()

			var thirdPartyIdsSent bool

			// Stream subtitles as they arrive
			for result := range c.StreamSubtitles(ctx, show.ID) {
				if result.Err != nil {
					logger.Warn().Err(result.Err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to stream subtitles for show")
					errorsMu.Lock()
					errs = append(errs, fmt.Errorf("failed to stream subtitles for show %d: %w", show.ID, result.Err))
					errorsMu.Unlock()
					return
				}

				// If we haven't sent ShowInfo yet and we found a valid episode ID, fetch and send third-party IDs
				if !thirdPartyIdsSent && result.Value.ID > 0 {
					thirdPartyIds := c.fetchThirdPartyIds(ctx, show, result.Value.ID)

					// Send ShowInfo immediately
					showInfo := models.ShowInfo{
						Show:          show,
						ThirdPartyIds: thirdPartyIds,
					}
					select {
					case ch <- StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{ShowInfo: &showInfo}}:
					case <-ctx.Done():
						return
					}
					thirdPartyIdsSent = true
					logger.Debug().Int("showID", show.ID).Str("showName", show.Name).Str("imdbId", thirdPartyIds.IMDBID).Int("tvdbId", thirdPartyIds.TVDBID).Msg("Sent ShowInfo with third-party IDs")
				}

				// Stream subtitle immediately
				subtitle := result.Value
				select {
				case ch <- StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{Subtitle: &subtitle}}:
				case <-ctx.Done():
					return
				}
			}

			// If we never found a valid episode ID, send ShowInfo with empty third-party IDs
			if !thirdPartyIdsSent {
				logger.Warn().Int("showID", show.ID).Str("showName", show.Name).Msg("No valid subtitle ID found, sending ShowInfo with empty third-party IDs")
				showInfo := models.ShowInfo{
					Show:          show,
					ThirdPartyIds: models.ThirdPartyIds{},
				}
				select {
				case ch <- StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{ShowInfo: &showInfo}}:
				case <-ctx.Done():
					return
				}
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
