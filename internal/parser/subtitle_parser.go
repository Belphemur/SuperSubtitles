package parser

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"

	"github.com/PuerkitoBio/goquery"
)

// Pre-compiled regex patterns for performance
var (
	seasonPackRegex  = regexp.MustCompile(`\(Season\s+(\d+)\)`)
	episodeRegex     = regexp.MustCompile(`(\d+)x(\d+)`)
	odalPageRegex    = regexp.MustCompile(`oldal=(\d+)`)
	parenthesesRegex = regexp.MustCompile(`\s*\([^)]*\)`)
)

// languageToISO maps Hungarian language names to ISO 639-1 codes
// Based on common languages found on feliratok.eu
var languageToISO = map[string]string{
	// Hungarian names (lowercase for case-insensitive matching)
	"magyar":   "hu",
	"angol":    "en",
	"német":    "de",
	"francia":  "fr",
	"spanyol":  "es",
	"olasz":    "it",
	"orosz":    "ru",
	"portugál": "pt",
	"holland":  "nl",
	"lengyel":  "pl",
	"török":    "tr",
	"arab":     "ar",
	"héber":    "he",
	"japán":    "ja",
	"kínai":    "zh",
	"koreai":   "ko",
	"cseh":     "cs",
	"dán":      "da",
	"finn":     "fi",
	"görög":    "el",
	"norvég":   "no",
	"svéd":     "sv",
	"román":    "ro",
	"szerb":    "sr",
	"horvát":   "hr",
	"bolgár":   "bg",
	"ukrán":    "uk",
	"thai":     "th",
	"vietnámi": "vi",
	"indonéz":  "id",
	"hindi":    "hi",
	"perzsa":   "fa",
	"brazil":   "pt", // Brazilian Portuguese maps to pt

	// English names (fallback)
	"hungarian":  "hu",
	"english":    "en",
	"german":     "de",
	"french":     "fr",
	"spanish":    "es",
	"italian":    "it",
	"russian":    "ru",
	"portuguese": "pt",
	"dutch":      "nl",
	"polish":     "pl",
	"turkish":    "tr",
	"arabic":     "ar",
	"hebrew":     "he",
	"japanese":   "ja",
	"chinese":    "zh",
	"korean":     "ko",
	"czech":      "cs",
	"danish":     "da",
	"finnish":    "fi",
	"greek":      "el",
	"norwegian":  "no",
	"swedish":    "sv",
	"romanian":   "ro",
	"serbian":    "sr",
	"croatian":   "hr",
	"bulgarian":  "bg",
	"ukrainian":  "uk",
	"vietnamese": "vi",
	"indonesian": "id",
}

// SubtitleParser implements the Parser interface for parsing HTML subtitle listings
type SubtitleParser struct {
	baseURL string
}

// SubtitlePageResult contains parsed subtitles and pagination information
type SubtitlePageResult struct {
	Subtitles   []models.Subtitle
	CurrentPage int
	TotalPages  int
	HasNextPage bool
}

// NewSubtitleParser creates a new subtitle parser instance
func NewSubtitleParser(baseURL string) *SubtitleParser {
	return &SubtitleParser{
		baseURL: baseURL,
	}
}

// convertLanguageToISO converts a language name (Hungarian or English) to ISO 639-1 code
// Returns the ISO code if found, otherwise returns the original input
func convertLanguageToISO(languageName string) string {
	// Normalize to lowercase and trim
	normalized := strings.ToLower(strings.TrimSpace(languageName))

	if normalized == "" {
		return ""
	}

	// Look up in the map
	if isoCode, exists := languageToISO[normalized]; exists {
		return isoCode
	}

	// If already looks like an ISO code (2-3 letters), return as-is
	if len(normalized) == 2 || len(normalized) == 3 {
		// Could be already an ISO code
		return normalized
	}

	logger := config.GetLogger()
	logger.Debug().
		Str("languageName", languageName).
		Msg("Unknown language name, returning original value")

	// Return original if no mapping found
	return languageName
}

// ParseHtml implements the Parser[models.Subtitle] interface
func (p *SubtitleParser) ParseHtml(body io.Reader) ([]models.Subtitle, error) {
	result, err := p.ParseHtmlWithPagination(body)
	if err != nil {
		return nil, err
	}
	return result.Subtitles, nil
}

