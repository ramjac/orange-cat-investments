package facilities

import (
	"context"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
}

func (m *mockRepo) GetAssetByID(ctx context.Context, id string) (*facilities.HardwareAsset, error) {
	return &facilities.HardwareAsset{AssetID: id, SerialNumber: "CAM-MOCK-01"}, nil
}

func (m *mockRepo) ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string, assetType ...*string) ([]*facilities.HardwareAsset, error) {
	return []*facilities.HardwareAsset{
		{AssetID: "mock-asset-1"},
	}, nil
}

func (m *mockRepo) CreateAsset(ctx context.Context, asset *facilities.HardwareAsset) (*facilities.HardwareAsset, error) {
	asset.AssetID = "created-mock-id"
	return asset, nil
}

func (m *mockRepo) CreateMaintenanceTicket(ctx context.Context, ticket *facilities.MaintenanceTicket) (*facilities.MaintenanceTicket, error) {
	return nil, nil
}

func (m *mockRepo) BatchInsertMaintenanceLogs(ctx context.Context, logs []*facilities.MaintenanceLog) ([]*facilities.MaintenanceLog, error) {
	for _, l := range logs {
		l.LogID = "synced-log-mock-id"
	}
	return logs, nil
}

func (m *mockRepo) ListMaintenanceLogsByAsset(ctx context.Context, assetID string) ([]*facilities.MaintenanceLog, error) {
	return []*facilities.MaintenanceLog{{LogID: "log-1", AssetID: assetID}}, nil
}

func (m *mockRepo) CreateFirmwareRelease(ctx context.Context, release *facilities.FirmwareRelease) (*facilities.FirmwareRelease, error) {
	release.ReleaseID = "release-mock-id"
	return release, nil
}

func (m *mockRepo) ListFirmwareReleases(ctx context.Context, deviceType string) ([]*facilities.FirmwareRelease, error) {
	return []*facilities.FirmwareRelease{{ReleaseID: "release-1", DeviceType: deviceType}}, nil
}

func (m *mockRepo) CreateOTAJob(ctx context.Context, job *facilities.DeviceOTAJob) (*facilities.DeviceOTAJob, error) {
	job.JobID = "job-mock-id"
	job.Status = "pending"
	return job, nil
}

func (m *mockRepo) BatchCreateOTAJobs(ctx context.Context, jobs []*facilities.DeviceOTAJob) ([]*facilities.DeviceOTAJob, error) {
	result := make([]*facilities.DeviceOTAJob, len(jobs))
	for i, job := range jobs {
		cp := *job
		cp.JobID = "job-mock-id"
		cp.Status = "pending"
		result[i] = &cp
	}
	return result, nil
}

func (m *mockRepo) UpdateOTAJobStatus(ctx context.Context, jobID string, status string, errorMsg *string, completedAt *time.Time) (*facilities.DeviceOTAJob, error) {
	return &facilities.DeviceOTAJob{JobID: jobID, Status: status}, nil
}

func (m *mockRepo) ListOTAJobsByAsset(ctx context.Context, assetID string) ([]*facilities.DeviceOTAJob, error) {
	return []*facilities.DeviceOTAJob{{JobID: "job-1", AssetID: assetID}}, nil
}

func (m *mockRepo) RecordEnvironmentalTelemetry(ctx context.Context, t *facilities.EnvironmentalTelemetry) (*facilities.EnvironmentalTelemetry, error) {
	t.TelemetryID = "mock-env-123"
	return t, nil
}

func (m *mockRepo) GetLatestEnvironmentalTelemetry(ctx context.Context, zoneID string, limit int) ([]*facilities.EnvironmentalTelemetry, error) {
	return []*facilities.EnvironmentalTelemetry{
		{
			TelemetryID:         "mock-env-123",
			ZoneID:              zoneID,
			TemperatureC:        21.5,
			RelativeHumidityPct: 45.0,
			LightIntensityLux:   520.0,
			NoiseLevelDB:        38.0,
			RecordedAt:          time.Now().UTC(),
		},
	}, nil
}

func TestFacilitiesService(t *testing.T) {
	repo := &mockRepo{}
	service := NewService(repo)

	asset, err := service.GetAsset(context.Background(), "asset-test-1")
	assert.NoError(t, err)
	assert.Equal(t, "asset-test-1", asset.AssetID)

	_, errErr := service.GetAsset(context.Background(), "")
	assert.Error(t, errErr)

	assets, errList := service.ListAssets(context.Background(), 10, nil, nil)
	assert.NoError(t, errList)
	assert.Len(t, assets, 1)

	newAsset, errCreate := service.CreateAsset(context.Background(), "SER-123", "edge_camera", "Model4K", "zone-1")
	assert.NoError(t, errCreate)
	assert.Equal(t, "created-mock-id", newAsset.AssetID)

	// Test SyncMaintenanceLogs
	synced, errSync := service.SyncMaintenanceLogs(context.Background(), []*facilities.MaintenanceLog{
		{AssetID: "asset-1", QRCodeScanned: "QR-01", ActionTaken: "Replaced battery"},
	})
	assert.NoError(t, errSync)
	assert.Len(t, synced, 1)

	// Test SyncMaintenanceLogs error validation
	_, errInvalidSync := service.SyncMaintenanceLogs(context.Background(), []*facilities.MaintenanceLog{
		{AssetID: "", QRCodeScanned: ""},
	})
	assert.Error(t, errInvalidSync)

	// Test Firmware Release & OTA
	rel, errRel := service.CreateFirmwareRelease(context.Background(), &facilities.FirmwareRelease{
		DeviceType: "edge_camera",
		Version:    "1.0.1",
		FileURL:    "http://example.com/fw.bin",
	})
	assert.NoError(t, errRel)
	assert.Equal(t, "release-mock-id", rel.ReleaseID)

	jobs, errOTA := service.TriggerOTAUpdate(context.Background(), []string{"asset-1", "asset-2"}, "release-mock-id")
	assert.NoError(t, errOTA)
	assert.Len(t, jobs, 2)

	// Test IngestEnvironmentalTelemetry
	envRes, errEnv := service.IngestEnvironmentalTelemetry(context.Background(), map[string]interface{}{
		"zone_id":       "zone-alpha",
		"temperature_c": 22.5,
	})
	assert.NoError(t, errEnv)
	assert.Equal(t, "ingested", envRes["status"])

	// Test ControlPTZ
	ptzPayload := map[string]interface{}{"pan": 45, "tilt": 90, "zoom": 2}
	ptzRes, errPTZ := service.ControlPTZ(context.Background(), "cam-001", ptzPayload)
	assert.NoError(t, errPTZ)
	assert.Equal(t, "cam-001", ptzRes["camera_id"])
	assert.Equal(t, "ptz_adjusted", ptzRes["status"])
	assert.Equal(t, ptzPayload, ptzRes["payload"])

	// Test CalibrateCameraLens
	calibRes, errCalib := service.CalibrateCameraLens(context.Background(), "cam-001")
	assert.NoError(t, errCalib)
	assert.Equal(t, "cam-001", calibRes["camera_id"])
	assert.Equal(t, "lens_calibrated", calibRes["status"])
}

