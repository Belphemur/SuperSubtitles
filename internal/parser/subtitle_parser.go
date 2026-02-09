package parser

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Belphemur/SuperSubtitles/internal/config"
	"github.com/Belphemur/SuperSubtitles/internal/models"

	"github.com/PuerkitoBio/goquery"
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
	// Structure: <tr><td>Language</td><td>Description with link</td><td>Uploader</td><td>Date</td><td>Download</td></tr>
	doc.Find("tr").Each(func(i int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() < 5 {
			return // Not a subtitle row
		}

		subtitle := p.extractSubtitleFromRow(row, tds)
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
func (p *SubtitleParser) extractSubtitleFromRow(row *goquery.Selection, tds *goquery.Selection) *models.Subtitle {
	logger := config.GetLogger()

	// Expected structure: | Language | Description | Uploader | Date | Download |
	// Indices:              0          1             2          3      4

	if tds.Length() < 5 {
		return nil
	}

	// Extract language from first td
	language := strings.TrimSpace(tds.Eq(0).Text())
	if language == "" {
		return nil
	}

	// Extract description (show name, episode, release info) from second td
	descriptionTd := tds.Eq(1)
	description := strings.TrimSpace(descriptionTd.Text())
	if description == "" {
		return nil
	}

	// Extract download link
	downloadLink, exists := descriptionTd.Find("a").Attr("href")
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

	// Extract quality and release groups from release info
	quality, releaseGroups := p.parseReleaseInfo(releaseInfo)

	// Extract uploader from third td
	uploader := strings.TrimSpace(tds.Eq(2).Text())

	// Extract and parse date from fourth td
	dateStr := strings.TrimSpace(tds.Eq(3).Text())
	uploadedAt := p.parseDate(dateStr)

	// Generate ID from download link
	subtitleID := p.extractIDFromDownloadLink(downloadLink)

	return &models.Subtitle{
		ID:            subtitleID,
		Name:          description,
		ShowName:      showName,
		Language:      language,
		Season:        season,
		Episode:       episode,
		DownloadURL:   downloadURL,
		Uploader:      uploader,
		UploadedAt:    uploadedAt,
		Quality:       quality,
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
	seasonPackRegex := regexp.MustCompile(`\(Season\s+(\d+)\)`)
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

		// Extract release info (everything in the last parentheses)
		if idx := strings.LastIndex(description, "("); idx != -1 {
			if endIdx := strings.Index(description[idx:], ")"); endIdx != -1 {
				releaseInfo = description[idx+1 : idx+endIdx]
			}
		}

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
	episodeRegex := regexp.MustCompile(`(\d+)x(\d+)`)
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

		// Extract release info (everything in last parentheses)
		if idx := strings.LastIndex(description, "("); idx != -1 {
			if endIdx := strings.Index(description[idx:], ")"); endIdx != -1 {
				releaseInfo = description[idx+1 : idx+endIdx]
			}
		}

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

// parseReleaseInfo extracts quality and multiple release groups from release info string
// Example: "AMZN.WEB-DL.720p-FLUX, WEB.1080p-SuccessfulCrab"
func (p *SubtitleParser) parseReleaseInfo(releaseInfo string) (quality models.Quality, releaseGroups []string) {
	if releaseInfo == "" {
		return models.QualityUnknown, nil
	}

	releaseGroups = make([]string, 0)
	quality = models.QualityUnknown

	// Split by comma to get individual releases
	releases := strings.Split(releaseInfo, ",")

	for i, release := range releases {
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

		// Detect quality from the first release only
		if i == 0 {
			quality = p.detectQuality(release)
		}
	}

	return quality, releaseGroups
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
	// Try to extract numeric ID from various patterns
	patterns := []string{
		`feliratid=(\d+)`,
		`id=(\d+)`,
		`/(\d+)(?:/|$)`, // Matches /123/ or /123 at end of string
		`(\d+)\.`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(link); len(matches) > 1 {
			return matches[1]
		}
	}

	// Fallback: use the entire link as ID
	return link
}

// extractPaginationInfo extracts current page and total pages from the document
func (p *SubtitleParser) extractPaginationInfo(doc *goquery.Document) (currentPage int, totalPages int) {
	logger := config.GetLogger()

	// Default values
	currentPage = 1
	totalPages = 1

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
			re := regexp.MustCompile(`oldal=(\d+)`)
			if matches := re.FindStringSubmatch(href); len(matches) > 1 {
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
