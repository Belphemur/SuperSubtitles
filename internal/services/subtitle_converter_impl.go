package services

import (
	"strconv"
	"strings"
	"time"

	"SuperSubtitles/internal/models"
)

// DefaultSubtitleConverter is the default implementation of SubtitleConverter
type DefaultSubtitleConverter struct {
	languageMap map[string]string
}

// NewSubtitleConverter creates a new instance of DefaultSubtitleConverter
func NewSubtitleConverter() SubtitleConverter {
	return &DefaultSubtitleConverter{
		languageMap: map[string]string{
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
		},
	}
}

// ConvertSuperSubtitle converts a single SuperSubtitle to normalized Subtitle
func (c *DefaultSubtitleConverter) ConvertSuperSubtitle(superSub *models.SuperSubtitle) models.Subtitle {
	return models.Subtitle{
		ID:           superSub.SubtitleID,
		ShowName:     c.extractShowName(superSub.Name),
		Language:     c.convertLanguageToISO(superSub.Language),
		Season:       c.convertSeasonNumber(superSub.Season),
		Episode:      c.convertEpisodeNumber(superSub.Episode),
		Filename:     superSub.Filename,
		DownloadURL:  c.buildDownloadURL(superSub.BaseLink, superSub.SubtitleID),
		Uploader:     superSub.Uploader,
		UploadedAt:   c.convertUploadTime(superSub.SubtitleID),
		Quality:      c.extractQuality(superSub.Name),
		ReleaseGroup: superSub.Name, // Keep original name as release group info
		Source:       superSub.Name, // Keep original name as source info
		IsSeasonPack: c.convertIsSeasonPack(superSub.IsSeasonPack),
		ExactMatch:   c.convertExactMatchScore(superSub.ExactMatch),
	}
}

// ConvertResponse converts a SuperSubtitleResponse to a SubtitleCollection
func (c *DefaultSubtitleConverter) ConvertResponse(response models.SuperSubtitleResponse) models.SubtitleCollection {
	subtitles := make([]models.Subtitle, 0, len(response))
	var showName string

	// Convert each SuperSubtitle to Subtitle
	for _, superSub := range response {
		subtitle := c.ConvertSuperSubtitle(&superSub)
		subtitles = append(subtitles, subtitle)

		// Use the first subtitle's show name for the collection
		if showName == "" {
			showName = subtitle.ShowName
		}
	}

	return models.SubtitleCollection{
		ShowName:  showName,
		Subtitles: subtitles,
		Total:     len(subtitles),
	}
}

// extractShowName extracts clean show name from subtitle name
func (c *DefaultSubtitleConverter) extractShowName(name string) string {
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

// extractQuality extracts quality information from subtitle name
func (c *DefaultSubtitleConverter) extractQuality(name string) models.Quality {
	nameLower := strings.ToLower(name)

	// Iterate through all possible quality values to find matches
	// Start from highest quality and work down for preference
	for q := models.Quality2160p; q >= models.Quality360p; q-- {
		qualityStr := strings.ToLower(q.String())
		if strings.Contains(nameLower, qualityStr) {
			return q
		}
	}

	return models.QualityUnknown
}

// convertLanguageToISO converts SuperSubtitles language to ISO code
func (c *DefaultSubtitleConverter) convertLanguageToISO(language string) string {
	if isoCode, exists := c.languageMap[language]; exists {
		return isoCode
	}
	// If language is not found in map, return the original language name in lowercase
	return strings.ToLower(language)
}

// buildDownloadURL constructs the download URL for a subtitle
func (c *DefaultSubtitleConverter) buildDownloadURL(baseLink, subtitleID string) string {
	// Avoid duplicate /index.php in the URL
	if strings.HasSuffix(baseLink, "/index.php") {
		return baseLink + "?action=letolt&felirat=" + subtitleID
	}
	return baseLink + "/index.php?action=letolt&felirat=" + subtitleID
}

// Helper methods for converting SuperSubtitle fields

// convertSeasonNumber converts the season string to an integer
func (c *DefaultSubtitleConverter) convertSeasonNumber(season string) int {
	if season == "-1" {
		return -1 // Indicates season pack or unknown
	}
	seasonNum, err := strconv.Atoi(season)
	if err != nil {
		return 0 // Default to 0 if conversion fails
	}
	return seasonNum
}

// convertEpisodeNumber converts the episode string to an integer
func (c *DefaultSubtitleConverter) convertEpisodeNumber(episode string) int {
	if episode == "-1" {
		return -1 // Indicates season pack or unknown
	}
	episodeNum, err := strconv.Atoi(episode)
	if err != nil {
		return 0 // Default to 0 if conversion fails
	}
	return episodeNum
}

// convertIsSeasonPack returns whether this is a season pack as a boolean
func (c *DefaultSubtitleConverter) convertIsSeasonPack(isSeasonPack string) bool {
	return isSeasonPack == "1"
}

// convertExactMatchScore converts the exact match string to an integer
func (c *DefaultSubtitleConverter) convertExactMatchScore(exactMatch string) int {
	score, err := strconv.Atoi(exactMatch)
	if err != nil {
		return 0
	}
	return score
}

// convertUploadTime converts the subtitle ID (timestamp) to a time.Time
func (c *DefaultSubtitleConverter) convertUploadTime(subtitleID string) time.Time {
	timestamp, err := strconv.ParseInt(subtitleID, 10, 64)
	if err != nil {
		return time.Time{} // Return zero time if conversion fails
	}
	return time.Unix(timestamp, 0)
}
