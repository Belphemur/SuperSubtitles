package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
	"github.com/Belphemur/SuperSubtitles/internal/parser"
)

// pageBatchSize controls how many pages are fetched in parallel at once.
const pageBatchSize = 10

// StreamShowList streams shows as they become available from multiple endpoints.
// Shows are deduplicated by ID on the fly. The channel is closed when all endpoints have been processed.
// Paginated endpoints are detected automatically: page 1 is fetched first to discover the total page count,
// then remaining pages are fetched in parallel batches of pageBatchSize.
func (c *client) StreamShowList(ctx context.Context) <-chan models.StreamResult[models.Show] {
	ch := make(chan models.StreamResult[models.Show])

	go func() {
		defer close(ch)
		logger := config.GetLogger()
		logger.Info().Str("baseURL", c.baseURL).Msg("Streaming show list from multiple endpoints in parallel")

		// Endpoints to query in parallel
		endpoints := []string{
			fmt.Sprintf("%s/index.php?sorf=varakozik-subrip", c.baseURL),
			fmt.Sprintf("%s/index.php?sorf=alatt-subrip", c.baseURL),
			fmt.Sprintf("%s/index.php?sorf=nem-all-forditas-alatt", c.baseURL),
		}

		// Thread-safe map to track seen show IDs
		var seen sync.Map

		// Track if we sent any shows and endpoint errors
		var sentShows int64
		var errsMu sync.Mutex
		var endpointErrors []error

		// Run all fetches in parallel and stream results as they arrive
		var wg sync.WaitGroup
		wg.Add(len(endpoints))

		for _, ep := range endpoints {
			ep := ep
			go func() {
				defer wg.Done()
				c.fetchEndpointPages(ctx, ep, &seen, &sentShows, &errsMu, &endpointErrors, ch)
			}()
		}

		// Wait for all endpoints to complete
		wg.Wait()

		// Check final status
		errsMu.Lock()
		errs := endpointErrors
		errsMu.Unlock()

		if atomic.LoadInt64(&sentShows) == 0 && len(errs) == len(endpoints) {
			select {
			case ch <- models.StreamResult[models.Show]{Err: fmt.Errorf("all show list endpoints failed: %v", errors.Join(errs...))}:
			case <-ctx.Done():
			}
		} else if len(errs) > 0 {
			logger.Warn().Err(errors.Join(errs...)).Int("successful_endpoints", len(endpoints)-len(errs)).Msg("Partial success fetching show lists")
		} else if atomic.LoadInt64(&sentShows) > 0 {
			logger.Info().Msg("Successfully fetched show lists from all endpoints")
		}
	}()

	return ch
}

// fetchEndpointPages fetches page 1 of the endpoint, discovers the total page count from
// the pagination HTML, then fetches remaining pages in parallel batches.
func (c *client) fetchEndpointPages(
	ctx context.Context,
	endpoint string,
	seen *sync.Map,
	sentShows *int64,
	errsMu *sync.Mutex,
	endpointErrors *[]error,
	ch chan<- models.StreamResult[models.Show],
) {
	logger := config.GetLogger()

	// Helper to record an endpoint-level error
	recordError := func(err error) {
		errsMu.Lock()
		*endpointErrors = append(*endpointErrors, err)
		errsMu.Unlock()
	}

	// --- Fetch page 1 ---
	bodyBytes, err := c.fetchPage(ctx, endpoint)
	if err != nil {
		logger.Warn().Err(err).Str("endpoint", endpoint).Msg("Failed to fetch first page")
		recordError(err)
		return
	}

	c.streamShowsFromBody(ctx, bodyBytes, seen, sentShows, ch)

	// --- Discover total pages ---
	lastPage := 1
	if showParser, ok := c.parser.(*parser.ShowParser); ok {
		lastPage = showParser.ExtractLastPage(bytes.NewReader(bodyBytes))
	}

	if lastPage <= 1 {
		logger.Debug().Str("endpoint", endpoint).Msg("Single page endpoint, done")
		return
	}

	logger.Info().Str("endpoint", endpoint).Int("totalPages", lastPage).Msg("Paginated endpoint detected, fetching remaining pages in parallel")

	// --- Fetch pages 2..lastPage in parallel batches ---
	for batchStart := 2; batchStart <= lastPage; batchStart += pageBatchSize {
		batchEnd := batchStart + pageBatchSize - 1
		if batchEnd > lastPage {
			batchEnd = lastPage
		}

		var batchWg sync.WaitGroup
		batchWg.Add(batchEnd - batchStart + 1)

		for page := batchStart; page <= batchEnd; page++ {
			pageURL := fmt.Sprintf("%s&oldal=%d", endpoint, page)
			go func() {
				defer batchWg.Done()

				pageBody, err := c.fetchPage(ctx, pageURL)
				if err != nil {
					logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to fetch page")
					return
				}

				c.streamShowsFromBody(ctx, pageBody, seen, sentShows, ch)
			}()
		}

		batchWg.Wait()

		// Check if context was cancelled between batches
		if ctx.Err() != nil {
			return
		}
	}

	logger.Debug().Str("endpoint", endpoint).Int("totalPages", lastPage).Msg("Completed fetching all pages for endpoint")
}

// fetchPage performs an HTTP GET and returns the response body bytes.
func (c *client) fetchPage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", config.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

// streamShowsFromBody parses shows from HTML bytes and sends them to the channel,
// deduplicating by show ID.
func (c *client) streamShowsFromBody(
	ctx context.Context,
	bodyBytes []byte,
	seen *sync.Map,
	sentShows *int64,
	ch chan<- models.StreamResult[models.Show],
) {
	shows, err := c.parser.ParseHtml(bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}

	for _, s := range shows {
		if _, exists := seen.LoadOrStore(s.ID, struct{}{}); exists {
			continue
		}
		select {
		case ch <- models.StreamResult[models.Show]{Value: s}:
			atomic.AddInt64(sentShows, 1)
		case <-ctx.Done():
			return
		}
	}
}
