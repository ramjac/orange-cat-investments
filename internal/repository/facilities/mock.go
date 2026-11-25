package facilities

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/outbox"
)

type mockRepository struct {
	mu          sync.RWMutex
	assets      map[string]*HardwareAsset
	tickets     map[string]*MaintenanceTicket
	logs        map[string][]*MaintenanceLog
	releases    map[string][]*FirmwareRelease
	otaJobs     map[string][]*DeviceOTAJob
	telemetry   []*EnvironmentalTelemetry
	outboxStore *outbox.MemoryStore
}

func NewMockRepository() Repository {
	return &mockRepository{
		assets:      make(map[string]*HardwareAsset),
		tickets:     make(map[string]*MaintenanceTicket),
		logs:        make(map[string][]*MaintenanceLog),
		releases:    make(map[string][]*FirmwareRelease),
		otaJobs:     make(map[string][]*DeviceOTAJob),
		telemetry:   make([]*EnvironmentalTelemetry, 0),
		outboxStore: outbox.NewMemoryStore(outbox.SchemaFacilities),
	}
}

func (m *mockRepository) GetAssetByID(ctx context.Context, assetID string) (*HardwareAsset, error) {
	m.mu.RLock()
	asset, ok := m.assets[assetID]
	if ok {
		copyAsset := *asset
		m.mu.RUnlock()
		return &copyAsset, nil
	}
	m.mu.RUnlock()
	now := time.Now().UTC()
	frankID := "emp-human-frank"
	bobID := "emp-human-bob"
	rickID := "emp-human-rick"

	switch assetID {
	case "asset-pebble-frank", "PEBBLE-FRANK-01":
		return &HardwareAsset{
			AssetID:            "asset-pebble-frank",
			SerialNumber:       "PEBBLE-FRANK-01",
			AssetType:          "pebble_watch",
			Model:              "Pebble Time Steel",
			Status:             "active",
			AssignedEmployeeID: &frankID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}, nil
	case "asset-pebble-bob", "PEBBLE-BOB-01":
		return &HardwareAsset{
			AssetID:            "asset-pebble-bob",
			SerialNumber:       "PEBBLE-BOB-01",
			AssetType:          "pebble_watch",
			Model:              "Pebble Time",
			Status:             "active",
			AssignedEmployeeID: &bobID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}, nil
	case "asset-laptop-rick", "MBP-RICK-01":
		return &HardwareAsset{
			AssetID:            "asset-laptop-rick",
			SerialNumber:       "MBP-RICK-01",
			AssetType:          "laptop",
			Model:              "MacBook Pro 16-inch M3 Max",
			Status:             "active",
			AssignedEmployeeID: &rickID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}, nil
	default:
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
}

func (m *mockRepository) ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string, assetType ...*string) ([]*HardwareAsset, error) {
	m.mu.RLock()
	var list []*HardwareAsset
	for _, a := range m.assets {
		copyAsset := *a
		list = append(list, &copyAsset)
	}
	m.mu.RUnlock()
	if len(list) == 0 {
		now := time.Now().UTC()
		frankID := "emp-human-frank"
		bobID := "emp-human-bob"
		rickID := "emp-human-rick"
		list = append(list,
			&HardwareAsset{
				AssetID:      "asset-uuid-001",
				SerialNumber: "CAM-ORANGE-01",
				AssetType:    "edge_camera",
				Model:        "4K-FelineCam-v2",
				Status:       "active",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			&HardwareAsset{
				AssetID:      "asset-switch-001",
				SerialNumber: "SW-CORE-01",
				AssetType:    "networking_switch",
				Model:        "Cisco-Catalyst-SMB-24P",
				Status:       "active",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			&HardwareAsset{
				AssetID:      "asset-gateway-001",
				SerialNumber: "GW-HABITAT-01",
				AssetType:    "gateway",
				Model:        "Edge-Gateway-IoT-V1",
				Status:       "active",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			&HardwareAsset{
				AssetID:            "asset-pebble-frank",
				SerialNumber:       "PEBBLE-FRANK-01",
				AssetType:          "pebble_watch",
				Model:              "Pebble Time Steel",
				Status:             "active",
				AssignedEmployeeID: &frankID,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			&HardwareAsset{
				AssetID:            "asset-pebble-bob",
				SerialNumber:       "PEBBLE-BOB-01",
				AssetType:          "pebble_watch",
				Model:              "Pebble Time",
				Status:             "active",
				AssignedEmployeeID: &bobID,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			&HardwareAsset{
				AssetID:            "asset-laptop-rick",
				SerialNumber:       "MBP-RICK-01",
				AssetType:          "laptop",
				Model:              "MacBook Pro 16-inch M3 Max",
				Status:             "active",
				AssignedEmployeeID: &rickID,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		)
	}

	var filterType *string
	if len(assetType) > 0 {
		filterType = assetType[0]
	}

	if filterType != nil && *filterType != "" {
		var filtered []*HardwareAsset
		for _, item := range list {
			if item.AssetType == *filterType {
				filtered = append(filtered, item)
			}
		}
		list = filtered
	}

	return list, nil
}

func (m *mockRepository) CreateAsset(ctx context.Context, asset *HardwareAsset) (*HardwareAsset, error) {
	now := time.Now().UTC()
	asset.AssetID = "asset-uuid-created"
	asset.CreatedAt = now
	asset.UpdatedAt = now
	m.mu.Lock()
	m.assets[asset.AssetID] = asset
	m.mu.Unlock()
	return asset, nil
}

func defaultMockAssets(now time.Time) []*HardwareAsset {
	frankID := "emp-human-frank"
	bobID := "emp-human-bob"
	rickID := "emp-human-rick"
	return []*HardwareAsset{
		{
			AssetID:      "asset-uuid-001",
			SerialNumber: "CAM-ORANGE-01",
			AssetType:    "edge_camera",
			Model:        "4K-FelineCam-v2",
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			AssetID:      "asset-switch-001",
			SerialNumber: "SW-CORE-01",
			AssetType:    "networking_switch",
			Model:        "Cisco-Catalyst-SMB-24P",
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			AssetID:      "asset-gateway-001",
			SerialNumber: "GW-HABITAT-01",
			AssetType:    "gateway",
			Model:        "Edge-Gateway-IoT-V1",
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			AssetID:            "asset-pebble-frank",
			SerialNumber:       "PEBBLE-FRANK-01",
			AssetType:          "pebble_watch",
			Model:              "Pebble Time Steel",
			Status:             "active",
			AssignedEmployeeID: &frankID,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			AssetID:            "asset-pebble-bob",
			SerialNumber:       "PEBBLE-BOB-01",
			AssetType:          "pebble_watch",
			Model:              "Pebble Time",
			Status:             "active",
			AssignedEmployeeID: &bobID,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			AssetID:            "asset-laptop-rick",
			SerialNumber:       "MBP-RICK-01",
			AssetType:          "laptop",
			Model:              "MacBook Pro 16-inch M3 Max",
			Status:             "active",
			AssignedEmployeeID: &rickID,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
	}
}

func (m *mockRepository) RecordAssetHeartbeat(ctx context.Context, serialNumber string, observedAt time.Time) (*HardwareAsset, error) {
	now := time.Now().UTC()
	if observedAt.IsZero() {
		observedAt = now
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var target *HardwareAsset
	for _, a := range m.assets {
		if a.SerialNumber == serialNumber {
			target = a
			break
		}
	}
	if target == nil {
		for _, def := range defaultMockAssets(now) {
			if def.SerialNumber == serialNumber {
				m.assets[def.AssetID] = def
				target = def
				break
			}
		}
	}
	if target == nil {
		return nil, ErrAssetNotFound
	}

	ts := observedAt.UTC()
	target.LastPingAt = &ts
	target.UpdatedAt = now
	copyAsset := *target
	return &copyAsset, nil
}

func (m *mockRepository) ListAssetConnectivity(ctx context.Context, zoneID *string, staleAfterSeconds int32) ([]*AssetConnectivity, error) {
	now := time.Now().UTC()
	m.mu.RLock()
	var list []*HardwareAsset
	for _, a := range m.assets {
		copyAsset := *a
		list = append(list, &copyAsset)
	}
	m.mu.RUnlock()

	if len(list) == 0 {
		list = defaultMockAssets(now)
	}

	var result []*AssetConnectivity
	for _, a := range list {
		if zoneID != nil && *zoneID != "" {
			if a.ZoneID == nil || *a.ZoneID != *zoneID {
				continue
			}
		}
		result = append(result, &AssetConnectivity{
			Asset:  a,
			Online: IsAssetOnline(a.LastPingAt, now, staleAfterSeconds),
		})
	}
	return result, nil
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

func (m *mockRepository) BatchInsertMaintenanceLogs(ctx context.Context, logs []*MaintenanceLog) ([]*MaintenanceLog, error) {
	now := time.Now().UTC()
	result := make([]*MaintenanceLog, len(logs))
	for i, l := range logs {
		cp := *l
		if cp.LogID == "" {
			cp.LogID = "log-uuid-synced"
		}
		cp.SyncedAt = now
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = now
		}
		result[i] = &cp
		m.logs[cp.AssetID] = append(m.logs[cp.AssetID], &cp)
	}
	return result, nil
}

func (m *mockRepository) ListMaintenanceLogsByAsset(ctx context.Context, assetID string) ([]*MaintenanceLog, error) {
	if list, ok := m.logs[assetID]; ok && len(list) > 0 {
		return list, nil
	}
	now := time.Now().UTC()
	return []*MaintenanceLog{
		{
			LogID:         "log-uuid-001",
			AssetID:       assetID,
			QRCodeScanned: "QR-CAM-ORANGE-01",
			ActionTaken:   "Sensor cleaning and recalibration",
			SyncedAt:      now,
			CreatedAt:     now.Add(-10 * time.Minute),
		},
	}, nil
}

func (m *mockRepository) CreateFirmwareRelease(ctx context.Context, release *FirmwareRelease) (*FirmwareRelease, error) {
	now := time.Now().UTC()
	release.ReleaseID = "release-uuid-created"
	if release.Status == "" {
		release.Status = "published"
	}
	release.CreatedAt = now
	release.UpdatedAt = now
	m.releases[release.DeviceType] = append(m.releases[release.DeviceType], release)
	return release, nil
}

func (m *mockRepository) ListFirmwareReleases(ctx context.Context, deviceType string) ([]*FirmwareRelease, error) {
	if deviceType == "" {
		var all []*FirmwareRelease
		for _, list := range m.releases {
			all = append(all, list...)
		}
		if len(all) > 0 {
			return all, nil
		}
		deviceType = "edge_camera"
	}
	if list, ok := m.releases[deviceType]; ok && len(list) > 0 {
		return list, nil
	}
	now := time.Now().UTC()
	return []*FirmwareRelease{
		{
			ReleaseID:  "release-uuid-v2",
			DeviceType: deviceType,
			Version:    "2.1.0",
			FileURL:    "http://forgejo-service.oci-internal.svc.cluster.local:3000/api/packages/oci/generic/edge-firmware/2.1.0/edge_camera.bin",
			Checksum:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Status:     "published",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}, nil
}

func (m *mockRepository) CreateOTAJob(ctx context.Context, job *DeviceOTAJob) (*DeviceOTAJob, error) {
	now := time.Now().UTC()
	job.JobID = "ota-job-uuid-created"
	job.Status = "pending"
	job.ScheduledAt = now
	job.CreatedAt = now
	job.UpdatedAt = now
	m.otaJobs[job.AssetID] = append(m.otaJobs[job.AssetID], job)
	if m.outboxStore != nil {
		if evID, raw, err := outbox.BuildEnvelopeBytes(ctx, "", envelope.TopicOTATriggered, job.JobID, envelope.OTATriggeredPayload{
			JobID:     job.JobID,
			AssetID:   job.AssetID,
			ReleaseID: job.ReleaseID,
		}); err == nil {
			m.outboxStore.Enqueue(evID, envelope.TopicOTATriggered, job.JobID, raw)
		}
	}
	return job, nil
}

func (m *mockRepository) BatchCreateOTAJobs(ctx context.Context, jobs []*DeviceOTAJob) ([]*DeviceOTAJob, error) {
	now := time.Now().UTC()
	result := make([]*DeviceOTAJob, len(jobs))
	for i, job := range jobs {
		cp := *job
		if cp.JobID == "" {
			cp.JobID = "ota-job-uuid-created"
		}
		cp.Status = "pending"
		cp.ScheduledAt = now
		cp.CreatedAt = now
		cp.UpdatedAt = now
		result[i] = &cp
		m.otaJobs[cp.AssetID] = append(m.otaJobs[cp.AssetID], &cp)
		if m.outboxStore != nil {
			if evID, raw, err := outbox.BuildEnvelopeBytes(ctx, "", envelope.TopicOTATriggered, cp.JobID, envelope.OTATriggeredPayload{
				JobID:     cp.JobID,
				AssetID:   cp.AssetID,
				ReleaseID: cp.ReleaseID,
			}); err == nil {
				m.outboxStore.Enqueue(evID, envelope.TopicOTATriggered, cp.JobID, raw)
			}
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateOTAJobStatus(ctx context.Context, jobID string, status string, errorMsg *string, completedAt *time.Time) (*DeviceOTAJob, error) {
	now := time.Now().UTC()
	for _, jobs := range m.otaJobs {
		for _, j := range jobs {
			if j.JobID == jobID {
				j.Status = status
				j.ErrorMessage = errorMsg
				j.CompletedAt = completedAt
				j.UpdatedAt = now
				return j, nil
			}
		}
	}
	return &DeviceOTAJob{
		JobID:        jobID,
		AssetID:      "asset-uuid-001",
		ReleaseID:    "release-uuid-v2",
		Status:       status,
		ErrorMessage: errorMsg,
		ScheduledAt:  now.Add(-5 * time.Minute),
		CompletedAt:  completedAt,
		CreatedAt:    now.Add(-5 * time.Minute),
		UpdatedAt:    now,
	}, nil
}

func (m *mockRepository) ListOTAJobsByAsset(ctx context.Context, assetID string) ([]*DeviceOTAJob, error) {
	if list, ok := m.otaJobs[assetID]; ok && len(list) > 0 {
		return list, nil
	}
	now := time.Now().UTC()
	return []*DeviceOTAJob{
		{
			JobID:       "ota-job-uuid-001",
			AssetID:     assetID,
			ReleaseID:   "release-uuid-v2",
			Status:      "completed",
			ScheduledAt: now.Add(-1 * time.Hour),
			CompletedAt: &now,
			CreatedAt:   now.Add(-1 * time.Hour),
			UpdatedAt:   now,
		},
	}, nil
}

func (m *mockRepository) RecordEnvironmentalTelemetry(ctx context.Context, t *EnvironmentalTelemetry) (*EnvironmentalTelemetry, error) {
	now := time.Now().UTC()
	if t.TelemetryID == "" {
		t.TelemetryID = "telemetry-env-" + time.Now().Format("20060102150405")
	}
	if t.RecordedAt.IsZero() {
		t.RecordedAt = now
	}
	m.telemetry = append(m.telemetry, t)
	return t, nil
}

func (m *mockRepository) GetLatestEnvironmentalTelemetry(ctx context.Context, zoneID string, limit int) ([]*EnvironmentalTelemetry, error) {
	if limit <= 0 {
		limit = 10
	}
	var res []*EnvironmentalTelemetry
	for i := len(m.telemetry) - 1; i >= 0; i-- {
		t := m.telemetry[i]
		if zoneID == "" || t.ZoneID == zoneID {
			res = append(res, t)
			if len(res) >= limit {
				break
			}
		}
	}
	if len(res) == 0 {
		now := time.Now().UTC()
		zid := zoneID
		if zid == "" {
			zid = "zone-alpha"
		}
		for i := 0; i < limit && i < 5; i++ {
			res = append(res, &EnvironmentalTelemetry{
				TelemetryID:         fmt.Sprintf("telemetry-env-seed-%d", i+1),
				ZoneID:              zid,
				TemperatureC:        21.5,
				RelativeHumidityPct: 45.0,
				LightIntensityLux:   520.0,
				NoiseLevelDB:        38.0,
				RecordedAt:          now.Add(-time.Duration(i) * time.Minute),
			})
		}
	}
	return res, nil
}

func (m *mockRepository) ClaimOutboxEvents(ctx context.Context, limit int32, lease time.Duration) ([]*outbox.Record, error) {
	if m.outboxStore == nil {
		return []*outbox.Record{}, nil
	}
	return m.outboxStore.ClaimPending(limit, lease), nil
}

func (m *mockRepository) AckOutboxEvents(ctx context.Context, publishedIDs []int64, failures []outbox.Failure) (int, error) {
	if m.outboxStore == nil {
		return 0, nil
	}
	m.outboxStore.MarkPublished(publishedIDs)
	for _, f := range failures {
		m.outboxStore.MarkFailed(f.ID, f.Error, 0)
	}
	return len(publishedIDs) + len(failures), nil
}

