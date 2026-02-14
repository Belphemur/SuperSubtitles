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

// GetRecentSubtitles fetches recent subtitles from the main show page, filtered by subtitle ID
// Returns only subtitles with ID greater than sinceID, grouped by show with full show details
func (c *client) GetRecentSubtitles(ctx context.Context, sinceID int) ([]models.ShowSubtitles, error) {
	// Collect streamed items and group by show
	showInfoMap := make(map[int]*models.ShowInfo)
	subtitlesByShow := make(map[int][]models.Subtitle)
	var showOrder []int

	for item := range c.StreamRecentSubtitles(ctx, sinceID) {
		if item.Err != nil {
			return nil, item.Err
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

// StreamRecentSubtitles streams recently uploaded subtitles as ShowSubtitleItems.
// Fetches the main page, filters by sinceID, groups by show, and streams results.
func (c *client) StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan StreamResult[models.ShowSubtitleItem] {
	ch := make(chan StreamResult[models.ShowSubtitleItem])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Int("sinceID", sinceID).Msg("Streaming recent subtitles from main page")

		// Fetch the main show page
		endpoint := fmt.Sprintf("%s/index.php?tab=sorozat", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			sendResult(ctx, ch, StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("failed to create request: %w", err)})
			return
		}
		req.Header.Set("User-Agent", config.GetUserAgent())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			sendResult(ctx, ch, StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("failed to fetch main page: %w", err)})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			sendResult(ctx, ch, StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("main page returned status %d", resp.StatusCode)})
			return
		}

		// Parse the HTML to extract subtitles
		subtitles, err := c.subtitleParser.ParseHtml(resp.Body)
		if err != nil {
			sendResult(ctx, ch, StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("failed to parse main page: %w", err)})
			return
		}

		logger.Info().Int("totalSubtitles", len(subtitles)).Msg("Parsed subtitles from main page")

		// Filter subtitles by ID (only those with ID > sinceID)
		filteredSubtitles := make([]models.Subtitle, 0)
		for _, subtitle := range subtitles {
			if sinceID == 0 || subtitle.ID > sinceID {
				filteredSubtitles = append(filteredSubtitles, subtitle)
			}
		}

		logger.Info().Int("filteredSubtitles", len(filteredSubtitles)).Msg("Filtered subtitles by ID")

		if len(filteredSubtitles) == 0 {
			return
		}

		// Group subtitles by show ID to avoid duplicate fetches
		subtitlesByShow := make(map[int][]models.Subtitle)
		for _, subtitle := range filteredSubtitles {
			if subtitle.ShowID == 0 {
				logger.Debug().Int("subtitleID", subtitle.ID).Msg("Skipping subtitle with no show ID")
				continue
			}
			subtitlesByShow[subtitle.ShowID] = append(subtitlesByShow[subtitle.ShowID], subtitle)
		}

		logger.Info().Int("uniqueShows", len(subtitlesByShow)).Msg("Grouped subtitles by show")

		// Convert map to slice for batched processing
		type showBatch struct {
			showID    int
			subtitles []models.Subtitle
		}
		var showBatches []showBatch
		for sid, subs := range subtitlesByShow {
			showBatches = append(showBatches, showBatch{showID: sid, subtitles: subs})
		}

		const batchSize = 20

		type showResult struct {
			showInfo  models.ShowInfo
			subtitles []models.Subtitle
			err       error
		}

		var allErrs []error
		successCount := 0

		// Process shows in batches to limit concurrency
		for i := 0; i < len(showBatches); i += batchSize {
			end := i + batchSize
			if end > len(showBatches) {
				end = len(showBatches)
			}

			batch := showBatches[i:end]
			logger.Debug().Int("batchStart", i).Int("batchEnd", end-1).Int("batchSize", len(batch)).Msg("Processing batch of shows")

			var wg sync.WaitGroup
			batchResults := make([]showResult, len(batch))

			for idx, item := range batch {
				wg.Add(1)
				go func() {
					defer wg.Done()

					sid := item.showID
					subs := item.subtitles

					// Fetch show details using the first valid subtitle ID
					var episodeID int
					if len(subs) == 0 {
						logger.Warn().Int("showID", sid).Msg("No subtitles for show, skipping")
						return
					}
					for _, subtitle := range subs {
						if subtitle.ID > 0 {
							episodeID = subtitle.ID
							break
						}
					}
					if episodeID == 0 {
						logger.Warn().Int("showID", sid).Msg("No valid subtitle IDs for show, skipping")
						return
					}

					// Construct detail page URL to get third-party IDs
					detailURL := fmt.Sprintf("%s/index.php?tipus=adatlap&azon=a_%d", c.baseURL, episodeID)

					req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
					if err != nil {
						logger.Warn().Err(err).Int("showID", sid).Str("detailURL", detailURL).Msg("Failed to create detail request")
						batchResults[idx] = showResult{err: fmt.Errorf("failed to create detail request for show %d (%s): %w", sid, detailURL, err)}
						return
					}
					req.Header.Set("User-Agent", config.GetUserAgent())

					resp, err := c.httpClient.Do(req)
					if err != nil {
						logger.Warn().Err(err).Int("showID", sid).Str("detailURL", detailURL).Msg("Failed to fetch detail page")
						batchResults[idx] = showResult{err: fmt.Errorf("failed to fetch detail page for show %d (%s): %w", sid, detailURL, err)}
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						err := fmt.Errorf("detail page for show %d (%s) returned status %d", sid, detailURL, resp.StatusCode)
						logger.Warn().Err(err).Int("showID", sid).Str("detailURL", detailURL).Msg("Detail page error")
						batchResults[idx] = showResult{err: err}
						return
					}

					// Parse third-party IDs from HTML
					thirdPartyIds, err := c.thirdPartyParser.ParseHtml(resp.Body)
					if err != nil {
						logger.Warn().Err(err).Int("showID", sid).Msg("Failed to parse third-party IDs, continuing with empty IDs")
						thirdPartyIds = models.ThirdPartyIds{}
					}

					// Build Show object from subtitle data
					showName := ""
					if len(subs) > 0 {
						showName = subs[0].ShowName
					}

					show := models.Show{
						ID:       sid,
						Name:     showName,
						ImageURL: "",
						Year:     0,
					}

					logger.Debug().Int("showID", sid).Str("showName", showName).Int("subtitleCount", len(subs)).Msg("Successfully processed show")

					batchResults[idx] = showResult{
						showInfo: models.ShowInfo{
							Show:          show,
							ThirdPartyIds: thirdPartyIds,
						},
						subtitles: subs,
					}
				}()
			}

			wg.Wait()

			// Stream results from this batch
			for _, result := range batchResults {
				if result.err != nil {
					allErrs = append(allErrs, result.err)
					continue
				}
				if len(result.subtitles) == 0 {
					// Skip shows that returned early (no subtitles)
					continue
				}

				successCount++

				// Send ShowInfo
				select {
				case ch <- StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{ShowInfo: &result.showInfo}}:
				case <-ctx.Done():
					return
				}

				// Send each subtitle
				for _, subtitle := range result.subtitles {
					subtitle := subtitle
					select {
					case ch <- StreamResult[models.ShowSubtitleItem]{Value: models.ShowSubtitleItem{Subtitle: &subtitle}}:
					case <-ctx.Done():
						return
					}
				}
			}
		}

		if successCount == 0 && len(allErrs) > 0 {
			sendResult(ctx, ch, StreamResult[models.ShowSubtitleItem]{Err: fmt.Errorf("all shows failed processing: %v", errors.Join(allErrs...))})
		} else if len(allErrs) > 0 {
			logger.Warn().Err(errors.Join(allErrs...)).Int("successfulShows", successCount).Int("totalShows", len(subtitlesByShow)).Msg("Partial success processing recent subtitles")
		} else {
			logger.Info().Int("totalShows", successCount).Msg("Successfully processed all recent subtitles")
		}
	}()

	return ch
}
