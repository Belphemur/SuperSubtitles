// Package apperrors tests verify the custom error types (ErrNotFound,
// ErrSubtitleNotFoundInZip, ErrSubtitleResourceNotFound), their Error()
// messages, Is() matching semantics, constructor helpers, and compatibility
// with errors.Is() including through fmt.Errorf wrapping.
package apperrors

import (
	"errors"
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// ErrNotFound
// ---------------------------------------------------------------------------

func TestErrNotFound_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      *ErrNotFound
		expected string
	}{
		{
			name:     "with string ID",
			err:      &ErrNotFound{Resource: "show", ID: "abc"},
			expected: "show with ID abc not found",
		},
		{
			name:     "with int ID",
			err:      &ErrNotFound{Resource: "subtitle", ID: 42},
			expected: "subtitle with ID 42 not found",
		},
		{
			name:     "with nil ID",
			err:      &ErrNotFound{Resource: "show", ID: nil},
			expected: "show not found",
		},
		{
			name:     "with zero int ID",
			err:      &ErrNotFound{Resource: "item", ID: 0},
			expected: "item with ID 0 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrNotFound_Is(t *testing.T) {
	t.Parallel()
	err := &ErrNotFound{Resource: "show", ID: 1}

	t.Run("matches another ErrNotFound", func(t *testing.T) {
		target := &ErrNotFound{}
		if !errors.Is(err, target) {
			t.Error("expected errors.Is to match *ErrNotFound")
		}
	})

	t.Run("matches ErrNotFound with different fields", func(t *testing.T) {
		target := &ErrNotFound{Resource: "other", ID: 99}
		if !errors.Is(err, target) {
			t.Error("expected errors.Is to match *ErrNotFound regardless of field values")
		}
	})

	t.Run("does not match ErrSubtitleNotFoundInZip", func(t *testing.T) {
		target := &ErrSubtitleNotFoundInZip{}
		if errors.Is(err, target) {
			t.Error("expected errors.Is not to match *ErrSubtitleNotFoundInZip")
		}
	})

	t.Run("does not match ErrSubtitleResourceNotFound", func(t *testing.T) {
		target := &ErrSubtitleResourceNotFound{}
		if errors.Is(err, target) {
			t.Error("expected errors.Is not to match *ErrSubtitleResourceNotFound")
		}
	})

	t.Run("does not match plain error", func(t *testing.T) {
		target := errors.New("some error")
		if errors.Is(err, target) {
			t.Error("expected errors.Is not to match a plain error")
		}
	})

	t.Run("matches through fmt.Errorf wrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", err)
		if !errors.Is(wrapped, &ErrNotFound{}) {
			t.Error("expected errors.Is to match *ErrNotFound through wrapping")
		}
	})

	t.Run("matches through double wrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("mid: %w", fmt.Errorf("inner: %w", err))
		if !errors.Is(wrapped, &ErrNotFound{}) {
			t.Error("expected errors.Is to match *ErrNotFound through double wrapping")
		}
	})
}

func TestNewNotFoundError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		resource string
		id       interface{}
		wantMsg  string
	}{
		{
			name:     "string resource and int ID",
			resource: "show",
			id:       7,
			wantMsg:  "show with ID 7 not found",
		},
		{
			name:     "string resource and string ID",
			resource: "episode",
			id:       "s01e02",
			wantMsg:  "episode with ID s01e02 not found",
		},
		{
			name:     "nil ID",
			resource: "subtitle",
			id:       nil,
			wantMsg:  "subtitle not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewNotFoundError(tt.resource, tt.id)
			if err.Resource != tt.resource {
				t.Errorf("Resource = %q, want %q", err.Resource, tt.resource)
			}
			if err.ID != tt.id {
				t.Errorf("ID = %v, want %v", err.ID, tt.id)
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.wantMsg)
			}
			if !errors.Is(err, &ErrNotFound{}) {
				t.Error("expected errors.Is to match *ErrNotFound")
			}
		})
	}
}

func TestNewSubtitlesNotFoundError(t *testing.T) {
	t.Parallel()
	showID := 123
	err := NewSubtitlesNotFoundError(showID)

	if err.Resource != "subtitles" {
		t.Errorf("Resource = %q, want %q", err.Resource, "subtitles")
	}
	if err.ID != showID {
		t.Errorf("ID = %v, want %v", err.ID, showID)
	}

	expectedMsg := "subtitles with ID 123 not found"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedMsg)
	}

	if !errors.Is(err, &ErrNotFound{}) {
		t.Error("expected errors.Is to match *ErrNotFound")
	}
}

// ---------------------------------------------------------------------------
// ErrSubtitleNotFoundInZip
// ---------------------------------------------------------------------------

