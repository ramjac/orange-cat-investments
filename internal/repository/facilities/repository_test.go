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
