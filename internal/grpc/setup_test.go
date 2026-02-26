package grpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
)

func TestNewGRPCServer_ReturnsNonNil(t *testing.T) {
	srv := NewGRPCServer(&mockClient{})
	if srv == nil {
		t.Fatal("Expected non-nil gRPC server")
	}
}

func TestNewGRPCServer_HealthCheck(t *testing.T) {
	srv := NewGRPCServer(&mockClient{})

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)

	// Check overall server health
	resp, err := healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected SERVING status, got %v", resp.Status)
	}

	// Check specific service health
	resp, err = healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
		Service: "supersubtitles.v1.SuperSubtitlesService",
	})
	if err != nil {
		t.Fatalf("Service-specific health check failed: %v", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected SERVING status for service, got %v", resp.Status)
	}
}

func TestNewGRPCServer_ReflectionEnabled(t *testing.T) {
	srv := NewGRPCServer(&mockClient{})

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Verify reflection is available by listing services
	reflectionClient := grpc_reflection_v1.NewServerReflectionClient(conn)
	stream, err := reflectionClient.ServerReflectionInfo(context.Background())
	if err != nil {
		t.Fatalf("Failed to create reflection stream: %v", err)
	}

	err = stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	})
	if err != nil {
		t.Fatalf("Failed to send reflection request: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive reflection response: %v", err)
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		t.Fatal("Expected list services response")
	}

	// Should contain the SuperSubtitles service
	found := false
	for _, svc := range listResp.Service {
		if svc.Name == "supersubtitles.v1.SuperSubtitlesService" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected SuperSubtitlesService to be registered")
	}
}

func TestNewGRPCServer_CalledMultipleTimes(t *testing.T) {
	// Verify sync.Once prevents double-registration panics
	srv1 := NewGRPCServer(&mockClient{})
	srv2 := NewGRPCServer(&mockClient{})

	if srv1 == nil || srv2 == nil {
		t.Fatal("Expected non-nil servers from multiple calls")
	}
}
