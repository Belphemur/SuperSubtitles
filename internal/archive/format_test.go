package archive

import (
	"testing"
)

func TestIsZipFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Standard ZIP magic number",
			content:  []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Empty ZIP magic number",
			content:  []byte{0x50, 0x4B, 0x05, 0x06, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Spanned ZIP magic number",
			content:  []byte{0x50, 0x4B, 0x07, 0x08, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Not a ZIP file - gzip",
			content:  []byte{0x1F, 0x8B, 0x08, 0x00},
			expected: false,
		},
		{
			name:     "Not a ZIP file - random data",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			expected: false,
		},
		{
			name:     "Too short",
			content:  []byte{0x50, 0x4B},
			expected: false,
		},
		{
			name:     "Empty",
			content:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsZipFile(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for content %v", tt.expected, result, tt.content)
			}
		})
	}
}

func TestIsRarFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{name: "rar4 signature", content: []byte{'R', 'a', 'r', '!', 0x1A, 0x07, 0x00}, want: true},
		{name: "rar5 signature", content: []byte{'R', 'a', 'r', '!', 0x1A, 0x07, 0x01, 0x00}, want: true},
		{name: "zip signature", content: []byte{0x50, 0x4B, 0x03, 0x04}, want: false},
		{name: "too short", content: []byte{'R', 'a', 'r'}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsRarFile(tt.content)
			if got != tt.want {
				t.Errorf("IsRarFile(%v) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestIsZipContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{name: "application/zip", contentType: "application/zip", want: true},
		{name: "application/zip with charset", contentType: "application/zip; charset=utf-8", want: true},
		{name: "application/x-zip-compressed", contentType: "application/x-zip-compressed", want: true},
		{name: "Application/ZIP (uppercase)", contentType: "Application/ZIP", want: true},
		{name: "application/gzip - should NOT match", contentType: "application/gzip", want: false},
		{name: "application/x-gzip - should NOT match", contentType: "application/x-gzip", want: false},
		{name: "text/plain", contentType: "text/plain", want: false},
		{name: "application/octet-stream", contentType: "application/octet-stream", want: false},
		{name: "application/vnd.rar is not zip", contentType: "application/vnd.rar", want: false},
		{name: "not zip", contentType: "application/x-subrip", want: false},
		{name: "parse error and not zip", contentType: "not a valid type", want: false},
		{name: "empty string", contentType: "", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsZipContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("IsZipContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestIsRarContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{name: "application/vnd.rar", contentType: "application/vnd.rar", want: true},
		{name: "application/x-rar-compressed", contentType: "application/x-rar-compressed", want: true},
		{name: "application/x-rar", contentType: "application/x-rar", want: true},
		{name: "malformed but trimmed rar value", contentType: " Application/X-RAR ", want: true},
		{name: "invalid media type", contentType: "not a valid type", want: false},
		{name: "application/zip", contentType: "application/zip", want: false},
		{name: "text/plain", contentType: "text/plain", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsRarContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("IsRarContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		content     []byte
		contentType string
		want        string
	}{
		{name: "rar magic wins", content: []byte{'R', 'a', 'r', '!', 0x1A, 0x07, 0x00}, contentType: "application/zip", want: FormatRAR},
		{name: "zip magic wins", content: []byte{0x50, 0x4B, 0x03, 0x04}, contentType: "application/vnd.rar", want: FormatZIP},
		{name: "rar content type fallback", content: []byte("not an archive"), contentType: "application/vnd.rar", want: FormatRAR},
		{name: "zip content type fallback", content: []byte("not an archive"), contentType: "application/zip", want: FormatZIP},
		{name: "unknown format", content: []byte("plain subtitle"), contentType: "text/plain", want: FormatUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetectFormat(tt.content, tt.contentType); got != tt.want {
				t.Errorf("DetectFormat(%v, %q) = %q, want %q", tt.content, tt.contentType, got, tt.want)
			}
		})
	}
}

func TestNormalizeContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		format      string
		want        string
	}{
		{name: "zip canonicalized", contentType: "application/x-zip-compressed", format: FormatZIP, want: "application/zip"},
		{name: "rar canonicalized", contentType: "application/x-rar", format: FormatRAR, want: "application/vnd.rar"},
		{name: "unknown format preserved", contentType: "text/plain", format: FormatUnknown, want: "text/plain"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeContentType(tt.contentType, tt.format); got != tt.want {
				t.Errorf("NormalizeContentType(%q, %q) = %q, want %q", tt.contentType, tt.format, got, tt.want)
			}
		})
	}
}

func TestExtensionForContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{name: "application zip", contentType: "application/zip", want: ".zip"},
		{name: "zip alias", contentType: "application/x-zip-compressed", want: ".zip"},
		{name: "rar canonical", contentType: "application/vnd.rar", want: ".rar"},
		{name: "rar alias", contentType: "application/x-rar-compressed", want: ".rar"},
		{name: "subrip", contentType: "application/x-subrip", want: ".srt"},
		{name: "generic srt fallback", contentType: "text/srt", want: ".srt"},
		{name: "ass", contentType: "application/x-ass", want: ".ass"},
		{name: "webvtt", contentType: "text/webvtt", want: ".vtt"},
		{name: "sub", contentType: "application/x-sub", want: ".sub"},
		{name: "malformed with params", contentType: "application/x-subrip; bad param=", want: ".srt"},
		{name: "completely invalid type", contentType: "not a valid type at all", want: ".srt"},
		{name: "unknown defaults to srt", contentType: "application/octet-stream", want: ".srt"},
		{name: "gzip is not zip", contentType: "application/gzip", want: ".srt"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ExtensionForContentType(tt.contentType); got != tt.want {
				t.Errorf("ExtensionForContentType(%q) = %q, want %q", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestContentTypeForFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "srt", filename: "episode.srt", want: "application/x-subrip"},
		{name: "ass", filename: "episode.ass", want: "application/x-ass"},
		{name: "vtt", filename: "episode.vtt", want: "text/vtt"},
		{name: "sub", filename: "episode.sub", want: "application/x-sub"},
		{name: "zip", filename: "archive.zip", want: "application/zip"},
		{name: "rar", filename: "archive.rar", want: "application/vnd.rar"},
		{name: "uppercase", filename: "EPISODE.SRT", want: "application/x-subrip"},
		{name: "unknown", filename: "notes.txt", want: "application/octet-stream"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ContentTypeForFilename(tt.filename); got != tt.want {
				t.Errorf("ContentTypeForFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}
