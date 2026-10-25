package main

import (
	"net"
	"testing"

	facilities_repo "github.com/orange-cat-investments/oci/internal/repository/facilities"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestFacilitiesAPI_GRPC(t *testing.T) {
	repo := facilities_repo.NewMockRepository()
	svc := facilities_svc.NewService(repo)

	grpcServer := grpcutil.NewServer(nil)
	grpcSrv := NewFacilitiesGRPCServer(svc)
	facilitiesv1.RegisterFacilitiesServiceServer(grpcServer, grpcSrv)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	conn, err := grpcutil.Dial(t.Context(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	// Test Health
	healthClient := grpc_health_v1.NewHealthClient(conn)
	hRes, err := healthClient.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, hRes.Status)

	// Test Service RPC
	client := facilitiesv1.NewFacilitiesServiceClient(conn)
	res, err := client.ListAssets(t.Context(), &facilitiesv1.ListAssetsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, res)

	// Test StreamEnvironmentalTelemetry
	stream, err := client.StreamEnvironmentalTelemetry(t.Context(), &facilitiesv1.StreamTelemetryRequest{
		ZoneID:      "zone-alpha",
		SampleCount: 3,
	})
	require.NoError(t, err)
	sampleCount := 0
	for {
		sample, err := stream.Recv()
		if err != nil {
			break
		}
		assert.NotEmpty(t, sample.TelemetryID)
		assert.Equal(t, "zone-alpha", sample.ZoneID)
		assert.Equal(t, 21.5, sample.TemperatureC)
		sampleCount++
	}
	assert.Equal(t, 3, sampleCount)
}
