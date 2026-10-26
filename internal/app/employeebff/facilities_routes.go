package employeebff

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

// RegisterFacilitiesRoutes mounts Facilities domain endpoints onto the Employee BFF mux.
// All business logic and domain persistence are delegated to the Tier 3 internal-bll over gRPC.
func (b *EmployeeBFF) RegisterFacilitiesRoutes(mux *http.ServeMux) {
	// Assets: List & Create
	listAssets := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		assetType := r.URL.Query().Get("asset_type")
		resp, err := b.bllClient.ListAssets(r.Context(), &facilitiesv1.ListAssetsRequest{Limit: 100, AssetType: assetType})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		assets := resp.Assets
		if assets == nil {
			assets = []*facilitiesv1.HardwareAsset{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": assets,
			"total": len(assets),
		})
	}
	mux.HandleFunc("GET /facilities/assets", listAssets)
	mux.HandleFunc("GET /api/v1/facilities/assets", listAssets)

	createAsset := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			SerialNumber       string `json:"serial_number"`
			AssetType          string `json:"asset_type"`
			Model              string `json:"model"`
			ZoneID             string `json:"zone_id"`
			AssignedTo         string `json:"assigned_to"`
			AssignedEmployeeID string `json:"assigned_employee_id"`
			Notes              string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err := b.bllClient.CreateAsset(r.Context(), &facilitiesv1.CreateAssetRequest{
			SerialNumber: req.SerialNumber,
			AssetType:    req.AssetType,
			Model:        req.Model,
			ZoneID:       req.ZoneID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		assigned := req.AssignedTo
		if assigned == "" {
			assigned = req.AssignedEmployeeID
		}

		respMap := map[string]any{
			"asset_id":      resp.Asset.AssetID,
			"serial_number": resp.Asset.SerialNumber,
			"asset_type":    resp.Asset.AssetType,
			"model":         resp.Asset.Model,
			"zone_id":       resp.Asset.ZoneID,
			"status":        resp.Asset.Status,
			"created_at":    resp.Asset.CreatedAt,
		}
		if assigned != "" {
			respMap["assigned_to"] = assigned
			respMap["assigned_employee_id"] = assigned
		}
		if req.Notes != "" {
			respMap["notes"] = req.Notes
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(respMap)
	}
	mux.HandleFunc("POST /facilities/assets", createAsset)
	mux.HandleFunc("POST /api/v1/facilities/assets", createAsset)

	// Asset Maintenance Ticket
	maintenanceHandler := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		assetID := r.PathValue("id")
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tResp, err := b.bllClient.CreateTicket(r.Context(), &opsv1.CreateTicketRequest{
			ForgejoRepo:    "facilities/assets",
			Title:          req.Title,
			Body:           fmt.Sprintf("Asset %s maintenance: %s", assetID, req.Description),
			AuthorUsername: "emp-human-bob",
			Severity:       req.Priority,
			Category:       "facilities",
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(tResp.Ticket)
	}
	mux.HandleFunc("POST /facilities/assets/{id}/maintenance", maintenanceHandler)
	mux.HandleFunc("POST /api/v1/facilities/assets/{id}/maintenance", maintenanceHandler)

	// Offline Maintenance Logs Sync
	syncLogs := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		type logEntry struct {
			ClientLogID  string `json:"client_log_id"`
			AssetID      string `json:"asset_id"`
			ActionTaken  string `json:"action_taken"`
			PerformedBy  string `json:"performed_by"`
			TechnicianID string `json:"technician_id"`
			Timestamp    string `json:"timestamp"`
			CreatedAt    string `json:"created_at"`
		}

		var entries []logEntry
		// Try parsing as JSON array
		if err := json.Unmarshal(bodyBytes, &entries); err != nil {
			// Fallback: try parsing as wrapped object {"logs": [...]}
			var req struct {
				Logs []logEntry `json:"logs"`
			}
			if err2 := json.Unmarshal(bodyBytes, &req); err2 != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			entries = req.Logs
		}

		var pbLogs []*facilitiesv1.MaintenanceLog
		for _, l := range entries {
			techID := l.PerformedBy
			if techID == "" {
				techID = l.TechnicianID
			}
			syncedAt := l.Timestamp
			if syncedAt == "" {
				syncedAt = l.CreatedAt
			}
			pbLogs = append(pbLogs, &facilitiesv1.MaintenanceLog{
				LogID:        l.ClientLogID,
				AssetID:      l.AssetID,
				ActionTaken:  l.ActionTaken,
				TechnicianID: techID,
				SyncedAt:     syncedAt,
			})
		}

		resp, err := b.bllClient.SyncOfflineLogs(r.Context(), &facilitiesv1.SyncOfflineLogsRequest{Logs: pbLogs})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"synced": resp.SyncedCount,
		})
	}
	mux.HandleFunc("POST /facilities/logs/sync", syncLogs)
	mux.HandleFunc("POST /api/v1/facilities/logs/sync", syncLogs)
	mux.HandleFunc("POST /facilities/sync/maintenance-logs", syncLogs)
	mux.HandleFunc("POST /api/v1/facilities/sync/maintenance-logs", syncLogs)

	// Firmware Releases
	listFirmware := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		resp, err := b.bllClient.ListFirmwareReleases(r.Context(), &facilitiesv1.ListFirmwareReleasesRequest{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		releases := resp.Releases
		if releases == nil {
			releases = []*facilitiesv1.FirmwareRelease{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": releases,
			"total": len(releases),
		})
	}
	mux.HandleFunc("GET /facilities/firmware/releases", listFirmware)
	mux.HandleFunc("GET /api/v1/facilities/firmware/releases", listFirmware)

	createFirmware := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			ReleaseID      string `json:"release_id"`
			Version        string `json:"version"`
			TargetHardware string `json:"target_hardware"`
			ChecksumSHA256 string `json:"checksum_sha256"`
			BinaryURL      string `json:"binary_url"`
			ReleaseNotes   string `json:"release_notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fileURL := req.BinaryURL
		if fileURL == "" {
			fileURL = req.ReleaseID
		}
		resp, err := b.bllClient.CreateFirmwareRelease(r.Context(), &facilitiesv1.CreateFirmwareReleaseRequest{
			DeviceType: req.TargetHardware,
			Version:    req.Version,
			FileURL:    fileURL,
			Checksum:   req.ChecksumSHA256,
			Status:     "available",
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp.Release)
	}
	mux.HandleFunc("POST /facilities/firmware/releases", createFirmware)
	mux.HandleFunc("POST /api/v1/facilities/firmware/releases", createFirmware)

	// OTA Jobs: Trigger & List
	triggerOTA := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			ReleaseID string   `json:"release_id"`
			AssetIDs  []string `json:"asset_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := b.bllClient.TriggerOTARollout(r.Context(), &facilitiesv1.CreateOTAJobRequest{
			ReleaseID: req.ReleaseID,
			AssetIDs:  req.AssetIDs,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "triggered",
			"jobs":   resp.Jobs,
		})
	}
	mux.HandleFunc("POST /facilities/ota/trigger", triggerOTA)
	mux.HandleFunc("POST /api/v1/facilities/ota/trigger", triggerOTA)

	listOTAJobs := func(w http.ResponseWriter, r *http.Request) {
		if b.bllClient == nil {
			http.Error(w, "internal-bll unavailable", http.StatusServiceUnavailable)
			return
		}
		assetID := r.URL.Query().Get("asset_id")
		status := r.URL.Query().Get("status")

		resp, err := b.bllClient.ListOTAJobs(r.Context(), &facilitiesv1.ListOTAJobsRequest{
			AssetID: assetID,
			Status:  status,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jobs := resp.Jobs
		if jobs == nil {
			jobs = []*facilitiesv1.DeviceOTAJob{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": jobs,
			"total": len(jobs),
		})
	}
	mux.HandleFunc("GET /facilities/ota/jobs", listOTAJobs)
	mux.HandleFunc("GET /api/v1/facilities/ota/jobs", listOTAJobs)
}
