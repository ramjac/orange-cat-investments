package facilities

import (
	"encoding/json"
	"net/http"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
)

type Handler struct {
	service facilities_svc.Service
}

func NewHandler(service facilities_svc.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /facilities/assets", h.ListAssets)
	mux.HandleFunc("POST /facilities/assets", h.CreateAsset)
	mux.HandleFunc("GET /facilities/assets/{id}", h.GetAssetByID)

	mux.HandleFunc("POST /facilities/sync/maintenance-logs", h.SyncMaintenanceLogs)

	mux.HandleFunc("GET /facilities/firmware", h.ListFirmwareReleases)
	mux.HandleFunc("POST /facilities/firmware", h.CreateFirmwareRelease)

	mux.HandleFunc("GET /facilities/firmware/ota-jobs", h.ListOTAJobs)
	mux.HandleFunc("POST /facilities/firmware/ota-jobs", h.TriggerOTAUpdate)

	mux.HandleFunc("POST /facilities/feeders/telemetry", h.IngestFeederTelemetry)
	mux.HandleFunc("POST /facilities/feeders/disburse", h.DisburseSnack)
	mux.HandleFunc("POST /facilities/collars/telemetry", h.IngestCollarTelemetry)
	mux.HandleFunc("POST /facilities/perches/telemetry", h.IngestPerchTelemetry)
	mux.HandleFunc("POST /facilities/environment/telemetry", h.IngestEnvironmentalTelemetry)
	mux.HandleFunc("POST /facilities/cameras/{id}/ptz", h.ControlPTZ)
	mux.HandleFunc("POST /facilities/cameras/{id}/calibrate", h.CalibrateCameraLens)

	// API v1 route aliases
	mux.HandleFunc("GET /api/v1/facilities/assets", h.ListAssets)
	mux.HandleFunc("POST /api/v1/facilities/assets", h.CreateAsset)
	mux.HandleFunc("GET /api/v1/facilities/assets/{id}", h.GetAssetByID)

	mux.HandleFunc("POST /api/v1/facilities/sync/maintenance-logs", h.SyncMaintenanceLogs)

	mux.HandleFunc("GET /api/v1/facilities/firmware", h.ListFirmwareReleases)
	mux.HandleFunc("POST /api/v1/facilities/firmware", h.CreateFirmwareRelease)

	mux.HandleFunc("GET /api/v1/facilities/firmware/ota-jobs", h.ListOTAJobs)
	mux.HandleFunc("POST /api/v1/facilities/firmware/ota-jobs", h.TriggerOTAUpdate)

	mux.HandleFunc("POST /api/v1/facilities/feeders/telemetry", h.IngestFeederTelemetry)
	mux.HandleFunc("POST /api/v1/facilities/feeders/disburse", h.DisburseSnack)
	mux.HandleFunc("POST /api/v1/facilities/collars/telemetry", h.IngestCollarTelemetry)
	mux.HandleFunc("POST /api/v1/facilities/perches/telemetry", h.IngestPerchTelemetry)
	mux.HandleFunc("POST /api/v1/facilities/environment/telemetry", h.IngestEnvironmentalTelemetry)
	mux.HandleFunc("POST /api/v1/facilities/cameras/{id}/ptz", h.ControlPTZ)
	mux.HandleFunc("POST /api/v1/facilities/cameras/{id}/calibrate", h.CalibrateCameraLens)
}

func (h *Handler) GetAssetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	if id == "" {
		http.Error(w, "asset id is required", http.StatusBadRequest)
		return
	}

	asset, err := h.service.GetAsset(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

func (h *Handler) ListAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.service.ListAssets(r.Context(), 50, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": assets,
		"total": len(assets),
	})
}

type CreateAssetRequest struct {
	SerialNumber string `json:"serial_number"`
	AssetType    string `json:"asset_type"`
	Model        string `json:"model"`
	ZoneID       string `json:"zone_id"`
}

func (h *Handler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var req CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	asset, err := h.service.CreateAsset(r.Context(), req.SerialNumber, req.AssetType, req.Model, req.ZoneID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(asset)
}

func (h *Handler) SyncMaintenanceLogs(w http.ResponseWriter, r *http.Request) {
	var logs []*facilities.MaintenanceLog
	if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	syncedLogs, err := h.service.SyncMaintenanceLogs(r.Context(), logs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"synced_count": len(syncedLogs),
		"items":        syncedLogs,
	})
}

func (h *Handler) ListFirmwareReleases(w http.ResponseWriter, r *http.Request) {
	deviceType := r.URL.Query().Get("device_type")
	if deviceType == "" {
		deviceType = "edge_camera"
	}

	releases, err := h.service.ListFirmwareReleases(r.Context(), deviceType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(releases)
}

func (h *Handler) CreateFirmwareRelease(w http.ResponseWriter, r *http.Request) {
	var req facilities.FirmwareRelease
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	release, err := h.service.CreateFirmwareRelease(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(release)
}

func (h *Handler) ListOTAJobs(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		http.Error(w, "asset_id is required", http.StatusBadRequest)
		return
	}

	jobs, err := h.service.ListOTAJobs(r.Context(), assetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

type OTATriggerRequest struct {
	AssetIDs  []string `json:"asset_ids"`
	ReleaseID string   `json:"release_id"`
}

func (h *Handler) IngestFeederTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ingested", "telemetry_type": "smart_feeder"})
}

func (h *Handler) DisburseSnack(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "disbursed", "snack_grams": 15.0})
}

func (h *Handler) IngestCollarTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ingested", "telemetry_type": "smart_collar_biometrics"})
}

func (h *Handler) IngestPerchTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ingested", "telemetry_type": "perch_comfort_thermal"})
}

func (h *Handler) IngestEnvironmentalTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ingested", "telemetry_type": "environmental_hvac_lux_db"})
}

func (h *Handler) ControlPTZ(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"camera_id": id, "status": "ptz_adjusted"})
}

func (h *Handler) CalibrateCameraLens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"camera_id": id, "status": "lens_calibrated"})
}

func (h *Handler) TriggerOTAUpdate(w http.ResponseWriter, r *http.Request) {
	var req OTATriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	jobs, err := h.service.TriggerOTAUpdate(r.Context(), req.AssetIDs, req.ReleaseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": jobs,
	})
}
