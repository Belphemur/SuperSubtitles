package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
)

// StreamRecentSubtitles streams recently uploaded subtitles, grouped by show as ShowSubtitles entries.
// For each fetched page, subtitles are grouped by show and emitted in that page's encounter order.
// A show can be emitted multiple times across pages as additional subtitles are discovered.
// ShowInfo is fetched once per unique show_id using an in-memory cache.
//
// When sinceID > 0, pages are fetched sequentially until a subtitle with ID <= sinceID is
// encountered, ensuring all newer subtitles from each page are collected.
// When sinceID == 0, only the first page is fetched.
func (c *client) StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles] {
	ch := make(chan models.StreamResult[models.ShowSubtitles])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Int("sinceID", sinceID).Msg("Streaming recent subtitles from main page")

		// Group subtitles by show; shows are emitted in encounter order within each page
		type showData struct {
			subtitles       []models.Subtitle
			firstValidSubID int
			showName        string
		}
		showDataMap := make(map[int]*showData)
		thirdPartyIDsByShow := make(map[int]models.ThirdPartyIds)
		totalEmitted := 0

		buildShowSubtitles := func(showID int) models.ShowSubtitles {
			sd := showDataMap[showID]
			show := models.Show{ID: showID, Name: sd.showName}

			if _, exists := thirdPartyIDsByShow[showID]; !exists {
				if sd.firstValidSubID > 0 {
					thirdPartyIDsByShow[showID] = c.fetchThirdPartyIds(ctx, show, sd.firstValidSubID)
				} else {
					logger.Warn().Int("showID", showID).Msg("No valid subtitle ID to fetch third-party IDs")
					thirdPartyIDsByShow[showID] = models.ThirdPartyIds{}
				}
			}

			return models.ShowSubtitles{
				Show:          show,
				ThirdPartyIds: thirdPartyIDsByShow[showID],
				SubtitleCollection: models.SubtitleCollection{
					ShowName:  sd.showName,
					Subtitles: sd.subtitles,
					Total:     len(sd.subtitles),
				},
			}
		}

		// Fetch pages sequentially until we reach the sinceID boundary
		baseEndpoint := fmt.Sprintf("%s/index.php?tab=sorozat", c.baseURL)
		reachedBoundary := false
		for page := 1; !reachedBoundary; page++ {
			endpoint := baseEndpoint
			if page > 1 {
				endpoint = fmt.Sprintf("%s&page=%d", baseEndpoint, page)
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

			pageShowOrder := make([]int, 0, 20)
			pageShowSeen := make(map[int]bool)
			for _, subtitle := range pageResult.Subtitles {
				if subtitle.ID <= 0 {
					logger.Error().
						Str("showName", subtitle.ShowName).
						Str("downloadURL", subtitle.DownloadURL).
						Str("filename", subtitle.Filename).
						Str("language", subtitle.Language).
						Int("season", subtitle.Season).
						Int("episode", subtitle.Episode).
						Msg("Subtitle has invalid ID (HTML parsing failure); skipping row - check HTML structure and extractIDFromDownloadLink")
					continue
				}

				if sinceID > 0 && subtitle.ID <= sinceID {
					reachedBoundary = true
					break
				}

				showID := subtitle.ShowID
				if showID == 0 {
					logger.Warn().Int("subtitleID", subtitle.ID).Str("showName", subtitle.ShowName).Msg("Skipping subtitle with missing show_id")
					continue
				}

				sd, exists := showDataMap[showID]
				if !exists {
					sd = &showData{showName: subtitle.ShowName}
					showDataMap[showID] = sd
				}

				if sd.firstValidSubID == 0 {
					sd.firstValidSubID = subtitle.ID
				}
				sd.subtitles = append(sd.subtitles, subtitle)

				if !pageShowSeen[showID] {
					pageShowSeen[showID] = true
					pageShowOrder = append(pageShowOrder, showID)
				}
			}

			for _, showID := range pageShowOrder {
				showSubtitles := buildShowSubtitles(showID)
				select {
				case ch <- models.StreamResult[models.ShowSubtitles]{Value: showSubtitles}:
					totalEmitted++
				case <-ctx.Done():
					return
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

		logger.Info().Int("uniqueShows", len(showDataMap)).Int("emittedItems", totalEmitted).Msg("Finished streaming recent subtitles")
	}()

	return ch
}
