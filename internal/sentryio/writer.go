package sentryio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// SentryWriter is a zerolog LevelWriter that parses JSON-serialized log events
// and automatically adds them as Sentry breadcrumbs and structured logs. It
// follows the same pattern as zerolog's ConsoleWriter: receive the serialized
// JSON produced by zerolog, parse it, and re-emit it in a different format.
//
// Breadcrumbs are attached to the next captured Sentry event, giving error
// reports a trail of recent application activity. Structured logs are forwarded
// independently via Sentry's Logger API.
//
// Error capture remains explicit via [CaptureException]; this writer only
// provides automatic log/breadcrumb context.
type SentryWriter struct {
	// reporter overrides the package-level global when non-nil.
	reporter *Reporter
	// sentryCtx and sentryLogger are cached to avoid per-event allocation.
	sentryCtx    context.Context
	sentryLogger sentry.Logger
}

// NewWriter creates a SentryWriter backed by the package-level global Reporter.
func NewWriter() *SentryWriter {
	w := &SentryWriter{}
	w.initLogger()
	return w
}

// Write implements io.Writer. It is a no-op fallback; zerolog uses WriteLevel
// when the writer satisfies the LevelWriter interface.
func (w *SentryWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// WriteLevel implements zerolog.LevelWriter. Every log event is recorded as a
// Sentry breadcrumb and forwarded as a structured Sentry log entry.
func (w *SentryWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	r := w.getReporter()
	if !r.Enabled() {
		return len(p), nil
	}

	fields, parseErr := parseLogJSON(p)
	if parseErr != nil {
		return len(p), nil // never break the logging pipeline
	}

	msg, _ := fields[zerolog.MessageFieldName].(string)

	// Build a data map from all non-standard zerolog fields.
	data := make(map[string]any)
	for k, v := range fields {
		switch k {
		case zerolog.TimestampFieldName, zerolog.LevelFieldName,
			zerolog.MessageFieldName:
			continue
		default:
			data[k] = v
		}
	}

	// Record a breadcrumb so the log entry appears in the next captured event.
	r.hub.AddBreadcrumb(&sentry.Breadcrumb{
		Type:      "default",
		Category:  "log",
		Message:   msg,
		Data:      data,
		Level:     mapBreadcrumbLevel(level),
		Timestamp: time.Now(),
	}, nil)

	// Forward as a structured Sentry log entry using the cached logger.
	if w.sentryLogger != nil {
		emitSentryLog(w.sentryLogger, level, msg, data)
	}

	return len(p), nil
}

// getReporter returns the reporter to use: the explicit one if set, otherwise
// the package-level global.
func (w *SentryWriter) getReporter() *Reporter {
	if w.reporter != nil {
		return w.reporter
	}
	return globalReporter
}

// initLogger caches a sentry.Logger so WriteLevel doesn't allocate one per call.
func (w *SentryWriter) initLogger() {
	r := w.getReporter()
	if !r.Enabled() {
		return
	}
	w.sentryCtx = sentry.SetHubOnContext(context.Background(), r.hub)
	w.sentryLogger = sentry.NewLogger(w.sentryCtx)
}

// parseLogJSON unmarshals the zerolog JSON payload into a generic field map.
func parseLogJSON(p []byte) (map[string]any, error) {
	var fields map[string]any
	if err := json.Unmarshal(p, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// emitSentryLog sends a structured log entry via the Sentry Logger API.
func emitSentryLog(logger sentry.Logger, level zerolog.Level, msg string, data map[string]any) {
	var entry sentry.LogEntry
	switch level {
	case zerolog.TraceLevel:
		entry = logger.Trace()
	case zerolog.DebugLevel:
		entry = logger.Debug()
	case zerolog.InfoLevel:
		entry = logger.Info()
	case zerolog.WarnLevel:
		entry = logger.Warn()
	case zerolog.ErrorLevel:
		entry = logger.Error()
	case zerolog.FatalLevel:
		entry = logger.Fatal()
	case zerolog.PanicLevel:
		entry = logger.Fatal()
	default:
		entry = logger.Info()
	}

	for k, v := range data {
		switch val := v.(type) {
		case string:
			entry = entry.String(k, val)
		case float64:
			if val == float64(int64(val)) {
				entry = entry.Int64(k, int64(val))
			} else {
				entry = entry.Float64(k, val)
			}
		case bool:
			entry = entry.Bool(k, val)
		default:
			entry = entry.String(k, fmt.Sprintf("%v", v))
		}
	}

	entry.Emit(msg)
}

// mapBreadcrumbLevel converts a zerolog level to the equivalent Sentry severity
// used for breadcrumbs.
func mapBreadcrumbLevel(level zerolog.Level) sentry.Level {
	switch level {
	case zerolog.TraceLevel, zerolog.DebugLevel:
		return sentry.LevelDebug
	case zerolog.InfoLevel:
		return sentry.LevelInfo
	case zerolog.WarnLevel:
		return sentry.LevelWarning
	case zerolog.ErrorLevel:
		return sentry.LevelError
	case zerolog.FatalLevel, zerolog.PanicLevel:
		return sentry.LevelFatal
	default:
		return sentry.LevelInfo
	}
}
