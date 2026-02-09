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
	seasonPackRegex = regexp.MustCompile(`\(Season\s+(\d+)\)`)
	episodeRegex    = regexp.MustCompile(`(\d+)x(\d+)`)
	odalPageRegex   = regexp.MustCompile(`oldal=(\d+)`)
)

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

	// Extract language from column 1
	language := strings.TrimSpace(tds.Eq(1).Text())
	if language == "" {
		return nil
	}

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

	return &models.Subtitle{
		ID:            subtitleID,
		Name:          description,
		ShowName:      showName,
		Language:      language,
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

// extractIDFromDownloadLink extracts a unique ID from the download link
func (p *SubtitleParser) extractIDFromDownloadLink(link string) string {
	// Parse the URL to extract query parameters
	parsedURL, err := url.Parse(link)
	if err == nil && parsedURL.RawQuery != "" {
		queryParams := parsedURL.Query()

		// Check for felirat parameter (most common in download links)
		if felirat := queryParams.Get("felirat"); felirat != "" {
			return felirat
		}

		// Check for feliratid parameter (sometimes used)
		if feliratid := queryParams.Get("feliratid"); feliratid != "" {
			return feliratid
		}

		// Check for generic id parameter
		if id := queryParams.Get("id"); id != "" {
			return id
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
			return matches[1]
		}
	}

	// Last resort: use the entire link as ID
	return link
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
