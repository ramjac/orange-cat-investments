package facilities

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"
)

// generateMockAssets creates dataset Size elements sorted by CreatedAt DESC, AssetID DESC
func generateMockAssets(size int) []*HardwareAsset {
	assets := make([]*HardwareAsset, size)
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	types := []string{"edge_camera", "observation_perch", "smart_collar", "gateway", "feeder", "pebble_watch", "laptop"}
	for i := 0; i < size; i++ {
		assets[i] = &HardwareAsset{
			AssetID:   fmt.Sprintf("asset-%08d", i),
			AssetType: types[i%len(types)],
			CreatedAt: baseTime.Add(time.Duration(i) * time.Second),
		}
	}
	// Sort DESC
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].CreatedAt.Equal(assets[j].CreatedAt) {
			return assets[i].AssetID > assets[j].AssetID
		}
		return assets[i].CreatedAt.After(assets[j].CreatedAt)
	})
	return assets
}

// simulateOffsetPagination skips `offset` items and returns `limit` items
func simulateOffsetPagination(assets []*HardwareAsset, limit, offset int) []*HardwareAsset {
	if offset >= len(assets) {
		return nil
	}
	end := offset + limit
	if end > len(assets) {
		end = len(assets)
	}
	res := make([]*HardwareAsset, 0, end-offset)
	for i := offset; i < end; i++ {
		res = append(res, assets[i])
	}
	return res
}

// simulateKeysetPagination uses binary search to locate cursor (lastCreatedAt, lastID) and takes `limit` items
func simulateKeysetPagination(assets []*HardwareAsset, limit int, cursorCreatedAt *time.Time, cursorID *string) []*HardwareAsset {
	startIndex := 0
	if cursorCreatedAt != nil && cursorID != nil {
		cTime := *cursorCreatedAt
		cID := *cursorID
		// Binary search for first element where (created_at, asset_id) < (cTime, cID)
		idx := sort.Search(len(assets), func(i int) bool {
			a := assets[i]
			if a.CreatedAt.Equal(cTime) {
				return a.AssetID < cID
			}
			return a.CreatedAt.Before(cTime)
		})
		startIndex = idx
	}

	if startIndex >= len(assets) {
		return nil
	}
	end := startIndex + limit
	if end > len(assets) {
		end = len(assets)
	}
	res := make([]*HardwareAsset, 0, end-startIndex)
	for i := startIndex; i < end; i++ {
		res = append(res, assets[i])
	}
	return res
}

func BenchmarkOffsetPagination_DeepPage(b *testing.B) {
	assets := generateMockAssets(100000)
	limit := 20
	offset := 80000 // Deep page query (e.g., page 4000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := simulateOffsetPagination(assets, limit, offset)
		if len(res) == 0 {
			b.Fatal("expected results")
		}
	}
}

func BenchmarkKeysetPagination_DeepPage(b *testing.B) {
	assets := generateMockAssets(100000)
	limit := 20
	// Deep page cursor (item 79999)
	cursorItem := assets[79999]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := simulateKeysetPagination(assets, limit, &cursorItem.CreatedAt, &cursorItem.AssetID)
		if len(res) == 0 {
			b.Fatal("expected results")
		}
	}
}

// BenchmarkFiltering_InMemory measures fetching 1000 assets and filtering in memory.
func BenchmarkFiltering_InMemory(b *testing.B) {
	repo := NewMockRepository()
	ctx := context.Background()
	// Seed mock repo with 1000 assets
	mockR, ok := repo.(*mockRepository)
	if ok {
		for i := 0; i < 1000; i++ {
			assetType := "edge_camera"
			if i%5 == 0 {
				assetType = "pebble_watch"
			}
			mockR.assets[fmt.Sprintf("asset-%d", i)] = &HardwareAsset{
				AssetID:   fmt.Sprintf("asset-%d", i),
				AssetType: assetType,
				CreatedAt: time.Now().UTC(),
			}
		}
	}

	targetType := "pebble_watch"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		items, err := repo.ListAssets(ctx, 1000, nil, nil)
		if err != nil {
			b.Fatal(err)
		}
		var filtered []*HardwareAsset
		for _, item := range items {
			if item.AssetType == targetType {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) == 0 {
			b.Fatal("expected items")
		}
	}
}

// BenchmarkFiltering_DirectFilter measures filtering directly with limit pushed down.
func BenchmarkFiltering_DirectFilter(b *testing.B) {
	repo := NewMockRepository()
	ctx := context.Background()
	// Seed mock repo with 1000 assets
	mockR, ok := repo.(*mockRepository)
	if ok {
		for i := 0; i < 1000; i++ {
			assetType := "edge_camera"
			if i%5 == 0 {
				assetType = "pebble_watch"
			}
			mockR.assets[fmt.Sprintf("asset-%d", i)] = &HardwareAsset{
				AssetID:   fmt.Sprintf("asset-%d", i),
				AssetType: assetType,
				CreatedAt: time.Now().UTC(),
			}
		}
	}

	targetType := "pebble_watch"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		items, err := repo.ListAssets(ctx, 20, nil, nil, &targetType)
		if err != nil {
			b.Fatal(err)
		}
		if len(items) == 0 {
			b.Fatal("expected items")
		}
	}
}
