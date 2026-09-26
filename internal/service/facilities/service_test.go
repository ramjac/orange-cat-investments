package facilities

import (
	"context"
	"testing"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct{}

func (m *mockRepo) GetAssetByID(ctx context.Context, id string) (*facilities.HardwareAsset, error) {
	return &facilities.HardwareAsset{AssetID: id, SerialNumber: "CAM-MOCK-01"}, nil
}

func (m *mockRepo) ListAssets(ctx context.Context, limit, offset int32) ([]*facilities.HardwareAsset, error) {
	return nil, nil
}

func (m *mockRepo) CreateAsset(ctx context.Context, asset *facilities.HardwareAsset) (*facilities.HardwareAsset, error) {
	asset.AssetID = "created-mock-id"
	return asset, nil
}

func (m *mockRepo) CreateMaintenanceTicket(ctx context.Context, ticket *facilities.MaintenanceTicket) (*facilities.MaintenanceTicket, error) {
	return nil, nil
}

func TestFacilitiesService(t *testing.T) {
	repo := &mockRepo{}
	service := NewService(repo)

	asset, err := service.GetAsset(context.Background(), "asset-test-1")
	assert.NoError(t, err)
	assert.Equal(t, "asset-test-1", asset.AssetID)

	_, errErr := service.GetAsset(context.Background(), "")
	assert.Error(t, errErr)

	newAsset, errCreate := service.CreateAsset(context.Background(), "SER-123", "edge_camera", "Model4K", "zone-1")
	assert.NoError(t, errCreate)
	assert.Equal(t, "created-mock-id", newAsset.AssetID)
}
