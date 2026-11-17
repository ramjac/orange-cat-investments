package internalbll

import (
	"context"
	"io"
	"log/slog"
	"testing"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockFacilitiesClient struct {
	facilitiesv1.FacilitiesServiceClient
	heartbeatCalls int
	lastHBReq      *facilitiesv1.RecordAssetHeartbeatRequest
	hbResp         *facilitiesv1.RecordAssetHeartbeatResponse
	hbErr          error
	connResp       *facilitiesv1.ListAssetConnectivityResponse
	connErr        error
	lastConnReq    *facilitiesv1.ListAssetConnectivityRequest
}

func (m *mockFacilitiesClient) RecordAssetHeartbeat(ctx context.Context, in *facilitiesv1.RecordAssetHeartbeatRequest) (*facilitiesv1.RecordAssetHeartbeatResponse, error) {
	m.heartbeatCalls++
	m.lastHBReq = in
	return m.hbResp, m.hbErr
}

func (m *mockFacilitiesClient) ListAssetConnectivity(ctx context.Context, in *facilitiesv1.ListAssetConnectivityRequest) (*facilitiesv1.ListAssetConnectivityResponse, error) {
	m.lastConnReq = in
	return m.connResp, m.connErr
}

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	updateCalls int
	lastReq     *coreinvestv1.UpdateCameraStreamStatusRequest
	resp        *coreinvestv1.UpdateCameraStreamStatusResponse
	err         error
}

func (m *mockCoreInvestClient) UpdateCameraStreamStatus(ctx context.Context, in *coreinvestv1.UpdateCameraStreamStatusRequest) (*coreinvestv1.UpdateCameraStreamStatusResponse, error) {
	m.updateCalls++
	m.lastReq = in
	return m.resp, m.err
}

func TestReportDeviceHeartbeat_FanOutAndErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("empty serial returns InvalidArgument", func(t *testing.T) {
		srv := NewServer(nil, &mockFacilitiesClient{}, nil, nil, logger, Config{CoreInvestClient: &mockCoreInvestClient{}})
		_, err := srv.ReportDeviceHeartbeat(context.Background(), &internalbllv1.ReportDeviceHeartbeatRequest{})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("unknown serial propagates NotFound without calling core-invest", func(t *testing.T) {
		fac := &mockFacilitiesClient{
			hbErr: status.Error(codes.NotFound, "hardware asset not found"),
		}
		ci := &mockCoreInvestClient{}
		srv := NewServer(nil, fac, nil, nil, logger, Config{CoreInvestClient: ci})

		_, err := srv.ReportDeviceHeartbeat(context.Background(), &internalbllv1.ReportDeviceHeartbeatRequest{
			SerialNumber: "UNKNOWN-SERIAL",
			CameraStatus: "active",
		})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(t, 1, fac.heartbeatCalls)
		assert.Equal(t, 0, ci.updateCalls)
	})

	t.Run("without camera_status calls facilities only", func(t *testing.T) {
		fac := &mockFacilitiesClient{
			hbResp: &facilitiesv1.RecordAssetHeartbeatResponse{
				Asset: &facilitiesv1.HardwareAsset{
					AssetID:      "asset-gw-01",
					SerialNumber: "GW-HABITAT-01",
				},
			},
		}
		ci := &mockCoreInvestClient{}
		srv := NewServer(nil, fac, nil, nil, logger, Config{CoreInvestClient: ci})

		res, err := srv.ReportDeviceHeartbeat(context.Background(), &internalbllv1.ReportDeviceHeartbeatRequest{
			SerialNumber: "GW-HABITAT-01",
		})
		require.NoError(t, err)
		require.NotNil(t, res.Asset)
		assert.Equal(t, "asset-gw-01", res.Asset.AssetID)
		assert.Nil(t, res.Stream)
		assert.Equal(t, 1, fac.heartbeatCalls)
		assert.Equal(t, 0, ci.updateCalls)
	})

	t.Run("with camera_status fans out to facilities and core-invest", func(t *testing.T) {
		fac := &mockFacilitiesClient{
			hbResp: &facilitiesv1.RecordAssetHeartbeatResponse{
				Asset: &facilitiesv1.HardwareAsset{
					AssetID:      "018e0000-0000-7000-8000-000000000001",
					SerialNumber: "QR-CAM-ORANGE-01",
				},
			},
		}
		ci := &mockCoreInvestClient{
			resp: &coreinvestv1.UpdateCameraStreamStatusResponse{
				Stream: &coreinvestv1.CameraStream{
					StreamID: "stream-01",
					AssetID:  "018e0000-0000-7000-8000-000000000001",
					Status:   "active",
				},
			},
		}
		srv := NewServer(nil, fac, nil, nil, logger, Config{CoreInvestClient: ci})

		res, err := srv.ReportDeviceHeartbeat(context.Background(), &internalbllv1.ReportDeviceHeartbeatRequest{
			SerialNumber: "QR-CAM-ORANGE-01",
			CameraStatus: "active",
		})
		require.NoError(t, err)
		require.NotNil(t, res.Asset)
		require.NotNil(t, res.Stream)
		assert.Equal(t, "active", res.Stream.Status)
		assert.Equal(t, 1, fac.heartbeatCalls)
		assert.Equal(t, 1, ci.updateCalls)
		assert.Equal(t, "018e0000-0000-7000-8000-000000000001", ci.lastReq.CameraAssetID)
		assert.Equal(t, "active", ci.lastReq.Status)
	})
}

func TestListDeviceConnectivity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fac := &mockFacilitiesClient{
		connResp: &facilitiesv1.ListAssetConnectivityResponse{
			Assets: []*facilitiesv1.AssetConnectivity{
				{
					Asset:  &facilitiesv1.HardwareAsset{SerialNumber: "QR-CAM-ORANGE-01"},
					Online: true,
				},
			},
		},
	}
	srv := NewServer(nil, fac, nil, nil, logger)

	res, err := srv.ListDeviceConnectivity(context.Background(), &internalbllv1.ListDeviceConnectivityRequest{
		ZoneID: "zone-alpha",
	})
	require.NoError(t, err)
	require.Len(t, res.Assets, 1)
	assert.True(t, res.Assets[0].Online)
	assert.Equal(t, "zone-alpha", fac.lastConnReq.ZoneID)
}
