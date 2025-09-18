package main

import (
	"SuperSubtitles/internal/client"
	"SuperSubtitles/internal/config"
	"context"
	"time"
)

func main() {
	cfg := config.GetConfig()
	logger := config.GetLogger()

	logger.Info().
		Str("proxy_connection_string", cfg.ProxyConnectionString).
		Str("super_subtitle_domain", cfg.SuperSubtitleDomain).
		Int("server_port", cfg.Server.Port).
		Str("server_address", cfg.Server.Address).
		Msg("Application started with configuration")

	// Create a client instance
	httpClient := client.NewClient(cfg)

	// Create a context with timeout for the request
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Fetch the list of shows
	shows, err := httpClient.GetShowList(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch shows")
		return
	}

	logger.Info().Int("total_shows", len(shows)).Msg("Successfully fetched shows")

	// Log first few shows as examples
	for i, show := range shows {
		if i >= 5 { // Limit to first 5 shows
			break
		}
		logger.Info().
			Int("id", show.ID).
			Str("name", show.Name).
			Int("year", show.Year).
			Str("image_url", show.ImageURL).
			Msg("Show information")
	}
}
