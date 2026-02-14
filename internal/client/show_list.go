package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
)

// GetShowList retrieves the list of shows from the SuperSubtitles website
func (c *client) GetShowList(ctx context.Context) ([]models.Show, error) {
	var shows []models.Show
	var firstErr error
	for result := range c.StreamShowList(ctx) {
		if result.Err != nil {
			if firstErr == nil {
				firstErr = result.Err
			}
			continue
		}
		shows = append(shows, result.Value)
	}
	if len(shows) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return shows, nil
}

// StreamShowList streams shows as they become available from multiple endpoints.
// Shows are deduplicated by ID on the fly. The channel is closed when all endpoints have been processed.
func (c *client) StreamShowList(ctx context.Context) <-chan StreamResult[models.Show] {
	ch := make(chan StreamResult[models.Show])

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

				req, err := http.NewRequestWithContext(ctx, "GET", ep, nil)
				if err != nil {
					logger.Warn().Err(err).Str("endpoint", ep).Msg("Failed to create request")
					errsMu.Lock()
					endpointErrors = append(endpointErrors, err)
					errsMu.Unlock()
					return
				}
				req.Header.Set("User-Agent", config.GetUserAgent())

				resp, err := c.httpClient.Do(req)
				if err != nil {
					logger.Warn().Err(err).Str("endpoint", ep).Msg("Failed to fetch endpoint")
					errsMu.Lock()
					endpointErrors = append(endpointErrors, err)
					errsMu.Unlock()
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					err := fmt.Errorf("endpoint returned status %d", resp.StatusCode)
					logger.Warn().Err(err).Str("endpoint", ep).Msg("Failed to fetch endpoint")
					errsMu.Lock()
					endpointErrors = append(endpointErrors, err)
					errsMu.Unlock()
					return
				}

				shows, err := c.parser.ParseHtml(resp.Body)
				if err != nil {
					logger.Warn().Err(err).Str("endpoint", ep).Msg("Failed to parse endpoint")
					errsMu.Lock()
					endpointErrors = append(endpointErrors, err)
					errsMu.Unlock()
					return
				}

				// Stream shows as they arrive, deduplicate by ID
				for _, s := range shows {
					if _, exists := seen.LoadOrStore(s.ID, struct{}{}); exists {
						continue
					}
					select {
					case ch <- StreamResult[models.Show]{Value: s}:
						atomic.AddInt64(&sentShows, 1)
					case <-ctx.Done():
						return
					}
				}
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
			case ch <- StreamResult[models.Show]{Err: fmt.Errorf("all show list endpoints failed: %v", errors.Join(errs...))}:
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
