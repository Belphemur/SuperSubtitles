package client

import (
	"SuperSubtitles/internal/config"
	"SuperSubtitles/internal/models"
	"SuperSubtitles/internal/parser"
	"SuperSubtitles/internal/services"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client defines the interface for querying the SuperSubtitles website
type Client interface {
	GetShowList(ctx context.Context) ([]models.Show, error)
	GetSubtitles(ctx context.Context, showID int) (*models.SubtitleCollection, error)
	GetShowSubtitles(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error)
	CheckForUpdates(ctx context.Context, contentID string) (*models.UpdateCheckResult, error)
}

// client implements the Client interface
type client struct {
	httpClient        *http.Client
	baseURL           string
	parser            parser.Parser[models.Show]
	subtitleConverter services.SubtitleConverter
	thirdPartyParser  parser.SingleResultParser[models.ThirdPartyIds]
}

// NewClient creates a new client instance with proxy configuration if provided
func NewClient(cfg *config.Config) Client {
	// Parse timeout duration
	timeout := 30 * time.Second // default
	if cfg.ClientTimeout != "" {
		if parsedTimeout, err := time.ParseDuration(cfg.ClientTimeout); err != nil {
			logger := config.GetLogger()
			logger.Warn().Err(err).Str("timeout", cfg.ClientTimeout).Msg("Invalid timeout duration, using default 30s")
		} else {
			timeout = parsedTimeout
		}
	}

	// Set up base transport with optional proxy
	// Clone DefaultTransport to preserve all its settings (timeouts, connection pooling, HTTP/2, etc.)
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()

	if cfg.ProxyConnectionString != "" {
		proxyURL, err := url.Parse(cfg.ProxyConnectionString)
		if err != nil {
			// Log error but continue without proxy
			logger := config.GetLogger()
			logger.Warn().Err(err).Str("proxy", cfg.ProxyConnectionString).Msg("Invalid proxy URL, continuing without proxy")
		} else {
			// Override only the Proxy field
			baseTransport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// Wrap transport with compression support (gzip, brotli, zstd)
	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: newCompressionTransport(baseTransport),
	}

	return &client{
		httpClient:        httpClient,
		baseURL:           cfg.SuperSubtitleDomain,
		parser:            parser.NewShowParser(cfg.SuperSubtitleDomain),
		subtitleConverter: services.NewSubtitleConverter(),
		thirdPartyParser:  parser.NewThirdPartyIdParser(),
	}
}

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
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

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

// GetSubtitles retrieves subtitles for a given show ID from the SuperSubtitles website
func (c *client) GetSubtitles(ctx context.Context, showID int) (*models.SubtitleCollection, error) {
	logger := config.GetLogger()
	logger.Info().Int("showID", showID).Msg("Fetching subtitles for show")

	// Construct the URL for fetching subtitles
	endpoint := fmt.Sprintf("%s/index.php?action=xbmc&sid=%d", c.baseURL, showID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.4472.124 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subtitles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	// If content length is zero, treat as not found
	if resp.ContentLength == 0 {
		return nil, NewNotFoundError("subtitles", showID)
	}

	// Parse JSON response into SuperSubtitleResponse
	var superSubtitleResponse models.SuperSubtitleResponse
	if err := json.NewDecoder(resp.Body).Decode(&superSubtitleResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	// Convert to our normalized subtitle collection
	subtitleCollection := c.subtitleConverter.ConvertResponse(superSubtitleResponse)

	logger.Info().Int("count", subtitleCollection.Total).Str("showName", subtitleCollection.ShowName).Msg("Successfully fetched subtitles")
	return &subtitleCollection, nil
}

// GetShowSubtitles retrieves third-party IDs and subtitle collections for multiple shows in parallel
// Processes shows in batches of 20 to avoid overwhelming the server
func (c *client) GetShowSubtitles(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error) {
	logger := config.GetLogger()
	logger.Info().Int("showCount", len(shows)).Msg("Fetching third-party IDs and subtitles for shows in batches")

	const batchSize = 20
	var allShowSubtitles []models.ShowSubtitles
	var allErrors []error

	// Process shows in batches
	for i := 0; i < len(shows); i += batchSize {
		end := i + batchSize
		if end > len(shows) {
			end = len(shows)
		}

		batch := shows[i:end]
		logger.Info().Int("batchStart", i).Int("batchEnd", end-1).Int("batchSize", len(batch)).Msg("Processing batch of shows")

		batchResults, batchErrors := c.processShowBatch(ctx, batch)

		allShowSubtitles = append(allShowSubtitles, batchResults...)
		allErrors = append(allErrors, batchErrors...)
	}

	if len(allShowSubtitles) == 0 && len(allErrors) > 0 {
		// All shows failed
		return nil, fmt.Errorf("all shows failed processing: %v", errors.Join(allErrors...))
	}

	if len(allErrors) > 0 {
		// Partial success - log aggregated error but still return data
		logger.Warn().Err(errors.Join(allErrors...)).Int("successfulShows", len(allShowSubtitles)).Int("totalShows", len(shows)).Msg("Partial success processing shows")
	} else {
		logger.Info().Int("totalShows", len(allShowSubtitles)).Msg("Successfully processed all shows")
	}

	return allShowSubtitles, nil
}

// processShowBatch processes a batch of shows concurrently (up to batch size)
func (c *client) processShowBatch(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, []error) {
	logger := config.GetLogger()

	type showResult struct {
		showSubtitles models.ShowSubtitles
		err           error
	}

	results := make([]showResult, len(shows))
	var wg sync.WaitGroup
	wg.Add(len(shows))

	for i, show := range shows {
		i, show := i, show // Capture loop variables
		go func() {
			defer wg.Done()

			// Get subtitles for this show
			subtitles, err := c.GetSubtitles(ctx, show.ID)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to fetch subtitles for show")
				results[i] = showResult{err: fmt.Errorf("failed to get subtitles for show %d: %w", show.ID, err)}
				return
			}

			// Find an episode ID from the subtitles (use first subtitle's ID)
			var episodeID string
			if len(subtitles.Subtitles) > 0 {
				episodeID = subtitles.Subtitles[0].ID
			} else {
				logger.Warn().Int("showID", show.ID).Str("showName", show.Name).Msg("No subtitles found, cannot fetch third-party IDs")
				// Create ShowSubtitles without third-party IDs
				results[i] = showResult{
					showSubtitles: models.ShowSubtitles{
						Show:               show,
						ThirdPartyIds:      models.ThirdPartyIds{}, // Empty
						SubtitleCollection: *subtitles,
					},
				}
				return
			}

			// Construct detail page URL
			detailURL := fmt.Sprintf("%s/index.php?tipus=adatlap&azon=a_%s", c.baseURL, episodeID)

			// Fetch detail page HTML
			req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to create detail page request")
				results[i] = showResult{err: fmt.Errorf("failed to create detail page request for show %d: %w", show.ID, err)}
				return
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

			resp, err := c.httpClient.Do(req)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Str("detailURL", detailURL).Msg("Failed to fetch detail page")
				results[i] = showResult{err: fmt.Errorf("failed to fetch detail page for show %d: %w", show.ID, err)}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				logger.Warn().Int("statusCode", resp.StatusCode).Int("showID", show.ID).Str("showName", show.Name).Str("detailURL", detailURL).Msg("Detail page returned non-OK status")
				results[i] = showResult{err: fmt.Errorf("detail page for show %d returned status %d", show.ID, resp.StatusCode)}
				return
			}

			// Parse third-party IDs from HTML
			thirdPartyIds, err := c.thirdPartyParser.ParseHtml(resp.Body)
			if err != nil {
				logger.Warn().Err(err).Int("showID", show.ID).Str("showName", show.Name).Msg("Failed to parse third-party IDs")
				// Don't fail completely, just log and continue with empty third-party IDs
				thirdPartyIds = models.ThirdPartyIds{}
			}

			// Create ShowSubtitles object
			showSubtitles := models.ShowSubtitles{
				Show:               show,
				ThirdPartyIds:      thirdPartyIds,
				SubtitleCollection: *subtitles,
			}

			logger.Debug().Int("showID", show.ID).Str("showName", show.Name).Str("imdbId", thirdPartyIds.IMDBID).Int("tvdbId", thirdPartyIds.TVDBID).Msg("Successfully processed show")
			results[i] = showResult{showSubtitles: showSubtitles}
		}()
	}

	wg.Wait()

	// Collect successful results and errors
	var showSubtitlesList []models.ShowSubtitles
	var errs []error
	for _, result := range results {
		if result.err != nil {
			errs = append(errs, result.err)
		} else {
			showSubtitlesList = append(showSubtitlesList, result.showSubtitles)
		}
	}

	return showSubtitlesList, errs
}

// CheckForUpdates checks if there are any updates available since a specific content ID
func (c *client) CheckForUpdates(ctx context.Context, contentID string) (*models.UpdateCheckResult, error) {
	logger := config.GetLogger()

	// Clean the content ID - remove "a_" prefix if present
	cleanContentID := contentID
	if strings.HasPrefix(contentID, "a_") {
		cleanContentID = strings.TrimPrefix(contentID, "a_")
	}

	logger.Info().Str("contentID", contentID).Str("cleanContentID", cleanContentID).Msg("Checking for updates since content ID")

	// Construct the URL for checking updates
	endpoint := fmt.Sprintf("%s/index.php?action=recheck&azon=%s", c.baseURL, cleanContentID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse JSON response
	var updateResponse models.UpdateCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	// Convert string counts to integers
	filmCount, _ := strconv.Atoi(updateResponse.Film)
	seriesCount, _ := strconv.Atoi(updateResponse.Sorozat)

	result := &models.UpdateCheckResult{
		FilmCount:   filmCount,
		SeriesCount: seriesCount,
		HasUpdates:  filmCount > 0 || seriesCount > 0,
	}

	logger.Info().
		Int("filmCount", filmCount).
		Int("seriesCount", seriesCount).
		Bool("hasUpdates", result.HasUpdates).
		Msg("Successfully checked for updates")

	return result, nil
}
