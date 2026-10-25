package main

import (
	"net"
	"testing"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestWorkforceAPI_GRPC(t *testing.T) {
	repo := workforce.NewMockRepository()
	svc := workforce_svc.NewService(repo)

	grpcServer := grpcutil.NewServer(nil)
	grpcSrv := NewWorkforceGRPCServer(svc)
	workforcev1.RegisterWorkforceServiceServer(grpcServer, grpcSrv)

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
	client := workforcev1.NewWorkforceServiceClient(conn)
	res, err := client.ListEmployees(t.Context(), &workforcev1.ListEmployeesRequest{})
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.Employees)
}

func TestWorkforceAPI_DisabledFeline(t *testing.T) {
	repo := workforce.NewMockRepository()
	svc := workforce_svc.NewService(repo)

	grpcServer := grpcutil.NewServer(nil)
	grpcSrv := NewWorkforceGRPCServer(svc, false)
	workforcev1.RegisterWorkforceServiceServer(grpcServer, grpcSrv)

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

	client := workforcev1.NewWorkforceServiceClient(conn)
	_, err = client.GetCareSchedule(t.Context(), &workforcev1.GetCareScheduleRequest{FelineID: "emp-feline-garfield"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feline workforce domain feature is disabled")
}
