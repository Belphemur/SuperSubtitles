package models

import "strings"

// Quality represents the video quality of a subtitle
type Quality int

const (
	QualityUnknown Quality = iota
	Quality360p
	Quality480p
	Quality720p
	Quality1080p
	Quality2160p // 4K
)

// String returns the string representation of the quality
func (q Quality) String() string {
	switch q {
	case Quality360p:
		return "360p"
	case Quality480p:
		return "480p"
	case Quality720p:
		return "720p"
	case Quality1080p:
		return "1080p"
	case Quality2160p:
		return "2160p"
	default:
		return "unknown"
	}
}

// ParseQuality converts a quality string to Quality enum
func ParseQuality(qualityStr string) Quality {
	switch strings.ToLower(qualityStr) {
	case "360p":
		return Quality360p
	case "480p":
		return Quality480p
	case "720p":
		return Quality720p
	case "1080p":
		return Quality1080p
	case "2160p":
		return Quality2160p
	default:
		return QualityUnknown
	}
}

// MarshalJSON implements json.Marshaler interface
func (q Quality) MarshalJSON() ([]byte, error) {
	return []byte(`"` + q.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (q *Quality) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	*q = ParseQuality(str)
	return nil
}
