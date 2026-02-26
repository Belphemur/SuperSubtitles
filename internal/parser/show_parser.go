package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/models"

	"github.com/PuerkitoBio/goquery"
)

// ShowParser implements the Parser interface for parsing show information
type ShowParser struct {
	baseURL string
}

// NewShowParser creates a new show parser instance
func NewShowParser(baseURL string) *ShowParser {
	return &ShowParser{
		baseURL: baseURL,
	}
}

// ParseHtml parses the HTML response and extracts show information
func (p *ShowParser) ParseHtml(body io.Reader) ([]models.Show, error) {
	logger := config.GetLogger()
	logger.Info().Msg("Starting HTML parsing for shows")

	// Convert any character encoding to UTF-8 before parsing
	utf8Body, err := NewUTF8Reader(body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to convert HTML to UTF-8")
		return nil, fmt.Errorf("failed to convert HTML to UTF-8: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse HTML document")
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	logger.Debug().Msg("HTML document parsed successfully, starting show extraction")

	var shows []models.Show
	var currentYear int

	// Find all table rows that contain show information
	doc.Find("tr").Each(func(i int, row *goquery.Selection) {
		// Check if this is a year header row
		yearCell := row.Find(`td[colspan="10"]`)
		if yearCell.Length() > 0 {
			yearText := strings.TrimSpace(yearCell.Text())
			if year, err := strconv.Atoi(yearText); err == nil {
				currentYear = year
				logger.Debug().Int("year", currentYear).Msg("Detected year header")
			} else {
				logger.Debug().Str("yearText", yearText).Msg("Failed to parse year from header")
			}
			return
		}

		// Check if this row contains show information
		showLinks := row.Find(`a[href*="index.php?sid="]`)
		if showLinks.Length() > 0 {
			logger.Debug().Int("row", i).Int("links", showLinks.Length()).Msg("Found show links in row")
			showLinks.Each(func(j int, link *goquery.Selection) {
				if show := p.extractShowFromGoquery(link, currentYear); show != nil {
					shows = append(shows, *show)
					logger.Debug().
						Int("id", show.ID).
						Str("name", show.Name).
						Int("year", show.Year).
						Msg("Successfully extracted show")
				} else {
					logger.Debug().Int("row", i).Int("link", j).Msg("Failed to extract show from link")
				}
			})
		}
	})

	logger.Info().Int("total_shows", len(shows)).Msg("Completed HTML parsing for shows")
	return shows, nil
}

// extractShowFromGoquery extracts show information from a goquery selection
func (p *ShowParser) extractShowFromGoquery(link *goquery.Selection, year int) *models.Show {
	logger := config.GetLogger()

	// Extract show ID from href
	href, exists := link.Attr("href")
	if !exists {
		logger.Debug().Msg("Show link missing href attribute")
		return nil
	}

	logger.Debug().Str("href", href).Msg("Processing show link")

	id := p.extractIDFromHref(href)
	if id == 0 {
		logger.Debug().Str("href", href).Msg("Failed to extract show ID from href")
		return nil
	}

	logger.Debug().Int("id", id).Msg("Extracted show ID")

	// Extract image URL from img src
	img := link.Find("img")
	if img.Length() == 0 {
		logger.Debug().Int("id", id).Msg("No image found for show")
		return nil
	}

	imgSrc, exists := img.Attr("src")
	if !exists {
		logger.Debug().Int("id", id).Msg("Image missing src attribute")
		return nil
	}

	imageURL := p.extractImageURL(imgSrc)
	if imageURL == "" {
		logger.Debug().Int("id", id).Str("imgSrc", imgSrc).Msg("Failed to construct image URL")
		return nil
	}

	logger.Debug().Int("id", id).Str("imageURL", imageURL).Msg("Extracted image URL")

	// Find the show name - it's usually in the next td.sangol element
	name := p.extractShowNameFromGoquery(link)
	if name == "" {
		logger.Debug().Int("id", id).Msg("No show name found, using fallback")
		name = fmt.Sprintf("Show %d", id) // Fallback
	}

	logger.Debug().Int("id", id).Str("name", name).Msg("Final show name")

	return &models.Show{
		Name:     name,
		ID:       id,
		Year:     year,
		ImageURL: imageURL,
	}
}

// extractIDFromHref extracts the show ID from href attribute
func (p *ShowParser) extractIDFromHref(href string) int {
	logger := config.GetLogger()
	const prefix = "index.php?sid="
	if idx := strings.Index(href, prefix); idx != -1 {
		idStr := href[idx+len(prefix):]
		if id, err := strconv.Atoi(idStr); err == nil {
			logger.Debug().Str("href", href).Int("id", id).Msg("Extracted ID from href")
			return id
		}
		logger.Debug().Str("href", href).Str("idStr", idStr).Msg("Failed to convert ID to integer")
	}
	logger.Debug().Str("href", href).Str("prefix", prefix).Msg("Prefix not found in href")
	return 0
}

// extractImageURL extracts the full image URL from src attribute
func (p *ShowParser) extractImageURL(src string) string {
	logger := config.GetLogger()
	const prefix = "sorozat_cat.php?kep="
	if idx := strings.Index(src, prefix); idx != -1 {
		imageID := src[idx+len(prefix):]
		fullURL := fmt.Sprintf("%s/sorozat_cat.php?kep=%s", p.baseURL, imageID)
		logger.Debug().Str("src", src).Str("imageID", imageID).Str("fullURL", fullURL).Msg("Constructed image URL")
		return fullURL
	}
	logger.Debug().Str("src", src).Str("prefix", prefix).Msg("Image prefix not found in src")
	return ""
}

// extractShowNameFromGoquery finds the show name from the goquery selection
func (p *ShowParser) extractShowNameFromGoquery(link *goquery.Selection) string {
	logger := config.GetLogger()

	// Get the parent td of the link (this is the image cell)
	parentTD := link.Closest("td")
	if parentTD.Length() == 0 {
		logger.Debug().Msg("No parent td found for show link")
		return ""
	}

	logger.Debug().Msg("Found parent td, searching for name in next sibling")

	// Find the next td with class "sangol" (this is the name cell)
	// The name cell is always immediately after the image cell in the HTML structure
	nameTD := parentTD.Next()
	if nameTD.Length() == 0 || !nameTD.HasClass("sangol") {
		logger.Debug().Msg("No td.sangol found after image td")
		return ""
	}

	// Extract the name from the div
	div := nameTD.Find("div").First()
	if div.Length() == 0 {
		logger.Debug().Msg("No div found in td.sangol")
		return ""
	}

	name := strings.TrimSpace(div.Text())
	if name == "" || name == "(Tuiskoms)" {
		logger.Debug().Str("name", name).Msg("Skipping invalid show name")
		return ""
	}

	logger.Debug().Str("name", name).Msg("Successfully extracted show name")
	return name
}

// ExtractLastPage extracts the last page number from the pagination HTML.
// Returns 1 if there is no pagination (single page).
func (p *ShowParser) ExtractLastPage(body io.Reader) int {
	logger := config.GetLogger()

	// Convert any character encoding to UTF-8 before parsing
	utf8Body, err := NewUTF8Reader(body)
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to convert HTML to UTF-8")
		return 1
	}

	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to parse HTML for pagination")
		return 1
	}

	lastPage := 1

	// The pagination div contains links like: <a href="/index.php?oldal=42&sorf=...">42</a>
	// We find all pagination links with "oldal=" and track the highest page number.
	doc.Find("div.pagination a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		if !strings.Contains(href, "oldal=") {
			return
		}

		// Extract page number from text content (skip ">" navigation links)
		text := strings.TrimSpace(s.Text())
		if page, err := strconv.Atoi(text); err == nil && page > lastPage {
			lastPage = page
		}
	})

	logger.Debug().Int("lastPage", lastPage).Msg("Extracted last page from pagination")
	return lastPage
}
