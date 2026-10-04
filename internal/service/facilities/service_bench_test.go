package facilities

import (
	"context"
	"fmt"
	"testing"
)

func generateAssetIDs(count int) []string {
	assetIDs := make([]string, count)
	for i := 0; i < count; i++ {
		assetIDs[i] = fmt.Sprintf("asset-uuid-%06d", i)
	}
	return assetIDs
}

func BenchmarkTriggerOTAUpdate_Batch10(b *testing.B) {
	repo := &mockRepo{}
	service := NewService(repo)
	ctx := context.Background()
	assetIDs := generateAssetIDs(10)
	releaseID := "release-mock-123"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		jobs, err := service.TriggerOTAUpdate(ctx, assetIDs, releaseID)
		if err != nil || len(jobs) != 10 {
			b.Fatalf("failed to trigger OTA update: %v", err)
		}
	}
}

func BenchmarkTriggerOTAUpdate_Batch100(b *testing.B) {
	repo := &mockRepo{}
	service := NewService(repo)
	ctx := context.Background()
	assetIDs := generateAssetIDs(100)
	releaseID := "release-mock-123"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		jobs, err := service.TriggerOTAUpdate(ctx, assetIDs, releaseID)
		if err != nil || len(jobs) != 100 {
			b.Fatalf("failed to trigger OTA update: %v", err)
		}
	}
}

func BenchmarkTriggerOTAUpdate_Batch1000(b *testing.B) {
	repo := &mockRepo{}
	service := NewService(repo)
	ctx := context.Background()
	assetIDs := generateAssetIDs(1000)
	releaseID := "release-mock-123"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		jobs, err := service.TriggerOTAUpdate(ctx, assetIDs, releaseID)
		if err != nil || len(jobs) != 1000 {
			b.Fatalf("failed to trigger OTA update: %v", err)
		}
	}
}
