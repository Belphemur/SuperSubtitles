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

	"github.com/Belphemur/SuperSubtitles/v2/internal/client"
	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	grpcserver "github.com/Belphemur/SuperSubtitles/v2/internal/grpc"
	"github.com/Belphemur/SuperSubtitles/v2/internal/metrics"
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
				logger.Fatal().Err(err).Msg("Failed to serve metrics")
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
		logger.Fatal().Err(err).Str("address", address).Msg("Failed to create listener")
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
		logger.Fatal().Err(err).Msg("Failed to serve gRPC")
	}

	logger.Info().Msg("Server stopped gracefully")
}
