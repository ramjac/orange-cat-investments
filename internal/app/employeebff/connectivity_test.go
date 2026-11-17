package employeebff_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/internal/app/employeebff"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockConnectivityBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	resp    *internalbllv1.ListDeviceConnectivityResponse
	err     error
	lastReq *internalbllv1.ListDeviceConnectivityRequest
}

func (m *mockConnectivityBLLClient) ListDeviceConnectivity(ctx context.Context, in *internalbllv1.ListDeviceConnectivityRequest) (*internalbllv1.ListDeviceConnectivityResponse, error) {
	m.lastReq = in
	return m.resp, m.err
}

func TestDeviceConnectivityEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("requires authenticated employee session", func(t *testing.T) {
		bff := employeebff.NewEmployeeBFF(&mockConnectivityBLLClient{}, logger)
		handler := bff.SetupRouter()

		req := httptest.NewRequest(http.MethodGet, "/facilities/devices/connectivity", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("returns device connectivity list for authenticated session", func(t *testing.T) {
		mockBLL := &mockConnectivityBLLClient{
			resp: &internalbllv1.ListDeviceConnectivityResponse{
				Assets: []*facilitiesv1.AssetConnectivity{
					{
						Asset: &facilitiesv1.HardwareAsset{
							AssetID:      "018e0000-0000-7000-8000-000000000001",
							SerialNumber: "QR-CAM-ORANGE-01",
							AssetType:    "edge_camera",
						},
						Online: true,
					},
				},
			},
		}
		bff := employeebff.NewEmployeeBFF(mockBLL, logger)
		handler := bff.SetupRouter()

		req := httptest.NewRequest(http.MethodGet, "/facilities/devices/connectivity?zone_id=zone-alpha", nil)
		req.AddCookie(sessionCookie(t, bff, "emp-eng-01", "staff"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "zone-alpha", mockBLL.lastReq.ZoneID)

		var body struct {
			Items []*facilitiesv1.AssetConnectivity `json:"items"`
			Total int                               `json:"total"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, 1, body.Total)
		require.Len(t, body.Items, 1)
		assert.Equal(t, "QR-CAM-ORANGE-01", body.Items[0].Asset.SerialNumber)
		assert.True(t, body.Items[0].Online)
	})

	t.Run("maps gRPC NotFound to HTTP 404", func(t *testing.T) {
		mockBLL := &mockConnectivityBLLClient{
			err: status.Error(codes.NotFound, "zone not found"),
		}
		bff := employeebff.NewEmployeeBFF(mockBLL, logger)
		handler := bff.SetupRouter()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/facilities/devices/connectivity?zone_id=missing", nil)
		req.AddCookie(sessionCookie(t, bff, "emp-eng-01", "staff"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
