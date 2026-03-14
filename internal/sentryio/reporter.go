package sentryio

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"github.com/getsentry/sentry-go"
)

const defaultFlushTimeout = 2 * time.Second

// Config controls optional Sentry reporting.
type Config struct {
	DSN          string
	Environment  string
	Debug        bool
	FlushTimeout time.Duration
	Transport    sentry.Transport
}

// Reporter captures application errors to Sentry when configured.
type Reporter struct {
	enabled      bool
	flushTimeout time.Duration
	hub          *sentry.Hub
}

var globalReporter = &Reporter{flushTimeout: defaultFlushTimeout}

// New creates a reporter backed by the official sentry-go SDK.
func New(cfg Config) (*Reporter, error) {
	flushTimeout := cfg.FlushTimeout
	if flushTimeout <= 0 {
		flushTimeout = defaultFlushTimeout
	}

	if strings.TrimSpace(cfg.DSN) == "" {
		return &Reporter{flushTimeout: flushTimeout}, nil
	}

	clientOptions := sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Debug:            cfg.Debug,
		AttachStacktrace: true,
		EnableLogs:       true,
	}
	if cfg.Transport != nil {
		clientOptions.Transport = cfg.Transport
	}

	client, err := sentry.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		enabled:      true,
		flushTimeout: flushTimeout,
		hub:          sentry.NewHub(client, sentry.NewScope()),
	}, nil
}

// SetGlobal replaces the package-level reporter.
func SetGlobal(reporter *Reporter) {
	if reporter == nil {
		globalReporter = &Reporter{flushTimeout: defaultFlushTimeout}
		return
	}
	globalReporter = reporter
}

// Enabled reports whether Sentry delivery is enabled.
func Enabled() bool {
	return globalReporter.Enabled()
}

// Enabled reports whether Sentry delivery is enabled.
func (r *Reporter) Enabled() bool {
	return r != nil && r.enabled && r.hub != nil
}

// CaptureException reports an error to Sentry when enabled and reportable.
func CaptureException(err error, configureScope func(*sentry.Scope)) bool {
	return globalReporter.CaptureException(err, configureScope)
}

// CaptureException reports an error to Sentry when enabled and reportable.
func (r *Reporter) CaptureException(err error, configureScope func(*sentry.Scope)) bool {
	if !r.Enabled() || !shouldReport(err) {
		return false
	}

	if configureScope == nil {
		r.hub.CaptureException(err)
		return true
	}

	r.hub.WithScope(func(scope *sentry.Scope) {
		configureScope(scope)
		r.hub.CaptureException(err)
	})
	return true
}

// Flush ensures queued events are sent before process shutdown.
func Flush() bool {
	return globalReporter.Flush()
}

// Flush ensures queued events are sent before process shutdown.
func (r *Reporter) Flush() bool {
	if !r.Enabled() {
		return true
	}
	return r.hub.Flush(r.flushTimeout)
}

func shouldReport(err error) bool {
	return err != nil &&
		!errors.Is(err, &apperrors.ErrSubtitleNotFoundInArchive{}) &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded)
}
