package models

import (
	"strconv"
	"strings"
	"time"
)

// languageMap maps SuperSubtitles language names to ISO 639-1 codes
var languageMap = map[string]string{
	"Magyar":             "hu", // Hungarian
	"Angol":              "en", // English
	"Albán":              "sq", // Albanian
	"Arab":               "ar", // Arabic
	"Bolgár":             "bg", // Bulgarian
	"Brazíliai portugál": "pt", // Brazilian Portuguese (using pt for Portuguese)
	"Cseh":               "cs", // Czech
	"Dán":                "da", // Danish
	"Finn":               "fi", // Finnish
	"Flamand":            "nl", // Flemish (using nl for Dutch as Flemish is a variant)
	"Francia":            "fr", // French
	"Görög":              "el", // Greek
	"Héber":              "he", // Hebrew
	"Holland":            "nl", // Dutch
	"Horvát":             "hr", // Croatian
	"Koreai":             "ko", // Korean
	"Lengyel":            "pl", // Polish
	"Német":              "de", // German
	"Norvég":             "no", // Norwegian
	"Olasz":              "it", // Italian
	"Orosz":              "ru", // Russian
	"Portugál":           "pt", // Portuguese
	"Román":              "ro", // Romanian
	"Spanyol":            "es", // Spanish
	"Svéd":               "sv", // Swedish
	"Szerb":              "sr", // Serbian
	"Szlovén":            "sl", // Slovenian
	"Szlovák":            "sk", // Slovak
	"Török":              "tr", // Turkish
}

// SuperSubtitle represents the raw subtitle data from SuperSubtitles API response
type SuperSubtitle struct {
	Language     string `json:"language"`       // Language of the subtitle (e.g., "Angol", "Magyar")
	Name         string `json:"nev"`            // Name/title with episode/season info
	BaseLink     string `json:"baselink"`       // Base URL for the subtitle service
	Filename     string `json:"fnev"`           // Filename of the subtitle
	SubtitleID   string `json:"felirat"`        // Timestamp ID for the subtitle
	Season       string `json:"evad"`           // Season number ("-1" for season packs)
	Episode      string `json:"ep"`             // Episode number ("-1" for season packs)
	Uploader     string `json:"feltolto"`       // Name of the uploader
	ExactMatch   string `json:"pontos_talalat"` // Exact match score
	IsSeasonPack string `json:"evadpakk"`       // Whether it's a season pack ("1" or "0")
}

// SuperSubtitleResponse represents the complete API response from SuperSubtitles
// The keys are string IDs and values are SuperSubtitle objects
type SuperSubtitleResponse map[string]SuperSubtitle

// Subtitle represents a normalized subtitle in our application
type Subtitle struct {
	ID           string    `json:"id"`
	ShowName     string    `json:"showName"`
	Language     string    `json:"language"`
	Season       int       `json:"season"`
	Episode      int       `json:"episode"`
	Filename     string    `json:"filename"`
	DownloadURL  string    `json:"downloadUrl"`
	Uploader     string    `json:"uploader"`
	UploadedAt   time.Time `json:"uploadedAt"`
	Quality      Quality   `json:"quality"`      // Video quality enum
	ReleaseGroup string    `json:"releaseGroup"` // Original name from API
	Source       string    `json:"source"`       // Original name from API
	IsSeasonPack bool      `json:"isSeasonPack"`
	ExactMatch   int       `json:"exactMatch"` // Converted exact match score
}

// SubtitleCollection represents a collection of subtitles for a show
type SubtitleCollection struct {
	ShowName  string     `json:"showName"`
	Subtitles []Subtitle `json:"subtitles"`
	Total     int        `json:"total"`
}

// GetSeasonNumber converts the season string to an integer
func (s *SuperSubtitle) GetSeasonNumber() int {
	if s.Season == "-1" {
		return -1 // Indicates season pack or unknown
	}
	season, err := strconv.Atoi(s.Season)
	if err != nil {
		return 0 // Default to 0 if conversion fails
	}
	return season
}

// GetEpisodeNumber converts the episode string to an integer
func (s *SuperSubtitle) GetEpisodeNumber() int {
	if s.Episode == "-1" {
		return -1 // Indicates season pack or unknown
	}
	episode, err := strconv.Atoi(s.Episode)
	if err != nil {
		return 0 // Default to 0 if conversion fails
	}
	return episode
}

