package facilities

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgxRepository(t *testing.T) {
	repo := NewRepository(nil)
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
		assets, err := repo.ListAssets(ctx, 10, 0)
		require.NoError(t, err)
		require.NotEmpty(t, assets)
		assert.Equal(t, "asset-uuid-001", assets[0].AssetID)
	})

	t.Run("CreateAsset", func(t *testing.T) {
		input := &HardwareAsset{
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

	t.Run("CreateMaintenanceTicket Nil DB error", func(t *testing.T) {
		techID := "tech-123"
		reporterID := "emp-456"

		inputTicket := &MaintenanceTicket{
			AssetID:              "asset-001",
			Title:                "Camera lens dirty",
			Description:          "The lens needs cleaning on zone 2 camera",
			Priority:             "medium",
			AssignedTechnicianID: &techID,
			ReportedBy:           &reporterID,
		}

		_, err := repo.CreateMaintenanceTicket(ctx, inputTicket)
		assert.Error(t, err)
		assert.EqualError(t, err, "database connection is nil")
	})
}
