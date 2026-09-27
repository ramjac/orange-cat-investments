package facilities_test

import (
	"context"
	"testing"

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
		ticket, err := repo.CreateMaintenanceTicket(ctx, &facilities.MaintenanceTicket{
			AssetID:     "asset-uuid-created",
			Title:       "Lens Cleaning",
			Description: "Clean cat paw smudges",
			Priority:    "medium",
		})
		require.NoError(t, err)
		assert.Equal(t, "maint-uuid-created", ticket.TicketID)
		assert.Equal(t, "open", ticket.Status)
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
