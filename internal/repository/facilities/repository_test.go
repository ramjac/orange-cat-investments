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

	t.Run("HardwareAsset Operations", func(t *testing.T) {
		asset, err := repo.CreateAsset(ctx, &facilities.HardwareAsset{
			SerialNumber: "CAM-TEST-01",
			AssetType:    "edge_camera",
			Model:        "TestModel",
			Status:       "active",
		})
		require.NoError(t, err)
		assert.Equal(t, "asset-uuid-created", asset.AssetID)

		fetched, err := repo.GetAssetByID(ctx, asset.AssetID)
		require.NoError(t, err)
		assert.Equal(t, asset.AssetID, fetched.AssetID)

		list, err := repo.ListAssets(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
	})

	t.Run("MaintenanceTicket Operations", func(t *testing.T) {
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
		assert.Nil(t, created.ResolvedAt)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())
		assert.True(t, created.CreatedAt.After(startTime) || created.CreatedAt.Equal(startTime))
		assert.True(t, created.UpdatedAt.After(startTime) || created.UpdatedAt.Equal(startTime))
	})

	t.Run("PgxRepository nil DB error", func(t *testing.T) {
		pgxRepo := facilities.NewRepository(nil)
		_, err := pgxRepo.GetAssetByID(ctx, "123")
		assert.Error(t, err)

		_, err = pgxRepo.ListAssets(ctx, 10, 0)
		assert.Error(t, err)

		_, err = pgxRepo.CreateAsset(ctx, &facilities.HardwareAsset{})
		assert.Error(t, err)

		_, err = pgxRepo.CreateMaintenanceTicket(ctx, &facilities.MaintenanceTicket{})
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
	if created.Status != "open" {
		t.Fatalf("expected status open, got %s", created.Status)
	}
	if created.CreatedAt.IsZero() {
		t.Fatalf("expected non-zero CreatedAt timestamp")
	}
	if created.UpdatedAt.IsZero() {
		t.Fatalf("expected non-zero UpdatedAt timestamp")
	}
}
