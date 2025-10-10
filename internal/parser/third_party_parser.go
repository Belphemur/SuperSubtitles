package parser

import (
	"SuperSubtitles/internal/config"
	"SuperSubtitles/internal/models"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ThirdPartyIdParser implements the SingleResultParser interface for parsing third-party IDs from HTML
type ThirdPartyIdParser struct{}

// NewThirdPartyIdParser creates a new third-party ID parser instance
func NewThirdPartyIdParser() SingleResultParser[models.ThirdPartyIds] {
	return &ThirdPartyIdParser{}
}

// ParseHtml parses the HTML response and extracts third-party IDs
func (p *ThirdPartyIdParser) ParseHtml(body io.Reader) (models.ThirdPartyIds, error) {
	logger := config.GetLogger()
	logger.Info().Msg("Starting third-party ID extraction from HTML")

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse HTML document")
		return models.ThirdPartyIds{}, fmt.Errorf("failed to parse HTML: %w", err)
	}

	logger.Debug().Msg("HTML document parsed successfully, searching for third-party links")

	result := models.ThirdPartyIds{}

	// Find all links in the adatlapRow that contains third-party service links
	doc.Find("div.adatlapRow a").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists {
			logger.Debug().Int("linkIndex", i).Msg("Link missing href attribute")
			return
		}

		logger.Debug().Int("linkIndex", i).Str("href", href).Msg("Found third-party link")

		// Extract IDs based on the service
		if strings.Contains(href, "imdb.com") {
			if imdbID, err := p.extractIMDBIDFromURL(href); err == nil {
				result.IMDBID = imdbID
				logger.Info().Str("imdbID", imdbID).Msg("Successfully extracted IMDB ID")
			} else {
				logger.Debug().Str("href", href).Err(err).Msg("Failed to extract IMDB ID from URL")
			}
		} else if strings.Contains(href, "thetvdb.com") {
			if tvdbID, err := p.extractTVDBIDFromURL(href); err == nil {
				result.TVDBID = tvdbID
				logger.Info().Int("tvdbID", tvdbID).Msg("Successfully extracted TVDB ID")
			} else {
				logger.Debug().Str("href", href).Err(err).Msg("Failed to extract TVDB ID from URL")
			}
		} else if strings.Contains(href, "tvmaze.com") {
			if tvMazeID, err := p.extractTVMazeIDFromURL(href); err == nil {
				result.TVMazeID = tvMazeID
				logger.Info().Int("tvMazeID", tvMazeID).Msg("Successfully extracted TVMaze ID")
			} else {
				logger.Debug().Str("href", href).Err(err).Msg("Failed to extract TVMaze ID from URL")
			}
		} else if strings.Contains(href, "trakt.tv") {
			if traktID, err := p.extractTraktIDFromURL(href); err == nil {
				result.TraktID = traktID
				logger.Info().Int("traktID", traktID).Msg("Successfully extracted Trakt ID")
			} else {
				logger.Debug().Str("href", href).Err(err).Msg("Failed to extract Trakt ID from URL")
			}
		}
	})

	logger.Info().
		Str("imdbId", result.IMDBID).
		Int("tvdbId", result.TVDBID).
		Int("tvMazeId", result.TVMazeID).
		Int("traktId", result.TraktID).
		Msg("Completed third-party ID extraction")

	return result, nil
}

// extractIMDBIDFromURL extracts the IMDB ID from an IMDB URL
func (p *ThirdPartyIdParser) extractIMDBIDFromURL(href string) (string, error) {
	logger := config.GetLogger()

	// IMDB URLs are like: http://www.imdb.com/title/tt14261112/
	re := regexp.MustCompile(`imdb\.com/title/(tt\d+)/?`)
	matches := re.FindStringSubmatch(href)
	if len(matches) < 2 {
		logger.Debug().Str("href", href).Msg("No IMDB ID found in URL")
		return "", fmt.Errorf("no IMDB ID found in URL")
	}

	imdbID := matches[1]
	logger.Debug().Str("href", href).Str("imdbID", imdbID).Msg("Successfully extracted IMDB ID from URL")
	return imdbID, nil
}

