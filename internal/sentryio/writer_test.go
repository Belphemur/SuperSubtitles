package sentryio

import (
	"errors"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

func TestSentryWriter_AddsBreadcrumbOnError(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	reporter, err := New(Config{
		DSN:          testDSN,
		FlushTimeout: time.Second,
		Transport:    transport,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	writer := &SentryWriter{reporter: reporter}
	logger := zerolog.New(writer)

	// Emit an error log — it should create a breadcrumb.
	logger.Error().Err(errors.New("boom")).Str("show_id", "999").Msg("Failed to get subtitles")

	// Capture an exception so the event carries accumulated breadcrumbs.
	reporter.CaptureException(errors.New("trigger"), nil)

	if len(transport.events) != 1 {
		t.Fatalf("event count = %d, want 1", len(transport.events))
	}

	event := transport.events[0]
	if len(event.Breadcrumbs) == 0 {
		t.Fatal("expected at least one breadcrumb on the captured event")
	}

	bc := event.Breadcrumbs[0]
	if bc.Message != "Failed to get subtitles" {
		t.Fatalf("breadcrumb message = %q, want %q", bc.Message, "Failed to get subtitles")
	}
	if bc.Level != sentry.LevelError {
		t.Fatalf("breadcrumb level = %q, want %q", bc.Level, sentry.LevelError)
	}
	if bc.Category != "log" {
		t.Fatalf("breadcrumb category = %q, want %q", bc.Category, "log")
	}
	if got, ok := bc.Data["show_id"]; !ok || got != "999" {
		t.Fatalf("breadcrumb data show_id = %v, want %q", got, "999")
	}
	if got, ok := bc.Data["error"]; !ok || got != "boom" {
		t.Fatalf("breadcrumb data error = %v, want %q", got, "boom")
	}
}

func TestSentryWriter_AddsBreadcrumbOnInfo(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	reporter, err := New(Config{
		DSN:          testDSN,
		FlushTimeout: time.Second,
		Transport:    transport,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	writer := &SentryWriter{reporter: reporter}
	logger := zerolog.New(writer)

	// Info-level logs should also produce breadcrumbs.
	logger.Info().Str("address", ":8080").Msg("Server started")

	reporter.CaptureException(errors.New("trigger"), nil)

	if len(transport.events) != 1 {
		t.Fatalf("event count = %d, want 1", len(transport.events))
	}
	if len(transport.events[0].Breadcrumbs) == 0 {
		t.Fatal("expected breadcrumb for info-level log")
	}

	bc := transport.events[0].Breadcrumbs[0]
	if bc.Level != sentry.LevelInfo {
		t.Fatalf("breadcrumb level = %q, want %q", bc.Level, sentry.LevelInfo)
	}
	if bc.Message != "Server started" {
		t.Fatalf("breadcrumb message = %q, want %q", bc.Message, "Server started")
	}
}

func TestSentryWriter_MultipleBreadcrumbsAccumulate(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	reporter, err := New(Config{
		DSN:          testDSN,
		FlushTimeout: time.Second,
		Transport:    transport,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	writer := &SentryWriter{reporter: reporter}
	logger := zerolog.New(writer)

	logger.Info().Msg("step 1")
	logger.Debug().Msg("step 2")
	logger.Warn().Msg("step 3")

	reporter.CaptureException(errors.New("trigger"), nil)

	if len(transport.events) != 1 {
		t.Fatalf("event count = %d, want 1", len(transport.events))
	}
	if got := len(transport.events[0].Breadcrumbs); got != 3 {
		t.Fatalf("breadcrumb count = %d, want 3", got)
	}
}

func TestSentryWriter_DisabledReporterDoesNotCapture(t *testing.T) {
	t.Parallel()

	reporter, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	writer := &SentryWriter{reporter: reporter}
	logger := zerolog.New(writer)

	// Should not panic or capture anything.
	logger.Error().Err(errors.New("boom")).Msg("this should not be captured")
}

func TestSentryWriter_NumericFieldsInBreadcrumb(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	reporter, err := New(Config{
		DSN:          testDSN,
		FlushTimeout: time.Second,
		Transport:    transport,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	writer := &SentryWriter{reporter: reporter}
	logger := zerolog.New(writer)

	logger.Error().
		Err(errors.New("network failure")).
		Int64("content_id", 12345).
		Int("show_count", 5).
		Msg("Failed to check for updates")

	reporter.CaptureException(errors.New("trigger"), nil)

	if len(transport.events) != 1 {
		t.Fatalf("event count = %d, want 1", len(transport.events))
	}
	bc := transport.events[0].Breadcrumbs[0]
	if got, ok := bc.Data["content_id"].(float64); !ok || got != 12345 {
		t.Fatalf("breadcrumb data content_id = %v, want 12345", bc.Data["content_id"])
	}
	if got, ok := bc.Data["show_count"].(float64); !ok || got != 5 {
		t.Fatalf("breadcrumb data show_count = %v, want 5", bc.Data["show_count"])
	}
}

func TestSentryWriter_InvalidJSONDoesNotPanic(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	reporter, err := New(Config{
		DSN:          testDSN,
		FlushTimeout: time.Second,
		Transport:    transport,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	writer := &SentryWriter{reporter: reporter}
	n, writeErr := writer.WriteLevel(zerolog.ErrorLevel, []byte("not valid json"))
	if writeErr != nil {
		t.Fatalf("WriteLevel error = %v", writeErr)
	}
	if n != len("not valid json") {
		t.Fatalf("n = %d, want %d", n, len("not valid json"))
	}

	// No breadcrumbs should have been added.
	reporter.CaptureException(errors.New("trigger"), nil)
	if len(transport.events[0].Breadcrumbs) != 0 {
		t.Fatal("expected no breadcrumbs for invalid JSON")
	}
}

func TestMapBreadcrumbLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   zerolog.Level
		want sentry.Level
	}{
		{zerolog.TraceLevel, sentry.LevelDebug},
		{zerolog.DebugLevel, sentry.LevelDebug},
		{zerolog.InfoLevel, sentry.LevelInfo},
		{zerolog.WarnLevel, sentry.LevelWarning},
		{zerolog.ErrorLevel, sentry.LevelError},
		{zerolog.FatalLevel, sentry.LevelFatal},
		{zerolog.PanicLevel, sentry.LevelFatal},
	}
	for _, tt := range tests {
		t.Run(tt.in.String(), func(t *testing.T) {
			t.Parallel()
			if got := mapBreadcrumbLevel(tt.in); got != tt.want {
				t.Fatalf("mapBreadcrumbLevel(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSentryWriter_WriteFallbackIsNoOp(t *testing.T) {
	t.Parallel()

	writer := NewWriter()
	data := []byte(`{"level":"error","message":"test"}`)
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if n != len(data) {
		t.Fatalf("n = %d, want %d", n, len(data))
	}
}
