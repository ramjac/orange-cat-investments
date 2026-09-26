package facilities

import (
	"context"
	"errors"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
)

type Service interface {
	GetAsset(ctx context.Context, id string) (*facilities.HardwareAsset, error)
	CreateAsset(ctx context.Context, serialNumber, assetType, model, zoneID string) (*facilities.HardwareAsset, error)
}

type facilitiesService struct {
	repo facilities.Repository
}

func NewService(repo facilities.Repository) Service {
	return &facilitiesService{repo: repo}
}

func (s *facilitiesService) GetAsset(ctx context.Context, id string) (*facilities.HardwareAsset, error) {
	if id == "" {
		return nil, errors.New("asset id cannot be empty")
	}
	return s.repo.GetAssetByID(ctx, id)
}

func (s *facilitiesService) CreateAsset(ctx context.Context, serialNumber, assetType, model, zoneID string) (*facilities.HardwareAsset, error) {
	if serialNumber == "" || assetType == "" {
		return nil, errors.New("serial number and asset type are required")
	}

	asset := &facilities.HardwareAsset{
		SerialNumber: serialNumber,
		AssetType:    assetType,
		Model:        model,
		Status:       "active",
	}

	return s.repo.CreateAsset(ctx, asset)
}
