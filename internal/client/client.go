package client

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"
	"github.com/Belphemur/SuperSubtitles/internal/parser"
	"github.com/Belphemur/SuperSubtitles/internal/services"
)

// Client defines the interface for querying the SuperSubtitles website
type Client interface {
	GetShowList(ctx context.Context) ([]models.Show, error)
	GetSubtitles(ctx context.Context, showID int) (*models.SubtitleCollection, error)
	GetShowSubtitles(ctx context.Context, shows []models.Show) ([]models.ShowSubtitles, error)
	CheckForUpdates(ctx context.Context, contentID string) (*models.UpdateCheckResult, error)
	DownloadSubtitle(ctx context.Context, downloadURL string, req models.DownloadRequest) (*models.DownloadResult, error)
	GetRecentSubtitles(ctx context.Context, sinceID string) ([]models.ShowSubtitles, error)
}

// client implements the Client interface
type client struct {
	httpClient         *http.Client
	baseURL            string
	parser             parser.Parser[models.Show]
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
		parser:             parser.NewShowParser(cfg.SuperSubtitleDomain),
		thirdPartyParser:   parser.NewThirdPartyIdParser(),
		subtitleDownloader: services.NewSubtitleDownloader(httpClient),
		subtitleParser:     parser.NewSubtitleParser(cfg.SuperSubtitleDomain),
	}
}
