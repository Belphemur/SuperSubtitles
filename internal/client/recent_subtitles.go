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
//
// When sinceID > 0, pages are fetched sequentially until a subtitle with ID <= sinceID is
// encountered, ensuring all newer subtitles are collected across multiple pages.
// When sinceID == 0, only the first page is fetched.
func (c *client) StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles] {
	ch := make(chan models.StreamResult[models.ShowSubtitles])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Int("sinceID", sinceID).Msg("Streaming recent subtitles from main page")

		// Group subtitles by show, preserving encounter order across all pages
		type showData struct {
			subtitles       []models.Subtitle
			firstValidSubID int
			showName        string
		}
		showDataMap := make(map[int]*showData)
		var showOrder []int

		// addSubtitle accumulates a single subtitle into the show grouping.
		// Returns true if the subtitle was at or below the sinceID boundary.
		addSubtitle := func(subtitle models.Subtitle) (reachedBoundary bool) {
			if sinceID > 0 && subtitle.ID <= sinceID {
				return true
			}

			showID := subtitle.ShowID
			if showID == 0 {
				logger.Warn().Int("subtitleID", subtitle.ID).Str("showName", subtitle.ShowName).Msg("Skipping subtitle with missing show_id")
				return false
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
			return false
		}

		// Fetch pages sequentially until we reach the sinceID boundary
		baseEndpoint := fmt.Sprintf("%s/index.php?tab=sorozat", c.baseURL)
		reachedBoundary := false
		for page := 1; !reachedBoundary; page++ {
			endpoint := baseEndpoint
			if page > 1 {
				endpoint = fmt.Sprintf("%s&oldal=%d", baseEndpoint, page)
			}

			req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
			if err != nil {
				sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("failed to create request for page %d: %w", page, err)})
				return
			}
			req.Header.Set("User-Agent", config.GetUserAgent())

			resp, err := c.httpClient.Do(req)
			if err != nil {
				sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("failed to fetch page %d: %w", page, err)})
				return
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("page %d returned status %d", page, resp.StatusCode)})
				return
			}

			pageResult, err := c.subtitleParser.ParseHtmlWithPagination(resp.Body)
			resp.Body.Close()
			if err != nil {
				sendResult(ctx, ch, models.StreamResult[models.ShowSubtitles]{Err: fmt.Errorf("failed to parse page %d: %w", page, err)})
				return
			}

			logger.Info().
				Int("page", page).
				Int("totalPages", pageResult.TotalPages).
				Int("subtitles", len(pageResult.Subtitles)).
				Msg("Parsed subtitles from page")

			for _, subtitle := range pageResult.Subtitles {
				if addSubtitle(subtitle) {
					reachedBoundary = true
					break
				}
			}

			// When sinceID is 0, only fetch the first page
			if sinceID == 0 || !pageResult.HasNextPage {
				break
			}

			// Check for context cancellation between pages
			select {
			case <-ctx.Done():
				return
			default:
			}
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
