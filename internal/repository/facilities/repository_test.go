package facilities_test

import (
	"context"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFacilitiesRepository(t *testing.T) {
	repo := facilities.NewMockRepository()
	ctx := context.Background()

	t.Run("GetAssetByID", func(t *testing.T) {
		asset, err := repo.GetAssetByID(ctx, "asset-001")
		require.NoError(t, err)
		assert.Equal(t, "asset-001", asset.AssetID)
		assert.Equal(t, "CAM-ORANGE-01", asset.SerialNumber)
		assert.Equal(t, "edge_camera", asset.AssetType)
		assert.Equal(t, "4K-FelineCam-v2", asset.Model)
		assert.Equal(t, "active", asset.Status)
		assert.False(t, asset.CreatedAt.IsZero())
		assert.False(t, asset.UpdatedAt.IsZero())
	})

	t.Run("ListAssets", func(t *testing.T) {
		assets, err := repo.ListAssets(ctx, 10, nil, nil)
		require.NoError(t, err)
		require.NotEmpty(t, assets)
		assert.Equal(t, "asset-uuid-001", assets[0].AssetID)
	})

	t.Run("ListAssets Filter By AssetType", func(t *testing.T) {
		pebbleType := "pebble_watch"
		assets, err := repo.ListAssets(ctx, 10, nil, nil, &pebbleType)
		require.NoError(t, err)
		require.NotEmpty(t, assets)
		for _, a := range assets {
			assert.Equal(t, "pebble_watch", a.AssetType)
		}
	})

	t.Run("CreateAsset", func(t *testing.T) {
		input := &facilities.HardwareAsset{
			SerialNumber: "SER-12345",
			AssetType:    "feeder",
			Model:        "AutoFeed-v1",
			Status:       "pending",
		}
		created, err := repo.CreateAsset(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, "asset-uuid-created", created.AssetID)
		assert.Equal(t, "SER-12345", created.SerialNumber)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())
	})

	t.Run("CreateMaintenanceTicket", func(t *testing.T) {
		techID := "tech-123"
		reporterID := "emp-456"

		inputTicket := &facilities.MaintenanceTicket{
			AssetID:              "asset-001",
			Title:                "Camera lens dirty",
			Description:          "The lens needs cleaning on zone 2 camera",
			Priority:             "medium",
			AssignedTechnicianID: &techID,
			ReportedBy:           &reporterID,
		}

		startTime := time.Now().UTC()
		created, err := repo.CreateMaintenanceTicket(ctx, inputTicket)
		require.NoError(t, err)
		require.NotNil(t, created)

		assert.Equal(t, "maint-uuid-created", created.TicketID)
		assert.Equal(t, "open", created.Status)
		assert.Equal(t, "asset-001", created.AssetID)
		assert.Equal(t, "Camera lens dirty", created.Title)
		assert.Equal(t, "The lens needs cleaning on zone 2 camera", created.Description)
		assert.Equal(t, "medium", created.Priority)
		assert.Equal(t, &techID, created.AssignedTechnicianID)
		assert.Equal(t, &reporterID, created.ReportedBy)
		assert.Nil(t, created.ResolvedAt)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())
		assert.True(t, created.CreatedAt.After(startTime) || created.CreatedAt.Equal(startTime))
		assert.True(t, created.UpdatedAt.After(startTime) || created.UpdatedAt.Equal(startTime))
	})

	t.Run("CreateMaintenanceTicket with optional fields omitted", func(t *testing.T) {
		inputTicket := &facilities.MaintenanceTicket{
			AssetID:     "asset-002",
			Title:       "Sensor fault",
			Description: "Sensor non-responsive",
			Priority:    "high",
		}

		created, err := repo.CreateMaintenanceTicket(ctx, inputTicket)
		require.NoError(t, err)
		require.NotNil(t, created)

		assert.Equal(t, "maint-uuid-created", created.TicketID)
		assert.Equal(t, "open", created.Status)
		assert.Nil(t, created.AssignedTechnicianID)
		assert.Nil(t, created.ReportedBy)
		assert.Nil(t, created.ResolvedAt)
	})

	t.Run("PgxRepository nil DB error", func(t *testing.T) {
		pgxRepo := facilities.NewRepository(nil)
		_, err := pgxRepo.GetAssetByID(ctx, "asset-001")
		assert.Error(t, err)

		_, err = pgxRepo.ListAssets(ctx, 10, nil, nil)
		assert.Error(t, err)

		_, err = pgxRepo.CreateAsset(ctx, &facilities.HardwareAsset{})
		assert.Error(t, err)

		_, err = pgxRepo.CreateMaintenanceTicket(ctx, &facilities.MaintenanceTicket{})
		assert.Error(t, err)

		_, err = pgxRepo.BatchInsertMaintenanceLogs(ctx, nil)
		assert.Error(t, err)

		_, err = pgxRepo.ListMaintenanceLogsByAsset(ctx, "asset-001")
		assert.Error(t, err)

		_, err = pgxRepo.CreateFirmwareRelease(ctx, &facilities.FirmwareRelease{})
		assert.Error(t, err)

		_, err = pgxRepo.ListFirmwareReleases(ctx, "edge_camera")
		assert.Error(t, err)

		_, err = pgxRepo.CreateOTAJob(ctx, &facilities.DeviceOTAJob{})
		assert.Error(t, err)

		_, err = pgxRepo.UpdateOTAJobStatus(ctx, "job-1", "completed", nil, nil)
		assert.Error(t, err)

		_, err = pgxRepo.ListOTAJobsByAsset(ctx, "asset-001")
		assert.Error(t, err)
	})
}