// ParseHtmlWithPagination parses HTML and returns both subtitles and pagination info
func (p *SubtitleParser) ParseHtmlWithPagination(body io.Reader) (*SubtitlePageResult, error) {
	logger := config.GetLogger()
	logger.Info().Msg("Starting HTML parsing for subtitles")

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse HTML document")
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	logger.Debug().Msg("HTML document parsed successfully, starting subtitle extraction")

	var subtitles []models.Subtitle

	// Find all table rows that contain subtitle information
	// Structure: <tr><td>Category</td><td>Language</td><td>Description with link</td><td>Uploader</td><td>Date</td><td>Download</td></tr>
	doc.Find("tbody").ChildrenFiltered("tr").Each(func(i int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() < 5 {
			return // Not a subtitle row
		}

		subtitle := p.extractSubtitleFromRow(tds)
		if subtitle != nil {
			subtitles = append(subtitles, *subtitle)
			logger.Debug().
				Str("language", subtitle.Language).
				Str("name", subtitle.Name).
				Int("season", subtitle.Season).
				Int("episode", subtitle.Episode).
				Bool("seasonPack", subtitle.IsSeasonPack).
				Msg("Successfully extracted subtitle")
		}
	})

	// Extract pagination info
	currentPage, totalPages := p.extractPaginationInfo(doc)

	logger.Info().
		Int("total_subtitles", len(subtitles)).
		Int("current_page", currentPage).
		Int("total_pages", totalPages).
		Msg("Completed HTML parsing for subtitles")

	return &SubtitlePageResult{
		Subtitles:   subtitles,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		HasNextPage: currentPage < totalPages,
	}, nil
}

// extractSubtitleFromRow extracts subtitle information from a table row
func (p *SubtitleParser) extractSubtitleFromRow(tds *goquery.Selection) *models.Subtitle {
	logger := config.GetLogger()

	// Expected structure: | Category | Language | Description | Uploader | Date | Download |
	// That's exactly 6 columns

	if tds.Length() < 6 {
		return nil
	}

	// Extract show ID from category column (column 0)
	// The category column contains a link like: <a href="index.php?sid=13051">
	categoryTd := tds.Eq(0)
	showID := p.extractShowIDFromCategory(categoryTd)

	// Extract language from column 1
	language := strings.TrimSpace(tds.Eq(1).Text())
	if language == "" {
		return nil
	}

	// Convert language name to ISO 639-1 code
	languageISO := convertLanguageToISO(language)

	// Extract description (show name, episode, release info) from column 2
	descriptionTd := tds.Eq(2).Find(".eredeti")
	description := strings.TrimSpace(descriptionTd.Text())
	if description == "" {
		return nil
	}

	// Extract download link from column 5 (the last column)
	downloadTd := tds.Eq(5)
	downloadLink, exists := downloadTd.Find("a").Attr("href")
	if !exists {
		logger.Debug().Str("description", description).Msg("No download link found")
		return nil
	}

	// Construct full download URL
	downloadURL := p.constructDownloadURL(downloadLink)
	if downloadURL == "" {
		logger.Debug().Str("downloadLink", downloadLink).Msg("Failed to construct download URL")
		return nil
	}

	// Normalize the URL by decoding it (ensures properly formed URLs)
	downloadURL = p.normalizeDownloadURL(downloadURL)

	// Parse description to extract show name, season, episode, and release info
	showName, season, episode, releaseInfo, isSeasonPack := p.parseDescription(description)

	// Extract qualities and release groups from release info
	qualities, releaseGroups := p.parseReleaseInfo(releaseInfo)

	// Extract uploader from column 3
	uploader := strings.TrimSpace(tds.Eq(3).Text())

	// Extract and parse date from column 4
	dateStr := strings.TrimSpace(tds.Eq(4).Text())
	uploadedAt := p.parseDate(dateStr)

	// Generate ID from download link
	subtitleID := p.extractIDFromDownloadLink(downloadLink)

	// Extract filename from download link
	filename := p.extractFilenameFromDownloadLink(downloadLink)

	// Extract only the episode title from description
	episodeTitle := extractEpisodeTitle(description)

	return &models.Subtitle{
		ID:            subtitleID,
		ShowID:        showID,
		Name:          episodeTitle,
		ShowName:      showName,
		Language:      languageISO,
		Season:        season,
		Episode:       episode,
		Filename:      filename,
		DownloadURL:   downloadURL,
		Uploader:      uploader,
		UploadedAt:    uploadedAt,
		Qualities:     qualities,
		ReleaseGroups: releaseGroups,
		Release:       releaseInfo,
		IsSeasonPack:  isSeasonPack,
	}
}

