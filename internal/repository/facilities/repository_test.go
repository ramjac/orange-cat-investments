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
