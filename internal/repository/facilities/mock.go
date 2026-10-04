package facilities

import (
	"context"
	"time"
)

type mockRepository struct {
	assets   map[string]*HardwareAsset
	tickets  map[string]*MaintenanceTicket
	logs     map[string][]*MaintenanceLog
	releases map[string][]*FirmwareRelease
	otaJobs  map[string][]*DeviceOTAJob
}

func NewMockRepository() Repository {
	return &mockRepository{
		assets:   make(map[string]*HardwareAsset),
		tickets:  make(map[string]*MaintenanceTicket),
		logs:     make(map[string][]*MaintenanceLog),
		releases: make(map[string][]*FirmwareRelease),
		otaJobs:  make(map[string][]*DeviceOTAJob),
	}
}

func (m *mockRepository) GetAssetByID(ctx context.Context, assetID string) (*HardwareAsset, error) {
	if asset, ok := m.assets[assetID]; ok {
		return asset, nil
	}
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

func (m *mockRepository) ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*HardwareAsset, error) {
	var list []*HardwareAsset
	for _, a := range m.assets {
		list = append(list, a)
	}
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
				AssetID:      "asset-perch-001",
				SerialNumber: "PERCH-HABITAT-01",
				AssetType:    "observation_perch",
				Model:        "Ergonomic-Alpha-Perch-V2",
				Status:       "active",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			&HardwareAsset{
				AssetID:      "asset-collar-001",
				SerialNumber: "COLLAR-SMART-01",
				AssetType:    "smart_collar",
				Model:        "BioSense-Feline-Tag-V3",
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
				AssetID:      "asset-feeder-001",
				SerialNumber: "FEEDER-01",
				AssetType:    "feeder",
				Model:        "AutoFeed-PortionMaster-V1",
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
	if list, ok := m.releases[deviceType]; ok && len(list) > 0 {
		return list, nil
	}
	now := time.Now().UTC()
	return []*FirmwareRelease{
		{
			ReleaseID:  "release-uuid-v2",
			DeviceType: deviceType,
			Version:    "2.1.0",
			FileURL:    "https://firmware.oci.local/edge_camera/v2.1.0.bin",
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

func (m *mockRepository) RecordFeederTelemetry(ctx context.Context, t *FeederTelemetry) (*FeederTelemetry, error) {
	now := time.Now().UTC()
	t.TelemetryID = "telemetry-feeder-" + time.Now().Format("20060102150405")
	if t.DispensedAt.IsZero() {
		t.DispensedAt = now
	}
	return t, nil
}

func (m *mockRepository) RecordCollarTelemetry(ctx context.Context, t *CollarTelemetry) (*CollarTelemetry, error) {
	now := time.Now().UTC()
	t.TelemetryID = "telemetry-collar-" + time.Now().Format("20060102150405")
	if t.RecordedAt.IsZero() {
		t.RecordedAt = now
	}
	return t, nil
}

func (m *mockRepository) RecordPerchTelemetry(ctx context.Context, t *PerchTelemetry) (*PerchTelemetry, error) {
	now := time.Now().UTC()
	t.TelemetryID = "telemetry-perch-" + time.Now().Format("20060102150405")
	if t.RecordedAt.IsZero() {
		t.RecordedAt = now
	}
	return t, nil
}

func (m *mockRepository) RecordEnvironmentalTelemetry(ctx context.Context, t *EnvironmentalTelemetry) (*EnvironmentalTelemetry, error) {
	now := time.Now().UTC()
	t.TelemetryID = "telemetry-env-" + time.Now().Format("20060102150405")
	if t.RecordedAt.IsZero() {
		t.RecordedAt = now
	}
	return t, nil
}

