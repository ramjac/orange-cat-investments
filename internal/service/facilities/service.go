package facilities

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
)

type Service interface {
	GetAsset(ctx context.Context, id string) (*facilities.HardwareAsset, error)
	ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*facilities.HardwareAsset, error)
	CreateAsset(ctx context.Context, serialNumber, assetType, model, zoneID string) (*facilities.HardwareAsset, error)
	SyncMaintenanceLogs(ctx context.Context, logs []*facilities.MaintenanceLog) ([]*facilities.MaintenanceLog, error)
	CreateFirmwareRelease(ctx context.Context, release *facilities.FirmwareRelease) (*facilities.FirmwareRelease, error)
	ListFirmwareReleases(ctx context.Context, deviceType string) ([]*facilities.FirmwareRelease, error)
	TriggerOTAUpdate(ctx context.Context, assetIDs []string, releaseID string) ([]*facilities.DeviceOTAJob, error)
	ListOTAJobs(ctx context.Context, assetID string) ([]*facilities.DeviceOTAJob, error)

	IngestFeederTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
	DisburseSnack(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
	IngestCollarTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
	IngestPerchTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
	IngestEnvironmentalTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)
	ControlPTZ(ctx context.Context, cameraID string, payload map[string]interface{}) (map[string]interface{}, error)
	CalibrateCameraLens(ctx context.Context, cameraID string) (map[string]interface{}, error)
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

func (s *facilitiesService) ListAssets(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *string) ([]*facilities.HardwareAsset, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListAssets(ctx, limit, cursorCreatedAt, cursorID)
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
	if zoneID != "" {
		asset.ZoneID = &zoneID
	}

	return s.repo.CreateAsset(ctx, asset)
}

func (s *facilitiesService) SyncMaintenanceLogs(ctx context.Context, logs []*facilities.MaintenanceLog) ([]*facilities.MaintenanceLog, error) {
	if len(logs) == 0 {
		return []*facilities.MaintenanceLog{}, nil
	}
	for _, l := range logs {
		if l.AssetID == "" || l.QRCodeScanned == "" {
			return nil, errors.New("asset_id and qr_code_scanned are required for each maintenance log")
		}
	}
	return s.repo.BatchInsertMaintenanceLogs(ctx, logs)
}

func (s *facilitiesService) CreateFirmwareRelease(ctx context.Context, release *facilities.FirmwareRelease) (*facilities.FirmwareRelease, error) {
	if release == nil || release.DeviceType == "" || release.Version == "" || release.FileURL == "" {
		return nil, errors.New("device_type, version, and file_url are required")
	}
	return s.repo.CreateFirmwareRelease(ctx, release)
}

func (s *facilitiesService) ListFirmwareReleases(ctx context.Context, deviceType string) ([]*facilities.FirmwareRelease, error) {
	if deviceType == "" {
		return nil, errors.New("device_type is required")
	}
	return s.repo.ListFirmwareReleases(ctx, deviceType)
}