// IsSeasonPackBool returns whether this is a season pack as a boolean
func (s *SuperSubtitle) IsSeasonPackBool() bool {
	return s.IsSeasonPack == "1"
}

// GetExactMatchScore converts the exact match string to an integer
func (s *SuperSubtitle) GetExactMatchScore() int {
	score, err := strconv.Atoi(s.ExactMatch)
	if err != nil {
		return 0
	}
	return score
}

// GetUploadTime converts the subtitle ID (timestamp) to a time.Time
func (s *SuperSubtitle) GetUploadTime() time.Time {
	timestamp, err := strconv.ParseInt(s.SubtitleID, 10, 64)
	if err != nil {
		return time.Time{} // Return zero time if conversion fails
	}
	return time.Unix(timestamp, 0)
}

// ExtractQuality attempts to extract quality information from the name
func (s *SuperSubtitle) ExtractQuality() Quality {
	name := strings.ToLower(s.Name)

	// Iterate through all possible quality values to find matches
	// Start from highest quality and work down for preference
	for q := Quality2160p; q >= Quality360p; q-- {
		qualityStr := strings.ToLower(q.String())
		if strings.Contains(name, qualityStr) {
			return q
		}
	}

	return QualityUnknown
}

// GetDownloadURL constructs the download URL for the subtitle
func (s *SuperSubtitle) GetDownloadURL() string {
	return s.BaseLink + "/index.php?action=letolt&felirat=" + s.SubtitleID
}

// GetLanguageISO converts the SuperSubtitles language name to ISO 639-1 code
func (s *SuperSubtitle) GetLanguageISO() string {
	if isoCode, exists := languageMap[s.Language]; exists {
		return isoCode
	}
	// If language is not found in map, return the original language name in lowercase
	return strings.ToLower(s.Language)
}

// ExtractShowName attempts to extract the clean show name
func (s *SuperSubtitle) ExtractShowName() string {
	name := s.Name

	// Remove common patterns to get clean show name
	// Split by " - " to get the part before episode info
	if strings.Contains(name, " - ") {
		parts := strings.Split(name, " - ")
		if len(parts) > 0 {
			showPart := parts[0]
			// Remove season info in parentheses
			if idx := strings.Index(showPart, " (Season"); idx != -1 {
				return strings.TrimSpace(showPart[:idx])
			}
			return strings.TrimSpace(showPart)
		}
	}

	// If no " - " found, look for (Season pattern
	if idx := strings.Index(name, " (Season"); idx != -1 {
		return strings.TrimSpace(name[:idx])
	}

	return name
}

// ToSubtitle converts a SuperSubtitle to our normalized Subtitle format
func (s *SuperSubtitle) ToSubtitle() Subtitle {
	return Subtitle{
		ID:           s.SubtitleID,
		ShowName:     s.ExtractShowName(),
		Language:     s.GetLanguageISO(),
		Season:       s.GetSeasonNumber(),
		Episode:      s.GetEpisodeNumber(),
		Filename:     s.Filename,
		DownloadURL:  s.GetDownloadURL(),
		Uploader:     s.Uploader,
		UploadedAt:   s.GetUploadTime(),
		Quality:      s.ExtractQuality(),
		ReleaseGroup: s.Name, // Keep original name as release group info
		Source:       s.Name, // Keep original name as source info
		IsSeasonPack: s.IsSeasonPackBool(),
		ExactMatch:   s.GetExactMatchScore(),
	}
}

// ConvertResponse converts a SuperSubtitleResponse to a SubtitleCollection
func ConvertResponse(response SuperSubtitleResponse) SubtitleCollection {
	subtitles := make([]Subtitle, 0, len(response))
	var showName string

	// Convert each SuperSubtitle to Subtitle
	for _, superSub := range response {
		subtitle := superSub.ToSubtitle()
		subtitles = append(subtitles, subtitle)

		// Use the first subtitle's show name for the collection
		if showName == "" {
			showName = subtitle.ShowName
		}
	}

	return SubtitleCollection{
		ShowName:  showName,
		Subtitles: subtitles,
		Total:     len(subtitles),
	}
}
