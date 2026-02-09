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
func (c *client) GetRecentSubtitles(ctx context.Context, sinceID string) ([]models.ShowSubtitles, error) {
	logger := config.GetLogger()
	logger.Info().Str("sinceID", sinceID).Msg("Fetching recent subtitles from main page")

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
		if sinceID == "" || subtitle.ID > sinceID {
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
			logger.Debug().Str("subtitleID", subtitle.ID).Msg("Skipping subtitle with no show ID")
			continue
		}
		subtitlesByShow[subtitle.ShowID] = append(subtitlesByShow[subtitle.ShowID], subtitle)
	}

	logger.Info().Int("uniqueShows", len(subtitlesByShow)).Msg("Grouped subtitles by show")

	// Fetch show details for each unique show
	type showResult struct {
		showSubtitles models.ShowSubtitles
		err           error
	}

	results := make([]showResult, 0, len(subtitlesByShow))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for showID, subs := range subtitlesByShow {
		wg.Add(1)
		go func(sid int, subtitles []models.Subtitle) {
			defer wg.Done()

			// Fetch show details using the first subtitle to get episode ID
			var episodeID string
			if len(subtitles) > 0 {
				episodeID = subtitles[0].ID
			} else {
				logger.Warn().Int("showID", sid).Msg("No subtitles for show, skipping")
				return
			}

			// Construct detail page URL to get third-party IDs
			detailURL := fmt.Sprintf("%s/index.php?tipus=adatlap&azon=a_%s", c.baseURL, episodeID)

			req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
			if err != nil {
				logger.Warn().Err(err).Int("showID", sid).Msg("Failed to create detail request")
				mu.Lock()
				results = append(results, showResult{err: err})
				mu.Unlock()
				return
			}
			req.Header.Set("User-Agent", config.GetUserAgent())

			resp, err := c.httpClient.Do(req)
			if err != nil {
				logger.Warn().Err(err).Int("showID", sid).Msg("Failed to fetch detail page")
				mu.Lock()
				results = append(results, showResult{err: err})
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				err := fmt.Errorf("detail page returned status %d", resp.StatusCode)
				logger.Warn().Err(err).Int("showID", sid).Msg("Detail page error")
				mu.Lock()
				results = append(results, showResult{err: err})
				mu.Unlock()
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

			mu.Lock()
			results = append(results, showResult{showSubtitles: showSubtitles})
			mu.Unlock()
		}(showID, subs)
	}

	wg.Wait()

	// Collect successful results
	var showSubtitlesList []models.ShowSubtitles
	var errs []error
	for _, result := range results {
		if result.err != nil {
			errs = append(errs, result.err)
		} else {
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