func TestCreateAsset(t *testing.T) {
	repo := facilities.NewMockRepository()

	input := &facilities.HardwareAsset{
		SerialNumber: "CAM-ORANGE-02",
		AssetType:    "edge_camera",
		Model:        "4K-FelineCam-v3",
		Status:       "active",
	}

	created, err := repo.CreateAsset(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if created.AssetID != "asset-uuid-created" {
		t.Fatalf("expected asset_id asset-uuid-created, got %s", created.AssetID)
	}
	if created.SerialNumber != input.SerialNumber {
		t.Fatalf("expected serial_number %s, got %s", input.SerialNumber, created.SerialNumber)
	}
	if created.AssetType != input.AssetType {
		t.Fatalf("expected asset_type %s, got %s", input.AssetType, created.AssetType)
	}
	if created.Model != input.Model {
		t.Fatalf("expected model %s, got %s", input.Model, created.Model)
	}
	if created.Status != input.Status {
		t.Fatalf("expected status %s, got %s", input.Status, created.Status)
	}
	if created.CreatedAt.IsZero() {
		t.Fatalf("expected non-zero CreatedAt timestamp")
	}
	if created.UpdatedAt.IsZero() {
		t.Fatalf("expected non-zero UpdatedAt timestamp")
	}
}

func TestCreateMaintenanceTicket(t *testing.T) {
	repo := facilities.NewMockRepository()

	input := &facilities.MaintenanceTicket{
		AssetID:     "asset-001",
		Title:       "Sensor calibration",
		Description: "Recalibrate optical sensor for laser pointer detection",
		Priority:    "high",
	}

	created, err := repo.CreateMaintenanceTicket(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if created.TicketID != "maint-uuid-created" {
		t.Fatalf("expected ticket_id maint-uuid-created, got %s", created.TicketID)
	}
	if created.AssetID != input.AssetID {
		t.Fatalf("expected asset_id %s, got %s", input.AssetID, created.AssetID)
	}
	if created.Title != input.Title {
		t.Fatalf("expected title %s, got %s", input.Title, created.Title)
	}
	if created.Description != input.Description {
		t.Fatalf("expected description %s, got %s", input.Description, created.Description)
	}
	if created.Priority != input.Priority {
		t.Fatalf("expected priority %s, got %s", input.Priority, created.Priority)
	}
	if created.Status != "open" {
		t.Fatalf("expected status open, got %s", created.Status)
	}
	if created.CreatedAt.IsZero() {
		t.Fatalf("expected non-zero CreatedAt timestamp")
	}
	if created.UpdatedAt.IsZero() {
		t.Fatalf("expected non-zero UpdatedAt timestamp")
	}

	// Test BatchInsertMaintenanceLogs
	logs, err := repo.BatchInsertMaintenanceLogs(context.Background(), []*facilities.MaintenanceLog{
		{AssetID: "asset-001", QRCodeScanned: "QR-CAM-01", ActionTaken: "Clean lens"},
	})
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 maintenance log inserted, got %v, err=%v", logs, err)
	}

	// Test Firmware & OTA repository functions
	release, err := repo.CreateFirmwareRelease(context.Background(), &facilities.FirmwareRelease{
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

	otaJob, err := repo.CreateOTAJob(context.Background(), &facilities.DeviceOTAJob{
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
