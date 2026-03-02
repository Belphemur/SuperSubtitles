// Tests for quality.go — Quality type String(), ParseQuality(), MarshalJSON(), and UnmarshalJSON().
package models

import (
	"encoding/json"
	"testing"
)

func TestQuality_String(t *testing.T) {
	tests := []struct {
		name    string
		quality Quality
		want    string
	}{
		{"unknown", QualityUnknown, "unknown"},
		{"360p", Quality360p, "360p"},
		{"480p", Quality480p, "480p"},
		{"720p", Quality720p, "720p"},
		{"1080p", Quality1080p, "1080p"},
		{"2160p", Quality2160p, "2160p"},
		{"invalid high value", Quality(99), "unknown"},
		{"negative value", Quality(-1), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.quality.String()
			if got != tt.want {
				t.Errorf("Quality(%d).String() = %q, want %q", tt.quality, got, tt.want)
			}
		})
	}
}

func TestParseQuality(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Quality
	}{
		{"360p", "360p", Quality360p},
		{"480p", "480p", Quality480p},
		{"720p", "720p", Quality720p},
		{"1080p", "1080p", Quality1080p},
		{"2160p", "2160p", Quality2160p},
		{"uppercase 720P", "720P", Quality720p},
		{"mixed case 1080P", "1080P", Quality1080p},
		{"unknown string", "blah", QualityUnknown},
		{"empty string", "", QualityUnknown},
		{"numeric only", "720", QualityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseQuality(tt.input)
			if got != tt.want {
				t.Errorf("ParseQuality(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuality_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		quality Quality
		want    string
	}{
		{"720p", Quality720p, `"720p"`},
		{"1080p", Quality1080p, `"1080p"`},
		{"2160p", Quality2160p, `"2160p"`},
		{"unknown", QualityUnknown, `"unknown"`},
		{"invalid", Quality(42), `"unknown"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.quality)
			if err != nil {
				t.Fatalf("MarshalJSON() unexpected error: %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", data, tt.want)
			}
		})
	}
}

func TestQuality_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Quality
	}{
		{"360p", `"360p"`, Quality360p},
		{"480p", `"480p"`, Quality480p},
		{"720p", `"720p"`, Quality720p},
		{"1080p", `"1080p"`, Quality1080p},
		{"2160p", `"2160p"`, Quality2160p},
		{"unknown string", `"foobar"`, QualityUnknown},
		{"empty string", `""`, QualityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q Quality
			if err := json.Unmarshal([]byte(tt.input), &q); err != nil {
				t.Fatalf("UnmarshalJSON(%s) unexpected error: %v", tt.input, err)
			}
			if q != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %d, want %d", tt.input, q, tt.want)
			}
		})
	}
}

func TestQuality_JSONRoundTrip(t *testing.T) {
	qualities := []Quality{
		QualityUnknown,
		Quality360p,
		Quality480p,
		Quality720p,
		Quality1080p,
		Quality2160p,
	}

	for _, original := range qualities {
		t.Run(original.String(), func(t *testing.T) {
			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal() unexpected error: %v", err)
			}

			var decoded Quality
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal(%s) unexpected error: %v", data, err)
			}

			if decoded != original {
				t.Errorf("roundtrip failed: original=%d, decoded=%d (json=%s)", original, decoded, data)
			}
		})
	}
}