// extractShowIDFromCategory extracts the show ID from the category column's link
// Example: <a href="index.php?sid=13051"> or <a href="/index.php?sid=13051">
func (p *SubtitleParser) extractShowIDFromCategory(categoryTd *goquery.Selection) int {
	logger := config.GetLogger()

	// Find the link in the category column
	href, exists := categoryTd.Find("a").Attr("href")
	if !exists {
		return 0
	}

	// Parse URL to extract sid parameter
	parsedURL, err := url.Parse(href)
	if err != nil {
		logger.Debug().Str("href", href).Err(err).Msg("Failed to parse category link")
		return 0
	}

	sidStr := parsedURL.Query().Get("sid")
	if sidStr == "" {
		return 0
	}

	showID, err := strconv.Atoi(sidStr)
	if err != nil {
		logger.Debug().Str("sid", sidStr).Err(err).Msg("Failed to convert sid to integer")
		return 0
	}

	return showID
}

// parseDescription extracts show name, season, episode, release info, and season pack flag
// Example: "Outlander - Az idegen - 7x16 Outlander - 7x16 - A Hundred Thousand Angels (AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab)"
// Example: "- Billy the Kid (Season 2) (WEB.720p-EDITH, AMZN.WEB-DL.720p-FLUX)"
func (p *SubtitleParser) parseDescription(description string) (showName string, season int, episode int, releaseInfo string, isSeasonPack bool) {
	logger := config.GetLogger()

	// Check if it's a season pack by looking for "(Season XX)" pattern
	if matches := seasonPackRegex.FindStringSubmatch(description); len(matches) > 1 {
		isSeasonPack = true
		seasonNum, _ := strconv.Atoi(matches[1])
		season = seasonNum
		episode = -1

		// Extract show name (everything before "(Season")
		parts := strings.Split(description, "(Season")
		if len(parts) > 0 {
			showName = strings.TrimSpace(parts[0])
			// Remove leading dash if present
			showName = strings.TrimPrefix(showName, "- ")
			showName = strings.TrimSpace(showName)
		}

		// Extract release info from last parentheses
		releaseInfo = p.extractReleaseInfo(description)

		logger.Debug().
			Str("description", description).
			Str("showName", showName).
			Int("season", season).
			Bool("isSeasonPack", isSeasonPack).
			Msg("Parsed season pack description")
		return
	}

	// Parse regular episode format: "ShowName - SxEE Episode Title (release info)"
	// The season/episode pattern is SxEE (e.g., 7x16)
	if matches := episodeRegex.FindStringSubmatch(description); len(matches) > 2 {
		seasonNum, _ := strconv.Atoi(matches[1])
		episodeNum, _ := strconv.Atoi(matches[2])
		season = seasonNum
		episode = episodeNum

		// Extract show name (everything before the first "- Sx")
		if idx := strings.Index(description, fmt.Sprintf("- %dx", season)); idx != -1 {
			showName = strings.TrimSpace(description[:idx])
			// Remove leading dash if present
			showName = strings.TrimPrefix(showName, "- ")
			showName = strings.TrimSpace(showName)
		} else {
			// Fallback: take everything before first dash
			parts := strings.SplitN(description, "-", 2)
			if len(parts) > 0 {
				showName = strings.TrimSpace(parts[0])
			}
		}

		// Extract release info from last parentheses
		releaseInfo = p.extractReleaseInfo(description)

		logger.Debug().
			Str("description", description).
			Str("showName", showName).
			Int("season", season).
			Int("episode", episode).
			Msg("Parsed episode description")
		return
	}

	// Fallback: couldn't parse episode info
	logger.Debug().Str("description", description).Msg("Failed to parse season/episode from description")
	showName = description
	season = -1
	episode = -1
	return
}

// extractReleaseInfo extracts the release info from the last parentheses in a description string
func (p *SubtitleParser) extractReleaseInfo(description string) string {
	idx := strings.LastIndex(description, "(")
	if idx == -1 {
		return ""
	}

	endIdx := strings.Index(description[idx:], ")")
	if endIdx == -1 {
		return ""
	}

	return description[idx+1 : idx+endIdx]
}

