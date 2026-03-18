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
