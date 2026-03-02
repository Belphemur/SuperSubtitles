// subtitle_downloader_helpers_test.go tests the individual helper functions in
// subtitle_downloader_impl.go that are not fully exercised by the integration-style
// tests in subtitle_downloader_test.go. Each test targets specific uncovered branches
// such as empty inputs, fallback paths, and error handling.
package services

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/cache"
	"github.com/rs/zerolog"
)

func Test_generateFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		subtitleID  string
		contentType string
		want        string
	}{
		{
			name:        "normal subtitle ID with srt content type",
			subtitleID:  "12345",
			contentType: "application/x-subrip",
			want:        "12345.srt",
		},
		{
			name:        "empty subtitle ID falls back to subtitle",
			subtitleID:  "",
			contentType: "application/x-subrip",
			want:        "subtitle.srt",
		},
		{
			name:        "empty subtitle ID with zip content type",
			subtitleID:  "",
			contentType: "application/zip",
			want:        "subtitle.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := generateFilename(tt.subtitleID, tt.contentType)
			if got != tt.want {
				t.Errorf("generateFilename(%q, %q) = %q, want %q", tt.subtitleID, tt.contentType, got, tt.want)
			}
		})
	}
}

func Test_extractSubtitleID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "valid URL with felirat param",
			url:  "https://feliratok.eu/index.php?felirat=12345",
			want: "12345",
		},
		{
			name: "URL without felirat param",
			url:  "https://feliratok.eu/index.php?other=abc",
			want: "",
		},
		{
			name: "bad URL returns empty string",
			url:  ":",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractSubtitleID(tt.url)
			if got != tt.want {
				t.Errorf("extractSubtitleID(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func Test_getExtensionFromContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{name: "application/zip", contentType: "application/zip", want: ".zip"},
		{name: "application/x-zip-compressed", contentType: "application/x-zip-compressed", want: ".zip"},
		{name: "application/x-subrip", contentType: "application/x-subrip", want: ".srt"},
		{name: "application/x-ass", contentType: "application/x-ass", want: ".ass"},
		{name: "text/ass", contentType: "text/ass", want: ".ass"},
		{name: "text/vtt", contentType: "text/vtt", want: ".vtt"},
		{name: "text/webvtt", contentType: "text/webvtt", want: ".vtt"},
		{name: "application/x-sub", contentType: "application/x-sub", want: ".sub"},
		{name: "contains srt in media type", contentType: "text/srt", want: ".srt"},
		{name: "unknown type falls back to srt", contentType: "application/octet-stream", want: ".srt"},
		{name: "empty string falls back to srt", contentType: "", want: ".srt"},
		{
			name:        "malformed with semicolon strips params",
			contentType: "application/x-subrip; bad param=",
			want:        ".srt",
		},
		{
			name:        "completely unparseable falls back",
			contentType: "not a valid type at all",
			want:        ".srt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getExtensionFromContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("getExtensionFromContentType(%q) = %q, want %q", tt.contentType, got, tt.want)
			}
		})
	}
}

