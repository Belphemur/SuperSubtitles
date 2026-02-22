package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "github.com/Belphemur/SuperSubtitles/v2/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/v2/internal/client"
	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	grpcserver "github.com/Belphemur/SuperSubtitles/v2/internal/grpc"
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

	// Create a gRPC server
	grpcServer := grpc.NewServer()

	// Register the SuperSubtitles service
	pb.RegisterSuperSubtitlesServiceServer(grpcServer, grpcserver.NewServer(httpClient))

	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("supersubtitles.v1.SuperSubtitlesService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING) // Overall server health

	// Register reflection service for tools like grpcurl
	reflection.Register(grpcServer)

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
