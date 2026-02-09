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

// GetShowList retrieves the list of shows from the SuperSubtitles website
func (c *client) GetShowList(ctx context.Context) ([]models.Show, error) {
	logger := config.GetLogger()
	logger.Info().Str("baseURL", c.baseURL).Msg("Fetching show list from multiple endpoints in parallel")

	// Endpoints to query in parallel. Both have the same table format.
	endpoints := []string{
		fmt.Sprintf("%s/index.php?sorf=varakozik-subrip", c.baseURL),       // pending / waiting
		fmt.Sprintf("%s/index.php?sorf=alatt-subrip", c.baseURL),           // in progress / under
		fmt.Sprintf("%s/index.php?sorf=nem-all-forditas-alatt", c.baseURL), // not all translated / under
	}

	type result struct {
		shows []models.Show
		err   error
	}

	// Worker function for fetching & parsing an endpoint
	fetch := func(ctx context.Context, endpoint string) result {
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return result{nil, fmt.Errorf("create request %s: %w", endpoint, err)}
		}
		req.Header.Set("User-Agent", config.GetUserAgent())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return result{nil, fmt.Errorf("fetch %s: %w", endpoint, err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return result{nil, fmt.Errorf("endpoint %s returned status %d", endpoint, resp.StatusCode)}
		}

		shows, err := c.parser.ParseHtml(resp.Body)
		if err != nil {
			return result{nil, fmt.Errorf("parse %s: %w", endpoint, err)}
		}
		return result{shows, nil}
	}

	// Run all fetches in parallel
	results := make([]result, len(endpoints))
	var wg sync.WaitGroup
	wg.Add(len(endpoints))
	for i, ep := range endpoints {
		i, ep := i, ep
		go func() {
			defer wg.Done()
			results[i] = fetch(ctx, ep)
		}()
	}
	wg.Wait()

	// Merge shows, deduplicating by ID. Preserve first occurrence order.
	merged := make([]models.Show, 0)
	seen := make(map[int]struct{})
	var errs []error
	for idx, r := range results {
		if r.err != nil {
			logger.Warn().Err(r.err).Int("endpoint_index", idx).Msg("Show list endpoint failed")
			errs = append(errs, r.err)
			continue
		}
		for _, s := range r.shows {
			if _, exists := seen[s.ID]; exists {
				continue
			}
			seen[s.ID] = struct{}{}
			merged = append(merged, s)
		}
	}

	// Determine error behavior: if all endpoints failed, return error. If at least one succeeded, return merged list without failing.
	if len(merged) == 0 && len(errs) == len(endpoints) {
		// Aggregate errors
		return nil, fmt.Errorf("all show list endpoints failed: %v", errors.Join(errs...))
	}

	if len(errs) > 0 {
		// Partial success - log aggregated error but still return data
		logger.Warn().Err(errors.Join(errs...)).Int("successful_endpoints", len(endpoints)-len(errs)).Int("total_shows", len(merged)).Msg("Partial success fetching show lists")
	} else {
		logger.Info().Int("total_shows", len(merged)).Msg("Successfully fetched show lists from all endpoints")
	}

	return merged, nil
}
