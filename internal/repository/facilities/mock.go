package facilities

import (
	"context"
	"time"
)

type mockRepository struct {
	assets  map[string]*HardwareAsset
	tickets map[string]*MaintenanceTicket
}

func NewMockRepository() Repository {
	return &mockRepository{
		assets:  make(map[string]*HardwareAsset),
		tickets: make(map[string]*MaintenanceTicket),
	}
}

func (m *mockRepository) GetAssetByID(ctx context.Context, assetID string) (*HardwareAsset, error) {
	if asset, ok := m.assets[assetID]; ok {
		return asset, nil
	}
	now := time.Now().UTC()
	return &HardwareAsset{
		AssetID:      assetID,
		SerialNumber: "CAM-ORANGE-01",
		AssetType:    "edge_camera",
		Model:        "4K-FelineCam-v2",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (m *mockRepository) ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*HardwareAsset, error) {
	var list []*HardwareAsset
	for _, a := range m.assets {
		list = append(list, a)
	}
	if len(list) == 0 {
		now := time.Now().UTC()
		list = append(list, &HardwareAsset{
			AssetID:      "asset-uuid-001",
			SerialNumber: "CAM-ORANGE-01",
			AssetType:    "edge_camera",
			Model:        "4K-FelineCam-v2",
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	return list, nil
}

func (m *mockRepository) CreateAsset(ctx context.Context, asset *HardwareAsset) (*HardwareAsset, error) {
	now := time.Now().UTC()
	asset.AssetID = "asset-uuid-created"
	asset.CreatedAt = now
	asset.UpdatedAt = now
	m.assets[asset.AssetID] = asset
	return asset, nil
}

func (m *mockRepository) CreateMaintenanceTicket(ctx context.Context, ticket *MaintenanceTicket) (*MaintenanceTicket, error) {
	now := time.Now().UTC()
	ticket.TicketID = "maint-uuid-created"
	ticket.Status = "open"
	ticket.CreatedAt = now
	ticket.UpdatedAt = now
	m.tickets[ticket.TicketID] = ticket
	return ticket, nil
}
