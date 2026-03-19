package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/buildinfo"
	"github.com/Belphemur/SuperSubtitles/v2/internal/client"
	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	grpcserver "github.com/Belphemur/SuperSubtitles/v2/internal/grpc"
	"github.com/Belphemur/SuperSubtitles/v2/internal/metrics"
	"github.com/Belphemur/SuperSubtitles/v2/internal/sentryio"
)

func main() {
	cfg := config.GetConfig()
	logger := config.GetLogger()
	defer config.FlushSentry()

	// Log application configuration at startup
	logEvent := logger.Info().
		Str("version", buildinfo.Version).
		Str("commit", buildinfo.Commit).
		Str("build_date", buildinfo.Date).
		Str("proxy_connection_string", cfg.ProxyConnectionString).
		Str("super_subtitle_domain", cfg.SuperSubtitleDomain).
		Int("server_port", cfg.Server.Port).
		Str("server_address", cfg.Server.Address)

	// Log cache configuration
	cacheType := cfg.Cache.Type
	if cacheType == "" {
		cacheType = "memory" // default
	}
	logEvent = logEvent.
		Str("cache_type", cacheType).
		Int("cache_size", cfg.Cache.Size).
		Str("cache_ttl", cfg.Cache.TTL)

	// Log Redis-specific configuration if using Redis cache
	if cacheType == "redis" {
		logEvent = logEvent.
			Str("cache_redis_address", cfg.Cache.Redis.Address).
			Int("cache_redis_db", cfg.Cache.Redis.DB)
	}

	// Log metrics configuration
	logEvent = logEvent.
		Bool("metrics_enabled", cfg.Metrics.Enabled)
	if cfg.Metrics.Enabled {
		logEvent = logEvent.Int("metrics_port", cfg.Metrics.Port)
	}

	// Log retry configuration
	logEvent = logEvent.
		Int("retry_max_attempts", cfg.Retry.MaxAttempts).
		Str("retry_initial_delay", cfg.Retry.InitialDelay).
		Str("retry_max_delay", cfg.Retry.MaxDelay)

	logEvent.Msg("Application started with configuration")

	// Create a client instance
	httpClient := client.NewClient(cfg)
	defer func() {
		if err := httpClient.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close client")
		}
	}()

	// Create and configure the gRPC server
	grpcServer := grpcserver.NewGRPCServer(httpClient)

	// Start Prometheus metrics HTTP server
	if cfg.Metrics.Enabled {
		metricsServer := metrics.NewHTTPServer(cfg.Server.Address, cfg.Metrics.Port)
		go func() {
			logger.Info().Str("address", metricsServer.Addr).Msg("Starting Prometheus metrics HTTP server")
			if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				sentryio.CaptureException(err, nil)
				logger.Error().Err(err).Msg("Failed to serve metrics")
				config.FlushSentry()
				os.Exit(1)
			}
		}()
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := metricsServer.Shutdown(ctx); err != nil {
				logger.Error().Err(err).Msg("Failed to shutdown metrics server")
			}
		}()
	}

	// Create a listener
	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		sentryio.CaptureException(err, nil)
		logger.Error().Err(err).Str("address", address).Msg("Failed to create listener")
		config.FlushSentry()
		os.Exit(1)
	}

	logger.Info().Str("address", address).Msg("Starting gRPC server")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		grpcServer.GracefulStop()
	}()

	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		sentryio.CaptureException(err, nil)
		logger.Error().Err(err).Msg("Failed to serve gRPC")
		config.FlushSentry()
		os.Exit(1)
	}

	logger.Info().Msg("Server stopped gracefully")
}
