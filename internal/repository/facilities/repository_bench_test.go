package facilities

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

// generateMockAssets creates dataset Size elements sorted by CreatedAt DESC, AssetID DESC
func generateMockAssets(size int) []*HardwareAsset {
	assets := make([]*HardwareAsset, size)
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < size; i++ {
		assets[i] = &HardwareAsset{
			AssetID:   fmt.Sprintf("asset-%08d", i),
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

func generateMockLogs(count int) []*MaintenanceLog {
	logs := make([]*MaintenanceLog, count)
	assetID := "018f3a9a-0000-7000-8000-000000000001"
	ticketID := "018f3a9a-0000-7000-8000-000000000002"
	techID := "018f3a9a-0000-7000-8000-000000000003"
	noteStr := "Routine sensor cleaning"
	now := time.Now().UTC()

	for i := 0; i < count; i++ {
		logs[i] = &MaintenanceLog{
			TicketID:      &ticketID,
			AssetID:       assetID,
			TechnicianID:  &techID,
			QRCodeScanned: "QR-CAM-ORANGE-01",
			ActionTaken:   "Lens recalibration",
			Notes:         &noteStr,
			CreatedAt:     now,
		}
	}
	return logs
}

func BenchmarkBatchInsertPrep_Loop(b *testing.B) {
	logs := generateMockLogs(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make([]*MaintenanceLog, len(logs))
		for j, l := range logs {
			cp := *l
			if cp.CreatedAt.IsZero() {
				cp.CreatedAt = time.Now().UTC()
			}
			// Simulate N query row parameter packaging
			_ = []interface{}{cp.TicketID, cp.AssetID, cp.TechnicianID, cp.QRCodeScanned, cp.ActionTaken, cp.Notes, cp.CreatedAt}
			result[j] = &cp
		}
	}
}

func BenchmarkBatchInsertPrep_Unnest(b *testing.B) {
	logs := generateMockLogs(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := len(logs)
		ticketIDs := make([]*string, n)
		assetIDs := make([]string, n)
		technicianIDs := make([]*string, n)
		qrCodes := make([]string, n)
		actions := make([]string, n)
		notes := make([]*string, n)
		createdAts := make([]time.Time, n)

		for j, l := range logs {
			ticketIDs[j] = l.TicketID
			assetIDs[j] = l.AssetID
			technicianIDs[j] = l.TechnicianID
			qrCodes[j] = l.QRCodeScanned
			actions[j] = l.ActionTaken
			notes[j] = l.Notes
			if l.CreatedAt.IsZero() {
				createdAts[j] = time.Now().UTC()
			} else {
				createdAts[j] = l.CreatedAt
			}
		}
		_ = []interface{}{ticketIDs, assetIDs, technicianIDs, qrCodes, actions, notes, createdAts}
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
