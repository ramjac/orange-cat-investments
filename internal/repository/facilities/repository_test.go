package facilities

import (
	"context"
	"testing"
)

func TestPgxRepository(t *testing.T) {
	repo := NewRepository(nil)

	asset, err := repo.GetAssetByID(context.Background(), "asset-001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if asset.AssetID != "asset-001" {
		t.Fatalf("expected asset_id asset-001, got %s", asset.AssetID)
	}

	assets, err := repo.ListAssets(context.Background(), 10, 0)
	if err != nil || len(assets) == 0 {
		t.Fatalf("expected assets list, got %v", err)
	}

	// Test BatchInsertMaintenanceLogs
	logs, err := repo.BatchInsertMaintenanceLogs(context.Background(), []*MaintenanceLog{
		{AssetID: "asset-001", QRCodeScanned: "QR-CAM-01", ActionTaken: "Clean lens"},
	})
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 maintenance log inserted, got %v, err=%v", logs, err)
	}

	// Test Firmware & OTA repository functions
	release, err := repo.CreateFirmwareRelease(context.Background(), &FirmwareRelease{
		DeviceType: "edge_camera",
		Version:    "2.1.0",
		FileURL:    "https://firmware.oci.local/v2.1.0.bin",
		Checksum:   "abc123checksum",
	})
	if err != nil || release.ReleaseID == "" {
		t.Fatalf("expected firmware release created, got %v, err=%v", release, err)
	}

	releases, err := repo.ListFirmwareReleases(context.Background(), "edge_camera")
	if err != nil || len(releases) == 0 {
		t.Fatalf("expected firmware releases list, got %v, err=%v", releases, err)
	}

	otaJob, err := repo.CreateOTAJob(context.Background(), &DeviceOTAJob{
		AssetID:   "asset-001",
		ReleaseID: release.ReleaseID,
	})
	if err != nil || otaJob.JobID == "" {
		t.Fatalf("expected OTA job created, got %v, err=%v", otaJob, err)
	}

	otaJobs, err := repo.ListOTAJobsByAsset(context.Background(), "asset-001")
	if err != nil || len(otaJobs) == 0 {
		t.Fatalf("expected OTA jobs list, got %v, err=%v", otaJobs, err)
	}
}
