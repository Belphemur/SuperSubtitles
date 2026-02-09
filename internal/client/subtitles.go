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

// GetSubtitles fetches subtitles for a given show ID from HTML pages with pagination support
// Fetches multiple pages in parallel (2 at a time) and aggregates results
func (c *client) GetSubtitles(ctx context.Context, showID int) (*models.SubtitleCollection, error) {
	logger := config.GetLogger()
	logger.Info().Int("showID", showID).Msg("Fetching subtitles for show via HTML with pagination")

	// Fetch first page
	endpoint := fmt.Sprintf("%s/index.php?sid=%d", c.baseURL, showID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for first page: %w", err)
	}
	req.Header.Set("User-Agent", config.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch first page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("first page returned status %d", resp.StatusCode)
	}

	// Parse first page with pagination info
	firstPageResult, err := c.subtitleParser.ParseHtmlWithPagination(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse first page: %w", err)
	}

	logger.Info().
		Int("showID", showID).
		Int("currentPage", firstPageResult.CurrentPage).
		Int("totalPages", firstPageResult.TotalPages).
		Int("subtitles", len(firstPageResult.Subtitles)).
		Msg("Fetched first page")

	// If only one page, return early
	if firstPageResult.TotalPages <= 1 {
		return buildSubtitleCollection(firstPageResult.Subtitles), nil
	}

	// Fetch remaining pages in parallel (2 at a time)
	const batchSize = 2
	var allSubtitles = make([]models.Subtitle, 0, len(firstPageResult.Subtitles)*firstPageResult.TotalPages)
	allSubtitles = append(allSubtitles, firstPageResult.Subtitles...)

	for page := 2; page <= firstPageResult.TotalPages; page += batchSize {
		// Determine which pages to fetch in this batch
		endPage := page + batchSize - 1
		if endPage > firstPageResult.TotalPages {
			endPage = firstPageResult.TotalPages
		}

		pageNumbers := make([]int, 0)
		for p := page; p <= endPage; p++ {
			pageNumbers = append(pageNumbers, p)
		}

		logger.Debug().Ints("pages", pageNumbers).Int("showID", showID).Msg("Fetching batch of pages in parallel")

		// Fetch pages in parallel
		type pageResult struct {
			pageNum   int
			subtitles []models.Subtitle
			err       error
		}

		results := make([]pageResult, len(pageNumbers))
		var wg sync.WaitGroup
		wg.Add(len(pageNumbers))

		for i, pageNum := range pageNumbers {
			i, pageNum := i, pageNum
			go func() {
				defer wg.Done()

				pageEndpoint := fmt.Sprintf("%s/index.php?sid=%d&oldal=%d", c.baseURL, showID, pageNum)

				pageReq, err := http.NewRequestWithContext(ctx, "GET", pageEndpoint, nil)
				if err != nil {
					logger.Warn().Err(err).Int("pageNum", pageNum).Int("showID", showID).Msg("Failed to create request for page")
					results[i] = pageResult{pageNum: pageNum, err: fmt.Errorf("failed to create request: %w", err)}
					return
				}
				pageReq.Header.Set("User-Agent", config.GetUserAgent())

				pageResp, err := c.httpClient.Do(pageReq)
				if err != nil {
					logger.Warn().Err(err).Int("pageNum", pageNum).Int("showID", showID).Msg("Failed to fetch page")
					results[i] = pageResult{pageNum: pageNum, err: fmt.Errorf("failed to fetch page: %w", err)}
					return
				}
				defer pageResp.Body.Close()

				if pageResp.StatusCode != http.StatusOK {
					logger.Warn().Int("statusCode", pageResp.StatusCode).Int("pageNum", pageNum).Int("showID", showID).Msg("Page returned non-OK status")
					results[i] = pageResult{pageNum: pageNum, err: fmt.Errorf("page returned status %d", pageResp.StatusCode)}
					return
				}

				pageData, err := c.subtitleParser.ParseHtml(pageResp.Body)
				if err != nil {
					logger.Warn().Err(err).Int("pageNum", pageNum).Int("showID", showID).Msg("Failed to parse page")
					results[i] = pageResult{pageNum: pageNum, err: fmt.Errorf("failed to parse page: %w", err)}
					return
				}

				logger.Debug().Int("pageNum", pageNum).Int("showID", showID).Int("subtitles", len(pageData)).Msg("Successfully fetched page")
				results[i] = pageResult{pageNum: pageNum, subtitles: pageData}
			}()
		}

		wg.Wait()

		// Collect results from this batch
		var batchErrors []error
		for _, result := range results {
			if result.err != nil {
				logger.Warn().Err(result.err).Int("pageNum", result.pageNum).Msg("Error fetching page")
				batchErrors = append(batchErrors, result.err)
			} else {
				allSubtitles = append(allSubtitles, result.subtitles...)
			}
		}

		// If any pages in the batch failed, log but continue
		if len(batchErrors) > 0 {
			logger.Warn().Err(errors.Join(batchErrors...)).Int("showID", showID).Msg("Some pages in batch failed, continuing with successful results")
		}
	}

	logger.Info().
		Int("showID", showID).
		Int("totalPages", firstPageResult.TotalPages).
		Int("totalSubtitles", len(allSubtitles)).
		Msg("Successfully fetched all pages")

	return buildSubtitleCollection(allSubtitles), nil
}

// buildSubtitleCollection constructs a SubtitleCollection from subtitles
func buildSubtitleCollection(subtitles []models.Subtitle) *models.SubtitleCollection {
	showName := ""
	if len(subtitles) > 0 {
		showName = subtitles[0].ShowName
	}

	return &models.SubtitleCollection{
		ShowName:  showName,
		Subtitles: subtitles,
		Total:     len(subtitles),
	}
}
