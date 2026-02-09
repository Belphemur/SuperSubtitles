package services

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/models"
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
	qualities := c.extractQualities(superSub.Name)
	releaseGroups := c.extractReleaseGroups(superSub.Name)

	return models.Subtitle{
		ID:            superSub.SubtitleID,
		Name:          superSub.Name,
		ShowName:      c.extractShowName(superSub.Name),
		Language:      c.convertLanguageToISO(superSub.Language),
		Season:        c.convertSeasonNumber(superSub.Season),
		Episode:       c.convertEpisodeNumber(superSub.Episode),
		DownloadURL:   c.buildDownloadURL(superSub.BaseLink, superSub.SubtitleID),
		Uploader:      superSub.Uploader,
		UploadedAt:    c.convertUploadTime(superSub.SubtitleID),
		Qualities:     qualities,
		Release:       superSub.Name, // Keep original name as release info
		ReleaseGroups: releaseGroups,
		IsSeasonPack:  c.convertIsSeasonPack(superSub.IsSeasonPack),
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

// extractQualities extracts all quality values from subtitle name, in order of appearance
func (c *DefaultSubtitleConverter) extractQualities(name string) []models.Quality {
	if name == "" {
		return nil
	}

	qualityRegex := regexp.MustCompile(`(?i)(2160p|4k|1080p|720p|480p|360p)`)
	matches := qualityRegex.FindAllStringSubmatch(name, -1)
	if len(matches) == 0 {
		return nil
	}

	qualities := make([]models.Quality, 0, len(matches))
	seen := make(map[models.Quality]struct{})
	for _, match := range matches {
		quality := parseQualityToken(match[1])
		if quality == models.QualityUnknown {
			continue
		}
		if _, exists := seen[quality]; exists {
			continue
		}
		qualities = append(qualities, quality)
		seen[quality] = struct{}{}
	}

	return qualities
}

func parseQualityToken(token string) models.Quality {
	switch strings.ToLower(token) {
	case "2160p", "4k":
		return models.Quality2160p
	case "1080p":
		return models.Quality1080p
	case "720p":
		return models.Quality720p
	case "480p":
		return models.Quality480p
	case "360p":
		return models.Quality360p
	default:
		return models.QualityUnknown
	}
}

// extractReleaseGroups extracts release groups and sources from subtitle name
// Example: "Show (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab)" → ["AMZN", "AMZN.WEB-DL", "FLUX", "SuccessfulCrab", "WEB"]
func (c *DefaultSubtitleConverter) extractReleaseGroups(name string) []string {
	if name == "" {
		return nil
	}

	// Extract content within the LAST pair of parentheses to handle cases like:
	// "Show (Season 1) (WEB.720p-GLHF, AMZN.1080p-PECULATE)"
	endIdx := strings.LastIndex(name, ")")
	if endIdx == -1 {
		return nil
	}

	// Find the opening parenthesis for the last closing parenthesis
	// by searching backwards from endIdx
	startIdx := strings.LastIndex(name[:endIdx], "(")
	if startIdx == -1 {
		return nil
	}

	content := name[startIdx+1 : endIdx]
	releaseGroupSet := make(map[string]struct{})

	// Split by comma first to get individual release patterns
	patterns := strings.Split(content, ",")

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Extract release groups from patterns like:
		// - "AMZN.WEB-DL.720p-FLUX" → AMZN, AMZN.WEB-DL, FLUX
		// - "WEB.1080p-SuccessfulCrab" → WEB, SuccessfulCrab
		// - "720p-RAWR" → RAWR
		// - "SubRip" → (no release group)

		// Find the first quality pattern position
		qualityRegex := regexp.MustCompile(`(?i)(2160p|4k|1080p|720p|480p|360p)`)
		qualityMatch := qualityRegex.FindStringIndex(pattern)

		var sourcePart, groupPart string

		if qualityMatch != nil {
			// Quality pattern found, split before and after
			qualityStart := qualityMatch[0]
			sourcePart = pattern[:qualityStart]

			// Find if there's content after the last quality pattern
			groupPart = pattern[qualityMatch[1]:]
		} else {
			// No quality pattern found, treat entire pattern as potential source
			sourcePart = pattern
			groupPart = ""
		}

		// Clean up source and group parts
		sourcePart = strings.TrimRight(sourcePart, ".-")
		groupPart = strings.TrimLeft(groupPart, ".-")

		// Process source part (e.g., "AMZN.WEB-DL" or "WEB")
		if sourcePart != "" && !isQualityPattern(sourcePart) {
			// Add the full source if it contains dots
			if strings.Contains(sourcePart, ".") {
				releaseGroupSet[sourcePart] = struct{}{}

				// Also extract individual components
				// For "AMZN.WEB-DL", split by dots: ["AMZN", "WEB-DL"]
				dotParts := strings.Split(sourcePart, ".")
				for _, dotPart := range dotParts {
					dotPart = strings.TrimSpace(dotPart)
					if dotPart != "" && !isQualityPattern(dotPart) {
						releaseGroupSet[dotPart] = struct{}{}
					}
				}
			} else if !isQualityPattern(sourcePart) {
				// Simple source like "WEB"
				releaseGroupSet[sourcePart] = struct{}{}
			}
		}

		// Process group part (e.g., "-FLUX" or "-SuccessfulCrab")
		if groupPart != "" {
			// Extract text after dashes
			// For "-FLUX", split by dash: ["", "FLUX"]
			dashParts := strings.Split(groupPart, "-")
			for _, dashPart := range dashParts {
				dashPart = strings.TrimSpace(dashPart)
				if dashPart != "" && !isQualityPattern(dashPart) {
					releaseGroupSet[dashPart] = struct{}{}
				}
			}
		}
	}

	// Convert set to slice
	if len(releaseGroupSet) == 0 {
		return nil
	}

	groups := make([]string, 0, len(releaseGroupSet))
	for group := range releaseGroupSet {
		groups = append(groups, group)
	}

	// Sort for deterministic output
	sort.Strings(groups)
	return groups
}

// isQualityPattern checks if a string is a quality pattern or contains quality indicators
func isQualityPattern(s string) bool {
	lowerS := strings.ToLower(s)

	// Check exact matches for known quality patterns
	qualityPatterns := []string{"2160p", "4k", "1080p", "720p", "480p", "360p", "subrip", "h.264", "h.265", "h264", "h265", "x.264", "x.265", "x264", "x265", "hevc", "avc"}
	for _, pattern := range qualityPatterns {
		if lowerS == pattern {
			return true
		}
	}

	// Check if string contains quality patterns with digits and 'p' (e.g., "1080p", "720p")
	hasQuality := regexp.MustCompile(`\d{3,4}p`)
	return hasQuality.MatchString(lowerS)
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

// convertUploadTime converts the subtitle ID (timestamp) to a time.Time
func (c *DefaultSubtitleConverter) convertUploadTime(subtitleID string) time.Time {
	timestamp, err := strconv.ParseInt(subtitleID, 10, 64)
	if err != nil {
		return time.Time{} // Return zero time if conversion fails
	}
	return time.Unix(timestamp, 0)
}
