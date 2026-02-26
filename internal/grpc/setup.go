package grpc

import (
	"sync"

	pb "github.com/Belphemur/SuperSubtitles/v2/api/proto/v1"
	"github.com/Belphemur/SuperSubtitles/v2/internal/client"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	grpcServerMetrics         *grpcprom.ServerMetrics
	registerServerMetricsOnce sync.Once
)

// NewGRPCServer creates a fully configured gRPC server with Prometheus metrics,
// health checking, and reflection.
func NewGRPCServer(c client.Client) *grpc.Server {
	// Set up Prometheus gRPC server metrics once per process
	registerServerMetricsOnce.Do(func() {
		grpcServerMetrics = grpcprom.NewServerMetrics(
			grpcprom.WithServerHandlingTimeHistogram(),
		)
		prometheus.MustRegister(grpcServerMetrics)
	})

	srvMetrics := grpcServerMetrics

	// Create a gRPC server with Prometheus interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(srvMetrics.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(srvMetrics.StreamServerInterceptor()),
	)

	// Register the SuperSubtitles service
	pb.RegisterSuperSubtitlesServiceServer(grpcServer, NewServer(c))

	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("supersubtitles.v1.SuperSubtitlesService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection service for tools like grpcurl
	reflection.Register(grpcServer)

	// Initialize gRPC metrics with all registered service methods
	srvMetrics.InitializeMetrics(grpcServer)

	return grpcServer
}