func TestErrSubtitleNotFoundInZip_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		episode   int
		fileCount int
		expected  string
	}{
		{
			name:      "typical values",
			episode:   5,
			fileCount: 12,
			expected:  "episode 5 not found in season pack ZIP (searched 12 files)",
		},
		{
			name:      "zero values",
			episode:   0,
			fileCount: 0,
			expected:  "episode 0 not found in season pack ZIP (searched 0 files)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := &ErrSubtitleNotFoundInZip{Episode: tt.episode, FileCount: tt.fileCount}
			got := err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrSubtitleNotFoundInZip_Is(t *testing.T) {
	t.Parallel()
	err := &ErrSubtitleNotFoundInZip{Episode: 3, FileCount: 10}

	t.Run("matches another ErrSubtitleNotFoundInZip", func(t *testing.T) {
		target := &ErrSubtitleNotFoundInZip{}
		if !errors.Is(err, target) {
			t.Error("expected errors.Is to match *ErrSubtitleNotFoundInZip")
		}
	})

	t.Run("matches with different fields", func(t *testing.T) {
		target := &ErrSubtitleNotFoundInZip{Episode: 99, FileCount: 50}
		if !errors.Is(err, target) {
			t.Error("expected errors.Is to match *ErrSubtitleNotFoundInZip regardless of field values")
		}
	})

	t.Run("does not match ErrNotFound", func(t *testing.T) {
		target := &ErrNotFound{}
		if errors.Is(err, target) {
			t.Error("expected errors.Is not to match *ErrNotFound")
		}
	})

	t.Run("does not match ErrSubtitleResourceNotFound", func(t *testing.T) {
		target := &ErrSubtitleResourceNotFound{}
		if errors.Is(err, target) {
			t.Error("expected errors.Is not to match *ErrSubtitleResourceNotFound")
		}
	})

	t.Run("does not match plain error", func(t *testing.T) {
		if errors.Is(err, errors.New("other")) {
			t.Error("expected errors.Is not to match a plain error")
		}
	})

	t.Run("matches through fmt.Errorf wrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("download failed: %w", err)
		if !errors.Is(wrapped, &ErrSubtitleNotFoundInZip{}) {
			t.Error("expected errors.Is to match *ErrSubtitleNotFoundInZip through wrapping")
		}
	})

	t.Run("matches through double wrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", err))
		if !errors.Is(wrapped, &ErrSubtitleNotFoundInZip{}) {
			t.Error("expected errors.Is to match *ErrSubtitleNotFoundInZip through double wrapping")
		}
	})
}

// ---------------------------------------------------------------------------
// ErrSubtitleResourceNotFound
// ---------------------------------------------------------------------------

func TestErrSubtitleResourceNotFound_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "typical URL",
			url:      "https://example.com/sub/123",
			expected: "subtitle resource not found at URL: https://example.com/sub/123",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "subtitle resource not found at URL: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := &ErrSubtitleResourceNotFound{URL: tt.url}
			got := err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrSubtitleResourceNotFound_Is(t *testing.T) {
	t.Parallel()
	err := &ErrSubtitleResourceNotFound{URL: "https://example.com/sub/1"}

	t.Run("matches another ErrSubtitleResourceNotFound", func(t *testing.T) {
		target := &ErrSubtitleResourceNotFound{}
		if !errors.Is(err, target) {
			t.Error("expected errors.Is to match *ErrSubtitleResourceNotFound")
		}
	})

	t.Run("matches with different URL", func(t *testing.T) {
		target := &ErrSubtitleResourceNotFound{URL: "https://other.com"}
		if !errors.Is(err, target) {
			t.Error("expected errors.Is to match *ErrSubtitleResourceNotFound regardless of URL")
		}
	})

	t.Run("does not match ErrNotFound", func(t *testing.T) {
		if errors.Is(err, &ErrNotFound{}) {
			t.Error("expected errors.Is not to match *ErrNotFound")
		}
	})

	t.Run("does not match ErrSubtitleNotFoundInZip", func(t *testing.T) {
		if errors.Is(err, &ErrSubtitleNotFoundInZip{}) {
			t.Error("expected errors.Is not to match *ErrSubtitleNotFoundInZip")
		}
	})

	t.Run("does not match plain error", func(t *testing.T) {
		if errors.Is(err, errors.New("other")) {
			t.Error("expected errors.Is not to match a plain error")
		}
	})

	t.Run("matches through fmt.Errorf wrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("fetch failed: %w", err)
		if !errors.Is(wrapped, &ErrSubtitleResourceNotFound{}) {
			t.Error("expected errors.Is to match *ErrSubtitleResourceNotFound through wrapping")
		}
	})

	t.Run("matches through double wrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", err))
		if !errors.Is(wrapped, &ErrSubtitleResourceNotFound{}) {
			t.Error("expected errors.Is to match *ErrSubtitleResourceNotFound through double wrapping")
		}
	})
}

// ---------------------------------------------------------------------------
// Cross-type isolation: no error type matches any other type
// ---------------------------------------------------------------------------

func TestErrorTypes_CrossTypeIsolation(t *testing.T) {
	t.Parallel()
	errs := []error{
		&ErrNotFound{Resource: "x", ID: 1},
		&ErrSubtitleNotFoundInZip{Episode: 1, FileCount: 1},
		&ErrSubtitleResourceNotFound{URL: "http://x"},
	}

	for i, a := range errs {
		for j, b := range errs {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("expected errors.Is(%T, %T) to be false", a, b)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// All types satisfy the error interface
// ---------------------------------------------------------------------------

func TestErrorTypes_ImplementErrorInterface(t *testing.T) {
	t.Parallel()
	var _ error = &ErrNotFound{}
	var _ error = &ErrSubtitleNotFoundInZip{}
	var _ error = &ErrSubtitleResourceNotFound{}
}
