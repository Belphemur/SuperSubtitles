package client

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"
	"github.com/Belphemur/SuperSubtitles/v2/internal/parser"
	"github.com/Belphemur/SuperSubtitles/v2/internal/services"
)

// Client defines the interface for querying the SuperSubtitles website
type Client interface {
	CheckForUpdates(ctx context.Context, contentID int64) (*models.UpdateCheckResult, error)
	DownloadSubtitle(ctx context.Context, subtitleID string, episode *int) (*models.DownloadResult, error)

	// Streaming methods return channels that emit results as they become available.
	// The channel is closed when all results have been sent.
	// Errors are sent as StreamResult with a non-nil Err field.
	StreamShowList(ctx context.Context) <-chan models.StreamResult[models.Show]
	StreamSubtitles(ctx context.Context, showID int) <-chan models.StreamResult[models.Subtitle]
	StreamShowSubtitles(ctx context.Context, shows []models.Show) <-chan models.StreamResult[models.ShowSubtitles]
	StreamRecentSubtitles(ctx context.Context, sinceID int) <-chan models.StreamResult[models.ShowSubtitles]

	// Close releases any resources held by the client (e.g., cache connections).
	Close() error
}

// client implements the Client interface
type client struct {
	httpClient         *http.Client
	baseURL            string
	showParser         parser.PaginatedParser[models.Show]
	thirdPartyParser   parser.SingleResultParser[models.ThirdPartyIds]
	subtitleDownloader services.SubtitleDownloader
	subtitleParser     *parser.SubtitleParser
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
		httpClient:         httpClient,
		baseURL:            cfg.SuperSubtitleDomain,
		showParser:         parser.NewShowParser(cfg.SuperSubtitleDomain),
		thirdPartyParser:   parser.NewThirdPartyIdParser(),
		subtitleDownloader: services.NewSubtitleDownloader(httpClient),
		subtitleParser:     parser.NewSubtitleParser(cfg.SuperSubtitleDomain),
	}
}

// Close releases any resources held by the client, such as cache connections.
func (c *client) Close() error {
	return c.subtitleDownloader.Close()
}