// parseReleaseInfo extracts qualities and multiple release groups from release info string
// Example: "AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab"
func (p *SubtitleParser) parseReleaseInfo(releaseInfo string) (qualities []models.Quality, releaseGroups []string) {
	if releaseInfo == "" {
		return nil, nil
	}

	releaseGroups = make([]string, 0)
	qualities = make([]models.Quality, 0)
	seenQualities := make(map[models.Quality]struct{})

	// Split by comma to get individual releases
	releases := strings.Split(releaseInfo, ",")

	for _, release := range releases {
		release = strings.TrimSpace(release)
		if release == "" {
			continue
		}

		// Extract release group (after the last dash)
		if idx := strings.LastIndex(release, "-"); idx != -1 {
			group := strings.TrimSpace(release[idx+1:])
			if group != "" {
				releaseGroups = append(releaseGroups, group)
			}
		}
		// Detect quality from each release, keep unique qualities
		detectedQuality := p.detectQuality(release)
		if detectedQuality != models.QualityUnknown {
			if _, exists := seenQualities[detectedQuality]; !exists {
				qualities = append(qualities, detectedQuality)
				seenQualities[detectedQuality] = struct{}{}
			}
		}
	}

	return qualities, releaseGroups
}

// detectQuality detects video quality from a release string
func (p *SubtitleParser) detectQuality(release string) models.Quality {
	lowerRelease := strings.ToLower(release)
	switch {
	case strings.Contains(lowerRelease, "2160p") || strings.Contains(lowerRelease, "4k"):
		return models.Quality2160p
	case strings.Contains(lowerRelease, "1080p"):
		return models.Quality1080p
	case strings.Contains(lowerRelease, "720p"):
		return models.Quality720p
	case strings.Contains(lowerRelease, "480p"):
		return models.Quality480p
	case strings.Contains(lowerRelease, "360p"):
		return models.Quality360p
	default:
		return models.QualityUnknown
	}
}

// parseDate parses a date string in the format "YYYY-MM-DD"
func (p *SubtitleParser) parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// Try parsing in YYYY-MM-DD format
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		logger := config.GetLogger()
		logger.Debug().Str("dateStr", dateStr).Err(err).Msg("Failed to parse date")
		return time.Time{}
	}

	return t
}

// constructDownloadURL constructs the full download URL from a relative link
func (p *SubtitleParser) constructDownloadURL(link string) string {
	if link == "" {
		return ""
	}

	// If it's already a full URL, return as-is
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}

	// If it starts with /, just prepend base URL
	if strings.HasPrefix(link, "/") {
		return p.baseURL + link
	}

	// Otherwise, it's a relative link
	return p.baseURL + "/" + link
}

// normalizeDownloadURL ensures the download URL is properly decoded and normalized
// It parses the URL and reconstructs it with properly decoded query parameters
func (p *SubtitleParser) normalizeDownloadURL(downloadURL string) string {
	logger := config.GetLogger()

	// Parse the URL
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		logger.Debug().Str("url", downloadURL).Err(err).Msg("Failed to parse download URL for normalization")
		return downloadURL // Return original if parsing fails
	}

	// Parse query parameters - this ensures any encoded characters are decoded
	queryParams := parsedURL.Query()

	// Reconstruct the URL with normalized query string
	// The Query().Encode() will properly re-encode any special characters
	normalizedURL := &url.URL{
		Scheme:   parsedURL.Scheme,
		User:     parsedURL.User,
		Host:     parsedURL.Host,
		Path:     parsedURL.Path,
		RawPath:  parsedURL.RawPath,
		RawQuery: queryParams.Encode(),
		Fragment: parsedURL.Fragment,
	}

	return normalizedURL.String()
}

// extractIDFromDownloadLink extracts a unique ID from the download link
func (p *SubtitleParser) extractIDFromDownloadLink(link string) int {
	// Parse the URL to extract query parameters
	parsedURL, err := url.Parse(link)
	if err == nil && parsedURL.RawQuery != "" {
		queryParams := parsedURL.Query()

		// Check for felirat parameter (most common in download links)
		if felirat := queryParams.Get("felirat"); felirat != "" {
			if id, err := strconv.Atoi(felirat); err == nil {
				return id
			}
		}

		// Check for feliratid parameter (sometimes used)
		if feliratid := queryParams.Get("feliratid"); feliratid != "" {
			if id, err := strconv.Atoi(feliratid); err == nil {
				return id
			}
		}

		// Check for generic id parameter
		if id := queryParams.Get("id"); id != "" {
			if idNum, err := strconv.Atoi(id); err == nil {
				return idNum
			}
		}
	}

	// Fallback: try to extract numeric ID from path segments
	patterns := []string{
		`/(\d+)(?:/|$)`, // Matches /123/ or /123 at end of string
		`(\d+)\.`,       // Matches digits before extension in filename
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(link); len(matches) > 1 {
			if id, err := strconv.Atoi(matches[1]); err == nil {
				return id
			}
		}
	}

	// Last resort: log and return a sentinel invalid ID (-1)
	logger := config.GetLogger()
	logger.Debug().Str("link", link).Msg("Failed to extract ID from download link; returning invalid ID sentinel")
	return -1
}