func (s *facilitiesService) TriggerOTAUpdate(ctx context.Context, assetIDs []string, releaseID string) ([]*facilities.DeviceOTAJob, error) {
	if len(assetIDs) == 0 || releaseID == "" {
		return nil, errors.New("asset_ids and release_id are required")
	}
	jobs := make([]*facilities.DeviceOTAJob, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		job, err := s.repo.CreateOTAJob(ctx, &facilities.DeviceOTAJob{
			AssetID:   assetID,
			ReleaseID: releaseID,
		})
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *facilitiesService) ListOTAJobs(ctx context.Context, assetID string) ([]*facilities.DeviceOTAJob, error) {
	if assetID == "" {
		return nil, errors.New("asset_id is required")
	}
	return s.repo.ListOTAJobsByAsset(ctx, assetID)
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func (s *facilitiesService) IngestFeederTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	assetID := getString(payload, "asset_id")
	if assetID == "" {
		return nil, errors.New("asset_id is required for feeder telemetry")
	}
	foodDispensed := getFloat(payload, "food_dispensed_grams")
	if foodDispensed < 0 {
		return nil, errors.New("food_dispensed_grams cannot be negative")
	}
	foodConsumed := getFloat(payload, "food_consumed_grams")
	if foodConsumed < 0 {
		return nil, errors.New("food_consumed_grams cannot be negative")
	}

	var felineIDPtr *string
	if felineID := getString(payload, "feline_id"); felineID != "" {
		felineIDPtr = &felineID
	}
	snackDisbursed, _ := payload["snack_disbursed"].(bool)

	rec, err := s.repo.RecordFeederTelemetry(ctx, &facilities.FeederTelemetry{
		AssetID:            assetID,
		FelineID:           felineIDPtr,
		FoodDispensedGrams: foodDispensed,
		FoodConsumedGrams:  foodConsumed,
		SnackDisbursed:     snackDisbursed,
		DispensedAt:        time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist feeder telemetry: %w", err)
	}

	return map[string]interface{}{
		"status":         "ingested",
		"telemetry_id":   rec.TelemetryID,
		"telemetry_type": "smart_feeder",
		"record":         rec,
	}, nil
}

func (s *facilitiesService) DisburseSnack(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	assetID := getString(payload, "asset_id")
	if assetID == "" {
		return nil, errors.New("asset_id is required for snack disbursal")
	}
	snackGrams := getFloat(payload, "snack_grams")
	if snackGrams <= 0 {
		snackGrams = 15.0
	}

	var felineIDPtr *string
	if felineID := getString(payload, "feline_id"); felineID != "" {
		felineIDPtr = &felineID
	}

	rec, err := s.repo.RecordFeederTelemetry(ctx, &facilities.FeederTelemetry{
		AssetID:            assetID,
		FelineID:           felineIDPtr,
		FoodDispensedGrams: snackGrams,
		FoodConsumedGrams:  snackGrams,
		SnackDisbursed:     true,
		DispensedAt:        time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to record snack disbursal: %w", err)
	}

	return map[string]interface{}{
		"status":       "disbursed",
		"telemetry_id": rec.TelemetryID,
		"snack_grams":  snackGrams,
		"record":       rec,
	}, nil
}

func (s *facilitiesService) IngestCollarTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	assetID := getString(payload, "asset_id")
	if assetID == "" {
		return nil, errors.New("asset_id is required for collar telemetry")
	}

	var felineIDPtr *string
	if felineID := getString(payload, "feline_id"); felineID != "" {
		felineIDPtr = &felineID
	}

	var heartRate *int
	if hrVal, ok := payload["heart_rate_bpm"]; ok {
		switch v := hrVal.(type) {
		case float64:
			hr := int(v)
			heartRate = &hr
		case int:
			heartRate = &v
		}
	}

	var pounceG *float64
	if gVal, ok := payload["pounce_g_force"]; ok {
		switch v := gVal.(type) {
		case float64:
			pounceG = &v
		}
	}

	var jumpM *float64
	if jVal, ok := payload["jump_height_meters"]; ok {
		switch v := jVal.(type) {
		case float64:
			jumpM = &v
		}
	}

	var sleepState *string
	if state := getString(payload, "circadian_sleep_state"); state != "" {
		sleepState = &state
	}

	rec, err := s.repo.RecordCollarTelemetry(ctx, &facilities.CollarTelemetry{
		AssetID:             assetID,
		FelineID:            felineIDPtr,
		HeartRateBPM:        heartRate,
		PounceGForce:        pounceG,
		JumpHeightMeters:    jumpM,
		CircadianSleepState: sleepState,
		RecordedAt:          time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist collar telemetry: %w", err)
	}

	return map[string]interface{}{
		"status":         "ingested",
		"telemetry_id":   rec.TelemetryID,
		"telemetry_type": "smart_collar_biometrics",
		"record":         rec,
	}, nil
}

func (s *facilitiesService) IngestPerchTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	perchAssetID := getString(payload, "perch_asset_id")
	if perchAssetID == "" {
		perchAssetID = getString(payload, "asset_id")
	}
	if perchAssetID == "" {
		return nil, errors.New("perch_asset_id is required for perch telemetry")
	}

	pressureMat := getFloat(payload, "pressure_mat_load_kg")
	surfaceTemp := getFloat(payload, "surface_temp_c")
	sunbeamAlignment := getFloat(payload, "sunbeam_alignment_pct")
	cushionWear := getFloat(payload, "cushion_wear_pct")

	rec, err := s.repo.RecordPerchTelemetry(ctx, &facilities.PerchTelemetry{
		PerchAssetID:        perchAssetID,
		PressureMatLoadKG:   pressureMat,
		SurfaceTempC:        surfaceTemp,
		SunbeamAlignmentPct: sunbeamAlignment,
		CushionWearPct:      cushionWear,
		RecordedAt:          time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist perch telemetry: %w", err)
	}

	return map[string]interface{}{
		"status":         "ingested",
		"telemetry_id":   rec.TelemetryID,
		"telemetry_type": "perch_comfort_thermal",
		"record":         rec,
	}, nil
}

func (s *facilitiesService) IngestEnvironmentalTelemetry(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	zoneID := getString(payload, "zone_id")
	if zoneID == "" {
		return nil, errors.New("zone_id is required for environmental telemetry")
	}

	temp := getFloat(payload, "temperature_c")
	humidity := getFloat(payload, "relative_humidity_pct")
	lux := getFloat(payload, "light_intensity_lux")
	noise := getFloat(payload, "noise_level_db")

	rec, err := s.repo.RecordEnvironmentalTelemetry(ctx, &facilities.EnvironmentalTelemetry{
		ZoneID:              zoneID,
		TemperatureC:        temp,
		RelativeHumidityPct: humidity,
		LightIntensityLux:   lux,
		NoiseLevelDB:        noise,
		RecordedAt:          time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist environmental telemetry: %w", err)
	}

	return map[string]interface{}{
		"status":         "ingested",
		"telemetry_id":   rec.TelemetryID,
		"telemetry_type": "environmental_hvac_lux_db",
		"record":         rec,
	}, nil
}

func (s *facilitiesService) ControlPTZ(ctx context.Context, cameraID string, payload map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"camera_id": cameraID, "status": "ptz_adjusted", "payload": payload}, nil
}

func (s *facilitiesService) CalibrateCameraLens(ctx context.Context, cameraID string) (map[string]interface{}, error) {
	return map[string]interface{}{"camera_id": cameraID, "status": "lens_calibrated"}, nil
}
