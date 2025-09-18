package client

import (
	"SuperSubtitles/internal/config"
	"SuperSubtitles/internal/models"
	"SuperSubtitles/internal/parser"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client defines the interface for querying the SuperSubtitles website
type Client interface {
	GetShowList(ctx context.Context) ([]models.Show, error)
}

// client implements the Client interface
type client struct {
	httpClient *http.Client
	baseURL    string
	parser     parser.Parser[models.Show]
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

	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Configure proxy if provided
	if cfg.ProxyConnectionString != "" {
		proxyURL, err := url.Parse(cfg.ProxyConnectionString)
		if err != nil {
			// Log error but continue without proxy
			logger := config.GetLogger()
			logger.Warn().Err(err).Str("proxy", cfg.ProxyConnectionString).Msg("Invalid proxy URL, continuing without proxy")
		} else {
			httpClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	return &client{
		httpClient: httpClient,
		baseURL:    cfg.SuperSubtitleDomain,
		parser:     parser.NewShowParser(cfg.SuperSubtitleDomain),
	}
}

// GetShowList retrieves the list of shows from the SuperSubtitles website
func (c *client) GetShowList(ctx context.Context) ([]models.Show, error) {
	logger := config.GetLogger()
	logger.Info().Str("baseURL", c.baseURL).Msg("Fetching show list")

	// Use the correct endpoint for fetching shows
	endpoint := fmt.Sprintf("%s/index.php?sorf=varakozik-subrip", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch shows: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse HTML response
	shows, err := c.parser.ParseHtml(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse shows from HTML: %w", err)
	}

	logger.Info().Int("count", len(shows)).Msg("Successfully fetched shows")
	return shows, nil
}
