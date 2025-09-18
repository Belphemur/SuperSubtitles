package main

import (
	"SuperSubtitles/internal/config"
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
}