// extractTVDBIDFromURL extracts the TVDB ID from a TheTVDB URL
func (p *ThirdPartyIdParser) extractTVDBIDFromURL(href string) (int, error) {
	logger := config.GetLogger()

	// Parse the URL to extract query parameters
	parsedURL, err := url.Parse(href)
	if err != nil {
		logger.Debug().Str("href", href).Err(err).Msg("Failed to parse TVDB URL")
		return 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Look for the "id" query parameter
	idStr := parsedURL.Query().Get("id")
	if idStr == "" {
		logger.Debug().Str("href", href).Msg("No 'id' parameter found in TVDB URL")
		return 0, fmt.Errorf("no 'id' parameter in TVDB URL")
	}

	// Convert the ID string to integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Debug().Str("href", href).Str("idStr", idStr).Err(err).Msg("Failed to convert TVDB ID to integer")
		return 0, fmt.Errorf("invalid TVDB ID format: %s", idStr)
	}

	if id <= 0 {
		logger.Debug().Str("href", href).Int("id", id).Msg("Invalid TVDB ID value")
		return 0, fmt.Errorf("invalid TVDB ID value: %d", id)
	}

	logger.Debug().Str("href", href).Int("id", id).Msg("Successfully extracted TVDB ID from URL")
	return id, nil
}

// extractTVMazeIDFromURL extracts the TVMaze ID from a TVMaze URL
func (p *ThirdPartyIdParser) extractTVMazeIDFromURL(href string) (int, error) {
	logger := config.GetLogger()

	// TVMaze URLs are like: http://www.tvmaze.com/shows/60743
	re := regexp.MustCompile(`tvmaze\.com/shows/(\d+)`)
	matches := re.FindStringSubmatch(href)
	if len(matches) < 2 {
		logger.Debug().Str("href", href).Msg("No TVMaze ID found in URL")
		return 0, fmt.Errorf("no TVMaze ID found in URL")
	}

	idStr := matches[1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Debug().Str("href", href).Str("idStr", idStr).Err(err).Msg("Failed to convert TVMaze ID to integer")
		return 0, fmt.Errorf("invalid TVMaze ID format: %s", idStr)
	}

	if id <= 0 {
		logger.Debug().Str("href", href).Int("id", id).Msg("Invalid TVMaze ID value")
		return 0, fmt.Errorf("invalid TVMaze ID value: %d", id)
	}

	logger.Debug().Str("href", href).Int("id", id).Msg("Successfully extracted TVMaze ID from URL")
	return id, nil
}

// extractTraktIDFromURL extracts the Trakt ID from a Trakt URL
func (p *ThirdPartyIdParser) extractTraktIDFromURL(href string) (int, error) {
	logger := config.GetLogger()

	// Parse the URL to extract query parameters
	parsedURL, err := url.Parse(href)
	if err != nil {
		logger.Debug().Str("href", href).Err(err).Msg("Failed to parse Trakt URL")
		return 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Look for the "query" parameter which contains the TVDB ID
	idStr := parsedURL.Query().Get("query")
	if idStr == "" {
		logger.Debug().Str("href", href).Msg("No 'query' parameter found in Trakt URL")
		return 0, fmt.Errorf("no 'query' parameter in Trakt URL")
	}

	// Convert the ID string to integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Debug().Str("href", href).Str("idStr", idStr).Err(err).Msg("Failed to convert Trakt ID to integer")
		return 0, fmt.Errorf("invalid Trakt ID format: %s", idStr)
	}

	if id <= 0 {
		logger.Debug().Str("href", href).Int("id", id).Msg("Invalid Trakt ID value")
		return 0, fmt.Errorf("invalid Trakt ID value: %d", id)
	}

	logger.Debug().Str("href", href).Int("id", id).Msg("Successfully extracted Trakt ID from URL")
	return id, nil
}