// extractFilenameFromDownloadLink extracts the filename from the fnev parameter in the download link
func (p *SubtitleParser) extractFilenameFromDownloadLink(link string) string {
	logger := config.GetLogger()

	// Look for fnev parameter in the URL
	re := regexp.MustCompile(`fnev=([^&]+)`)
	matches := re.FindStringSubmatch(link)
	if len(matches) > 1 {
		// URL decode the filename
		filename, err := url.QueryUnescape(matches[1])
		if err != nil {
			logger.Debug().Str("rawFilename", matches[1]).Err(err).Msg("Failed to unescape filename")
			return matches[1] // Return raw value if decoding fails
		}
		return filename
	}

	return ""
}

// removeParentheticalContent removes all content within parentheses from a string
// and trims any trailing whitespace or punctuation
func removeParentheticalContent(text string) string {
	// Remove all content within parentheses
	result := parenthesesRegex.ReplaceAllString(text, "")

	// Trim whitespace and trailing punctuation
	result = strings.TrimSpace(result)
	result = strings.TrimRight(result, ".-")
	result = strings.TrimSpace(result)

	return result
}

// extractEpisodeTitle extracts only the episode title from a subtitle description
// Example: "Outlander - Az idegen - 7x16 Outlander - 7x16 - A Hundred Thousand Angels (AMZN...)" -> "A Hundred Thousand Angels"
// Example: "Billy the Kid (Season 2) (WEB...)" -> "" (season packs have no episode title)
// Example: "Show - 2x05 - Title With - Many - Dashes (Release)" -> "Title With - Many - Dashes"
func extractEpisodeTitle(description string) string {
	if description == "" {
		return ""
	}

	// Check if this is a season pack (contains "(Season X)")
	if seasonPackRegex.MatchString(description) {
		// Season packs don't have episode titles, only season info
		return ""
	}

	// First, remove content within parentheses
	withoutParens := parenthesesRegex.ReplaceAllString(description, "")
	withoutParens = strings.TrimSpace(withoutParens)

	if withoutParens == "" {
		return ""
	}

	// Check for season/episode pattern (SxEE like 7x16, 1x01, etc)
	allMatches := episodeRegex.FindAllStringIndex(withoutParens, -1)
	if len(allMatches) > 0 {
		// Use the last match, as it's most likely to precede the episode title
		matches := allMatches[len(allMatches)-1]

		// Found season/episode pattern at position [start, end)
		// Look for the dash that comes after the SxEE pattern
		afterSxEE := withoutParens[matches[1]:]
		dashIdx := strings.Index(afterSxEE, "-")
		if dashIdx == -1 {
			// No dash after SxEE, return everything after SxEE
			episodeTitle := strings.TrimSpace(afterSxEE)
			return episodeTitle
		}

		// Take everything after the first dash following SxEE
		episodeTitle := strings.TrimSpace(afterSxEE[dashIdx+1:])
		return episodeTitle
	}

	// If no season/episode pattern found, return the whole string (without parentheses),
	// but trim any trailing punctuation/dashes to avoid titles like "Show Name -".
	clean := strings.TrimRight(withoutParens, ".- ")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return ""
	}
	return clean
}

// extractPaginationInfo extracts current page and total pages from the document
func (p *SubtitleParser) extractPaginationInfo(doc *goquery.Document) (currentPage int, totalPages int) {
	logger := config.GetLogger()

	// Default values
	currentPage = 1

	// In feliratok.eu, pagination looks like "1 2 3" where the current page is usually just text
	// and other pages are links. We look for links with "oldal=" to find the max page.
	maxPage := 1
	doc.Find("a").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		// Look for oldal parameter (page parameter in Hungarian)
		if strings.Contains(href, "oldal=") {
			if matches := odalPageRegex.FindStringSubmatch(href); len(matches) > 1 {
				pageNum, _ := strconv.Atoi(matches[1])
				if pageNum > maxPage {
					maxPage = pageNum
				}
			}
		}
	})

	totalPages = maxPage

	logger.Debug().
		Int("currentPage", currentPage).
		Int("totalPages", totalPages).
		Msg("Extracted pagination info")

	return currentPage, totalPages
}