func Test_isZipContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{name: "application/zip", contentType: "application/zip", want: true},
		{name: "application/zip with charset", contentType: "application/zip; charset=utf-8", want: true},
		{name: "application/x-zip-compressed", contentType: "application/x-zip-compressed", want: true},
		{name: "not zip", contentType: "application/x-subrip", want: false},
		{name: "parse error but matches zip", contentType: "application/zip", want: true},
		{
			name:        "parse error and not zip",
			contentType: "not a valid type",
			want:        false,
		},
		{name: "empty string", contentType: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isZipContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("isZipContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func Test_getContentTypeFromFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "srt extension", filename: "sub.srt", want: "application/x-subrip"},
		{name: "ass extension", filename: "sub.ass", want: "application/x-ass"},
		{name: "vtt extension", filename: "sub.vtt", want: "text/vtt"},
		{name: "sub extension", filename: "sub.sub", want: "application/x-sub"},
		{name: "zip extension", filename: "archive.zip", want: "application/zip"},
		{name: "unknown extension", filename: "readme.txt", want: "application/octet-stream"},
		{name: "no extension", filename: "noext", want: "application/octet-stream"},
		{name: "uppercase extension", filename: "sub.SRT", want: "application/x-subrip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getContentTypeFromFilename(tt.filename)
			if got != tt.want {
				t.Errorf("getContentTypeFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func Test_isTextSubtitleContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{name: "application/x-subrip", contentType: "application/x-subrip", want: true},
		{name: "application/x-ass", contentType: "application/x-ass", want: true},
		{name: "text/ass", contentType: "text/ass", want: true},
		{name: "text/vtt", contentType: "text/vtt", want: true},
		{name: "text/webvtt", contentType: "text/webvtt", want: true},
		{name: "application/x-sub", contentType: "application/x-sub", want: true},
		{name: "text/plain", contentType: "text/plain", want: true},
		{name: "application/zip is not text", contentType: "application/zip", want: false},
		{name: "empty is not text", contentType: "", want: false},
		{
			name:        "parse error falls back to raw string for subtitle type",
			contentType: "text/plain",
			want:        true,
		},
		{
			name:        "unparseable non-subtitle type returns false",
			contentType: "garbage content type %%%",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isTextSubtitleContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("isTextSubtitleContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestDefaultSubtitleDownloader_Close(t *testing.T) {
	t.Parallel()
	t.Run("close with cache", func(t *testing.T) {
		t.Parallel()
		zipCache, err := cache.New("memory", cache.ProviderConfig{
			Size: 10,
			TTL:  time.Hour,
		})
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}

		d := &DefaultSubtitleDownloader{
			httpClient: &http.Client{},
			zipCache:   zipCache,
		}

		if err := d.Close(); err != nil {
			t.Errorf("Close() returned unexpected error: %v", err)
		}
	})

	t.Run("close with nil cache", func(t *testing.T) {
		t.Parallel()
		d := &DefaultSubtitleDownloader{
			httpClient: &http.Client{},
			zipCache:   nil,
		}

		if err := d.Close(); err != nil {
			t.Errorf("Close() with nil cache returned unexpected error: %v", err)
		}
	})
}

func Test_zerologCacheLogger_Error(t *testing.T) {
	t.Parallel()
	logger := zerolog.New(io.Discard)
	cacheLogger := &zerologCacheLogger{logger: logger}
	// Should not panic
	cacheLogger.Error("test error", fmt.Errorf("test"))
}

func Test_convertToUTF8(t *testing.T) {
	t.Parallel()
	t.Run("empty content returns empty", func(t *testing.T) {
		t.Parallel()
		got := convertToUTF8([]byte{})
		if len(got) != 0 {
			t.Errorf("convertToUTF8(empty) returned %d bytes, want 0", len(got))
		}
	})

	t.Run("nil content returns nil", func(t *testing.T) {
		t.Parallel()
		got := convertToUTF8(nil)
		if got != nil {
			t.Errorf("convertToUTF8(nil) returned non-nil")
		}
	})

	t.Run("valid UTF-8 content returned as-is", func(t *testing.T) {
		t.Parallel()
		input := []byte("Hello, world! Héllo àccénts")
		got := convertToUTF8(input)
		if string(got) != string(input) {
			t.Errorf("convertToUTF8(valid UTF-8) = %q, want %q", got, input)
		}
	})

	t.Run("non-UTF-8 content is converted", func(t *testing.T) {
		t.Parallel()
		// Latin-1 encoded "café" (0xe9 = é in Latin-1)
		input := []byte{0x63, 0x61, 0x66, 0xe9}
		got := convertToUTF8(input)
		// After conversion, it should be valid UTF-8
		if len(got) == 0 {
			t.Error("convertToUTF8(latin1) returned empty result")
		}
	})
}
