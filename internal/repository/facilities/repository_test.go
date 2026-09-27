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
}

func TestCreateAsset(t *testing.T) {
	repo := NewRepository(nil)

	input := &HardwareAsset{
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
	repo := NewRepository(nil)

	input := &MaintenanceTicket{
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
