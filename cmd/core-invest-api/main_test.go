package main

import (
	"net"
	"testing"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestCoreInvestAPI_GRPC(t *testing.T) {
	repo := core_invest_repo.NewMockRepository()
	svc := core_invest_svc.NewService(repo)

	grpcServer := grpcutil.NewServer(nil)
	grpcSrv := NewCoreInvestGRPCServer(svc, repo)
	coreinvestv1.RegisterCoreInvestServiceServer(grpcServer, grpcSrv)

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

	// Test Service RPCs
	client := coreinvestv1.NewCoreInvestServiceClient(conn)

	// List Portfolios
	ports, err := client.ListPortfolios(t.Context(), &coreinvestv1.ListPortfoliosRequest{})
	require.NoError(t, err)
	assert.NotNil(t, ports)

	// Update balance
	balRes, err := client.UpdatePortfolioBalance(t.Context(), &coreinvestv1.UpdatePortfolioBalanceRequest{
		PortfolioID: "port-test-01",
		DeltaUSD:    100.50,
		Reason:      "test balance update",
	})
	require.NoError(t, err)
	assert.Equal(t, "settled", balRes.Status)

	// UpdateCameraStreamStatus - validation & NotFound
	_, err = client.UpdateCameraStreamStatus(t.Context(), &coreinvestv1.UpdateCameraStreamStatusRequest{
		CameraAssetID: "",
		Status:        "idle",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = client.UpdateCameraStreamStatus(t.Context(), &coreinvestv1.UpdateCameraStreamStatusRequest{
		CameraAssetID: "asset-cam-01",
		Status:        "bogus",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = client.UpdateCameraStreamStatus(t.Context(), &coreinvestv1.UpdateCameraStreamStatusRequest{
		CameraAssetID: "unknown-asset-id",
		Status:        "idle",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))

	// Register a stream and update its status
	regRes, err := client.RegisterCameraStream(t.Context(), &coreinvestv1.RegisterCameraStreamRequest{
		AssetID:   "asset-cam-grpc-01",
		StreamURL: "rtsp://cam.local/live",
		Protocol:  "rtsp",
	})
	require.NoError(t, err)
	assert.Equal(t, "active", regRes.Stream.Status)

	updRes, err := client.UpdateCameraStreamStatus(t.Context(), &coreinvestv1.UpdateCameraStreamStatusRequest{
		CameraAssetID: "asset-cam-grpc-01",
		Status:        "idle",
	})
	require.NoError(t, err)
	assert.Equal(t, "idle", updRes.Stream.Status)
}
