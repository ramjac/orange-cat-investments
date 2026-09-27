package facilities

import (
	"context"
	"testing"
	"time"

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

	t.Run("CreateAsset DB Nil Error", func(t *testing.T) {
		input := &HardwareAsset{
			SerialNumber: "SER-12345",
			AssetType:    "feeder",
			Model:        "AutoFeed-v1",
			Status:       "pending",
		}
		_, err := repo.CreateAsset(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection is nil")
	})

	t.Run("CreateMaintenanceTicket", func(t *testing.T) {
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
		inputTicket := &MaintenanceTicket{
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
}
