package sentryio

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"github.com/getsentry/sentry-go"
)

const testDSN = "https://public@example.com/1"

type recordingTransport struct {
	events []*sentry.Event
}

func (t *recordingTransport) Flush(time.Duration) bool {
	return true
}

func (t *recordingTransport) FlushWithContext(context.Context) bool {
	return true
}

func (t *recordingTransport) Configure(sentry.ClientOptions) {}

func (t *recordingTransport) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}

func (t *recordingTransport) Close() {}

func TestReporterCaptureException_DisabledWithoutDSN(t *testing.T) {
	t.Parallel()

	reporter, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if reporter.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
	if reporter.CaptureException(errors.New("boom"), nil) {
		t.Fatal("CaptureException() = true, want false")
	}
}

func TestReporterCaptureException_SendsEvent(t *testing.T) {
	t.Parallel()

	transport := &recordingTransport{}
	reporter, err := New(Config{
		DSN:          testDSN,
		Environment:  "test",
		FlushTimeout: time.Second,
		Transport:    transport,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	sent := reporter.CaptureException(errors.New("boom"), func(scope *sentry.Scope) {
		scope.SetTag("grpc.method", "DownloadSubtitle")
		scope.SetContext("request", map[string]any{"subtitle_id": "101"})
	})
	if !sent {
		t.Fatal("CaptureException() = false, want true")
	}
	if len(transport.events) != 1 {
		t.Fatalf("event count = %d, want 1", len(transport.events))
	}

	event := transport.events[0]
	if len(event.Exception) == 0 {
		t.Fatal("expected captured exception details")
	}
	if event.Tags["grpc.method"] != "DownloadSubtitle" {
		t.Fatalf("grpc.method = %q, want %q", event.Tags["grpc.method"], "DownloadSubtitle")
	}

	requestContext, ok := event.Contexts["request"]
	if !ok {
		t.Fatal("expected request context to be set")
	}
	if got := requestContext["subtitle_id"]; got != "101" {
		t.Fatalf("request.subtitle_id = %v, want %q", got, "101")
	}
}

func TestReporterCaptureException_FiltersContextCanceled(t *testing.T) {
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

	// Direct context.Canceled
	if reporter.CaptureException(context.Canceled, nil) {
		t.Fatal("CaptureException(context.Canceled) = true, want false")
	}

	// Wrapped context.Canceled
	wrapped := fmt.Errorf("download interrupted: %w", context.Canceled)
	if reporter.CaptureException(wrapped, nil) {
		t.Fatal("CaptureException(wrapped context.Canceled) = true, want false")
	}

	if len(transport.events) != 0 {
		t.Fatalf("event count = %d, want 0", len(transport.events))
	}
}

func TestReporterCaptureException_FiltersContextDeadlineExceeded(t *testing.T) {
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

	// Direct context.DeadlineExceeded
	if reporter.CaptureException(context.DeadlineExceeded, nil) {
		t.Fatal("CaptureException(context.DeadlineExceeded) = true, want false")
	}

	// Wrapped context.DeadlineExceeded
	wrapped := fmt.Errorf("request timed out: %w", context.DeadlineExceeded)
	if reporter.CaptureException(wrapped, nil) {
		t.Fatal("CaptureException(wrapped context.DeadlineExceeded) = true, want false")
	}

	if len(transport.events) != 0 {
		t.Fatalf("event count = %d, want 0", len(transport.events))
	}
}

func TestReporterCaptureException_FiltersArchiveNotFound(t *testing.T) {
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

	err = fmt.Errorf("download failed: %w", &apperrors.ErrSubtitleNotFoundInArchive{
		Episode:   5,
		FileCount: 3,
	})
	if reporter.CaptureException(err, nil) {
		t.Fatal("CaptureException() = true, want false")
	}
	if len(transport.events) != 0 {
		t.Fatalf("event count = %d, want 0", len(transport.events))
	}
}
