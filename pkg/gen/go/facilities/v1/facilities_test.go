package facilitiesv1

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFacilitiesServer struct {
	UnimplementedFacilitiesServiceServer
}

func (m *mockFacilitiesServer) GetAsset(ctx context.Context, in *GetAssetRequest) (*GetAssetResponse, error) {
	return &GetAssetResponse{Asset: &HardwareAsset{AssetID: in.AssetID, Model: "OCI-Cam-4K"}}, nil
}
func (m *mockFacilitiesServer) ListAssets(ctx context.Context, in *ListAssetsRequest) (*ListAssetsResponse, error) {
	return &ListAssetsResponse{Assets: []*HardwareAsset{{AssetID: "ast-01"}}}, nil
}
func (m *mockFacilitiesServer) CreateAsset(ctx context.Context, in *CreateAssetRequest) (*CreateAssetResponse, error) {
	return &CreateAssetResponse{Asset: &HardwareAsset{SerialNumber: in.SerialNumber}}, nil
}
func (m *mockFacilitiesServer) CreateMaintenanceLog(ctx context.Context, in *CreateMaintenanceLogRequest) (*CreateMaintenanceLogResponse, error) {
	return &CreateMaintenanceLogResponse{Log: &MaintenanceLog{AssetID: in.AssetID}}, nil
}
func (m *mockFacilitiesServer) ListMaintenanceLogs(ctx context.Context, in *ListMaintenanceLogsRequest) (*ListMaintenanceLogsResponse, error) {
	return &ListMaintenanceLogsResponse{}, nil
}
func (m *mockFacilitiesServer) ListFirmwareReleases(ctx context.Context, in *ListFirmwareReleasesRequest) (*ListFirmwareReleasesResponse, error) {
	return &ListFirmwareReleasesResponse{}, nil
}
func (m *mockFacilitiesServer) CreateOTAJob(ctx context.Context, in *CreateOTAJobRequest) (*CreateOTAJobResponse, error) {
	return &CreateOTAJobResponse{Jobs: []*DeviceOTAJob{{ReleaseID: in.ReleaseID}}}, nil
}
func (m *mockFacilitiesServer) ListOTAJobs(ctx context.Context, in *ListOTAJobsRequest) (*ListOTAJobsResponse, error) {
	return &ListOTAJobsResponse{}, nil
}
func (m *mockFacilitiesServer) IngestEnvironmentalTelemetry(ctx context.Context, in *IngestTelemetryRequest) (*IngestTelemetryResponse, error) {
	return &IngestTelemetryResponse{TelemetryID: "tel-01", Status: "ingested"}, nil
}
func (m *mockFacilitiesServer) StreamEnvironmentalTelemetry(req *StreamTelemetryRequest, stream FacilitiesService_StreamEnvironmentalTelemetryServer) error {
	count := req.SampleCount
	if count <= 0 {
		count = 3
	}
	for i := 0; i < int(count); i++ {
		err := stream.Send(&TelemetrySample{
			TelemetryID:         fmt.Sprintf("tel-%d", i),
			ZoneID:              req.ZoneID,
			TemperatureC:        22.5 + float64(i)*0.1,
			RelativeHumidityPct: 45.0,
			LightIntensityLux:   500.0,
			Timestamp:           time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func TestFacilitiesGRPC_RoundTripAndStreaming(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := grpcutil.NewServer(nil)
	RegisterFacilitiesServiceServer(srv, &mockFacilitiesServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpcutil.Dial(context.Background(), lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	client := NewFacilitiesServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Unary
	asset, err := client.GetAsset(ctx, &GetAssetRequest{AssetID: "ast-01"})
	require.NoError(t, err)
	assert.Equal(t, "OCI-Cam-4K", asset.Asset.Model)

	// Streaming
	stream, err := client.StreamEnvironmentalTelemetry(ctx, &StreamTelemetryRequest{
		ZoneID:      "zone-alpha",
		SampleCount: 3,
	})
	require.NoError(t, err)

	var received []*TelemetrySample
	for {
		sample, err := stream.Recv()
		if err != nil {
			break
		}
		received = append(received, sample)
	}

	assert.Len(t, received, 3)
	assert.Equal(t, "zone-alpha", received[0].ZoneID)
}
