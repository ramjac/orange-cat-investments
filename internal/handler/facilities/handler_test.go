package facilities

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	"github.com/stretchr/testify/assert"
)

func TestFacilitiesHandler(t *testing.T) {
	repo := facilities.NewMockRepository()
	svc := facilities_svc.NewService(repo)
	handler := NewHandler(svc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Test GET /facilities/assets/asset-001
	req := httptest.NewRequest("GET", "/facilities/assets/asset-001", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Test POST /facilities/sync/maintenance-logs
	logs := []*facilities.MaintenanceLog{
		{AssetID: "asset-1", QRCodeScanned: "QR-01", ActionTaken: "Lens cleaning"},
	}
	body, _ := json.Marshal(logs)
	reqSync := httptest.NewRequest("POST", "/facilities/sync/maintenance-logs", bytes.NewBuffer(body))
	rrSync := httptest.NewRecorder()
	mux.ServeHTTP(rrSync, reqSync)
	assert.Equal(t, http.StatusOK, rrSync.Code)

	// Test POST /facilities/firmware
	release := &facilities.FirmwareRelease{
		DeviceType: "edge_camera",
		Version:    "2.2.0",
		FileURL:    "https://firmware.oci.local/v2.2.0.bin",
		Checksum:   "sha256val",
	}
	relBody, _ := json.Marshal(release)
	reqRel := httptest.NewRequest("POST", "/facilities/firmware", bytes.NewBuffer(relBody))
	rrRel := httptest.NewRecorder()
	mux.ServeHTTP(rrRel, reqRel)
	assert.Equal(t, http.StatusCreated, rrRel.Code)

	// Test POST /facilities/firmware/ota-jobs
	otaReq := OTATriggerRequest{
		AssetIDs:  []string{"asset-1", "asset-2"},
		ReleaseID: "release-mock-1",
	}
	otaBody, _ := json.Marshal(otaReq)
	reqOTA := httptest.NewRequest("POST", "/facilities/firmware/ota-jobs", bytes.NewBuffer(otaBody))
	rrOTA := httptest.NewRecorder()
	mux.ServeHTTP(rrOTA, reqOTA)
	assert.Equal(t, http.StatusAccepted, rrOTA.Code)
}
