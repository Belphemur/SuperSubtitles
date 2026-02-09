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
	logger := config.GetLogger()
	logger.Info().Int("sinceID", sinceID).Msg("Fetching recent subtitles from main page")

	// Fetch the main show page
	endpoint := fmt.Sprintf("%s/index.php?tab=sorozat", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", config.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch main page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("main page returned status %d", resp.StatusCode)
	}

	// Parse the HTML to extract subtitles
	subtitles, err := c.subtitleParser.ParseHtml(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse main page: %w", err)
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
		return []models.ShowSubtitles{}, nil
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

	// Fetch show details for each unique show with concurrency limit (batch size 20)
	type showResult struct {
		showSubtitles models.ShowSubtitles
		err           error
	}

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
	var allResults []showResult
	var mu sync.Mutex

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
			idx, item := idx, item // Capture loop variables
			go func() {
				defer wg.Done()

				sid := item.showID
				subtitles := item.subtitles

				// Fetch show details using the first valid subtitle ID to get episode ID
				var episodeID int
				if len(subtitles) == 0 {
					logger.Warn().Int("showID", sid).Msg("No subtitles for show, skipping")
					return
				}
				for _, subtitle := range subtitles {
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
				if len(subtitles) > 0 {
					showName = subtitles[0].ShowName
				}

				show := models.Show{
					ID:       sid,
					Name:     showName,
					ImageURL: "", // Not available from main page
					Year:     0,  // Not available from main page
				}

				// Create SubtitleCollection
				subtitleCollection := models.SubtitleCollection{
					ShowName:  showName,
					Subtitles: subtitles,
					Total:     len(subtitles),
				}

				// Create ShowSubtitles object
				showSubtitles := models.ShowSubtitles{
					Show:               show,
					ThirdPartyIds:      thirdPartyIds,
					SubtitleCollection: subtitleCollection,
				}

				logger.Debug().Int("showID", sid).Str("showName", showName).Int("subtitleCount", len(subtitles)).Msg("Successfully processed show")

				batchResults[idx] = showResult{showSubtitles: showSubtitles}
			}()
		}

		wg.Wait()

		// Collect results from this batch
		mu.Lock()
		allResults = append(allResults, batchResults...)
		mu.Unlock()
	}

	// Collect successful results
	var showSubtitlesList []models.ShowSubtitles
	var errs []error
	for _, result := range allResults {
		if result.err != nil {
			errs = append(errs, result.err)
		} else if result.showSubtitles.Show.ID != 0 { // Skip empty results from skipped shows
			showSubtitlesList = append(showSubtitlesList, result.showSubtitles)
		}
	}

	if len(showSubtitlesList) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all shows failed processing: %v", errors.Join(errs...))
	}

	if len(errs) > 0 {
		logger.Warn().Err(errors.Join(errs...)).Int("successfulShows", len(showSubtitlesList)).Int("totalShows", len(subtitlesByShow)).Msg("Partial success processing recent subtitles")
	} else {
		logger.Info().Int("totalShows", len(showSubtitlesList)).Msg("Successfully processed all recent subtitles")
	}

	return showSubtitlesList, nil
}
