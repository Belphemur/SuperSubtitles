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
	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/failsafehttp"
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
	baseTransport      *http.Transport // retained for testing / proxy verification
}

// NewClient creates a new client instance with proxy configuration if provided
func NewClient(cfg *config.Config) Client {
	logger := config.GetLogger()

	// Parse timeout duration
	timeout := 30 * time.Second // default
	if cfg.ClientTimeout != "" {
		if parsedTimeout, err := time.ParseDuration(cfg.ClientTimeout); err != nil {
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
			logger.Warn().Err(err).Str("proxy", cfg.ProxyConnectionString).Msg("Invalid proxy URL, continuing without proxy")
		} else {
			// Override only the Proxy field
			baseTransport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// Build the retry policy using failsafe-go's built-in HTTP retry policy builder.
	// It retries on connection errors, 429 Too Many Requests, and 5xx server errors
	// (except 501 Not Implemented). Context cancellation aborts retries immediately.
	maxAttempts := cfg.Retry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // default: 3 total attempts (2 retries)
	}
	retryBuilder := failsafehttp.NewRetryPolicyBuilder().
		WithMaxAttempts(maxAttempts).
		OnRetry(func(e failsafe.ExecutionEvent[*http.Response]) {
			lastErr := e.LastError()
			lastResult := e.LastResult()
			logEvent := logger.Warn().Int("attempt", e.Attempts())
			if lastErr != nil {
				logEvent = logEvent.Err(lastErr)
			}
			if lastResult != nil {
				logEvent = logEvent.Int("status", lastResult.StatusCode)
			}
			logEvent.Msg("Retrying HTTP request")
		})

	if cfg.Retry.InitialDelay != "" {
		initialDelay, err := time.ParseDuration(cfg.Retry.InitialDelay)
		if err != nil {
			logger.Warn().Err(err).Str("initial_delay", cfg.Retry.InitialDelay).Msg("Invalid retry initial delay, using no delay")
		} else {
			maxDelay := initialDelay
			if cfg.Retry.MaxDelay != "" {
				if parsedMax, err := time.ParseDuration(cfg.Retry.MaxDelay); err != nil {
					logger.Warn().Err(err).Str("max_delay", cfg.Retry.MaxDelay).Msg("Invalid retry max delay, using initial delay as max")
				} else {
					maxDelay = parsedMax
				}
			}
			retryBuilder = retryBuilder.WithBackoff(initialDelay, maxDelay)
		}
	}

	retryPolicy := retryBuilder.Build()

	// Wrap transport with compression support (gzip, brotli, zstd), then wrap the
	// compression transport with the failsafe retry round-tripper so that every
	// HTTP call made through httpClient is automatically retried on transient failures.
	resilientTransport := failsafehttp.NewRoundTripper(newCompressionTransport(baseTransport), retryPolicy)

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: resilientTransport,
	}

	return &client{
		httpClient:         httpClient,
		baseURL:            cfg.SuperSubtitleDomain,
		showParser:         parser.NewShowParser(cfg.SuperSubtitleDomain),
		thirdPartyParser:   parser.NewThirdPartyIdParser(),
		subtitleDownloader: services.NewSubtitleDownloader(httpClient),
		subtitleParser:     parser.NewSubtitleParser(cfg.SuperSubtitleDomain),
		baseTransport:      baseTransport,
	}
}

// Close releases any resources held by the client, such as cache connections.
func (c *client) Close() error {
	return c.subtitleDownloader.Close()
}
