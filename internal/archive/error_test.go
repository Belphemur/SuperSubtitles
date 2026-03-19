package archive

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestArchiveError_Error(t *testing.T) {
	t.Parallel()

	t.Run("nil archive error", func(t *testing.T) {
		t.Parallel()
		var err *ArchiveError
		if got := err.Error(); got != "" {
			t.Errorf("Error() = %q, want empty string", got)
		}
	})

	t.Run("message and cause", func(t *testing.T) {
		t.Parallel()
		err := &ArchiveError{Message: "failed to extract archive", Err: errors.New("zip bomb detected")}
		if got, want := err.Error(), "failed to extract archive: zip bomb detected"; got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("message only", func(t *testing.T) {
		t.Parallel()
		err := &ArchiveError{Message: "unsupported archive format"}
		if got, want := err.Error(), "unsupported archive format"; got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("url only", func(t *testing.T) {
		t.Parallel()
		err := &ArchiveError{URL: "https://example.com/sub.zip"}
		if got, want := err.Error(), "url: https://example.com/sub.zip"; got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("cause only", func(t *testing.T) {
		t.Parallel()
		err := &ArchiveError{Err: errors.New("io failure")}
		if got, want := err.Error(), "io failure"; got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("url message and cause", func(t *testing.T) {
		t.Parallel()
		err := &ArchiveError{Message: "failed to sanitize", URL: "https://example.com/sub.zip", Err: errors.New("corrupt header")}
		if got, want := err.Error(), "failed to sanitize (url: https://example.com/sub.zip): corrupt header"; got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestArchiveError_IsAndUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("nil unwrap", func(t *testing.T) {
		t.Parallel()
		var err *ArchiveError
		if got := err.Unwrap(); got != nil {
			t.Errorf("Unwrap() = %v, want nil", got)
		}
	})

	cause := errors.New("corrupt entry")
	err := NewError("failed archive operation", cause)

	if !errors.Is(err, &ArchiveError{}) {
		t.Error("expected errors.Is to match *ArchiveError")
	}

	if !errors.Is(err, cause) {
		t.Error("expected errors.Is to match wrapped cause")
	}

	wrapped := fmt.Errorf("download failed: %w", err)
	if !errors.Is(wrapped, &ArchiveError{}) {
		t.Error("expected errors.Is to match *ArchiveError through wrapping")
	}
}

func TestArchiveError_GRPCBinding(t *testing.T) {
	t.Parallel()

	t.Run("recoverable maps to failed precondition", func(t *testing.T) {
		t.Parallel()
		err := &ArchiveError{Message: "archive validation failed"}

		if got, want := err.GRPCCode(), codes.FailedPrecondition; got != want {
			t.Errorf("GRPCCode() = %v, want %v", got, want)
		}

		if got, want := err.HTTPStatusCode(), http.StatusUnprocessableEntity; got != want {
			t.Errorf("HTTPStatusCode() = %d, want %d", got, want)
		}
	})

	t.Run("unrecoverable maps to data loss", func(t *testing.T) {
		t.Parallel()
		err := NewUnrecoverableError("archive is permanently unusable", errors.New("zip bomb detected"))

		if got, want := err.GRPCCode(), codes.DataLoss; got != want {
			t.Errorf("GRPCCode() = %v, want %v", got, want)
		}

		if got, want := err.HTTPStatusCode(), http.StatusUnprocessableEntity; got != want {
			t.Errorf("HTTPStatusCode() = %d, want %d", got, want)
		}
	})
}

func TestArchiveError_Constructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		err           *ArchiveError
		wantCode      codes.Code
		wantHasURL    bool
		wantMessage   string
		wantInnerText string
	}{
		{
			name:          "recoverable without URL",
			err:           NewError("failed to sanitize archive", errors.New("bad zip header")),
			wantCode:      codes.FailedPrecondition,
			wantHasURL:    false,
			wantMessage:   "failed to sanitize archive",
			wantInnerText: "bad zip header",
		},
		{
			name:          "recoverable with URL",
			err:           NewErrorWithURL("failed to sanitize archive", "https://example.com/file.zip", errors.New("bad zip header")),
			wantCode:      codes.FailedPrecondition,
			wantHasURL:    true,
			wantMessage:   "failed to sanitize archive",
			wantInnerText: "bad zip header",
		},
		{
			name:          "unrecoverable without URL",
			err:           NewUnrecoverableError("ZIP bomb detected", errors.New("suspicious compression ratio")),
			wantCode:      codes.DataLoss,
			wantHasURL:    false,
			wantMessage:   "ZIP bomb detected",
			wantInnerText: "suspicious compression ratio",
		},
		{
			name:          "unrecoverable with URL",
			err:           NewUnrecoverableErrorWithURL("ZIP bomb detected", "https://example.com/file.zip", errors.New("suspicious compression ratio")),
			wantCode:      codes.DataLoss,
			wantHasURL:    true,
			wantMessage:   "ZIP bomb detected",
			wantInnerText: "suspicious compression ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err == nil {
				t.Fatal("expected constructor to return non-nil error")
			}
			if tt.err.GRPCCode() != tt.wantCode {
				t.Fatalf("GRPCCode() = %v, want %v", tt.err.GRPCCode(), tt.wantCode)
			}
			if (tt.err.URL != "") != tt.wantHasURL {
				t.Fatalf("URL presence mismatch: got %q", tt.err.URL)
			}
			if tt.err.Message != tt.wantMessage {
				t.Fatalf("Message = %q, want %q", tt.err.Message, tt.wantMessage)
			}
			if tt.err.Err == nil || tt.err.Err.Error() != tt.wantInnerText {
				t.Fatalf("inner error = %v, want %q", tt.err.Err, tt.wantInnerText)
			}
		})
	}
}
