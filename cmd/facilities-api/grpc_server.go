package main

import (
	"context"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/facilities"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FacilitiesGRPCServer struct {
	facilitiesv1.UnimplementedFacilitiesServiceServer
	svc  facilities_svc.Service
	repo facilities.Repository
}

func NewFacilitiesGRPCServer(svc facilities_svc.Service) *FacilitiesGRPCServer {
	return &FacilitiesGRPCServer{svc: svc}
}

func NewFacilitiesGRPCServerWithRepo(svc facilities_svc.Service, repo facilities.Repository) *FacilitiesGRPCServer {
	return &FacilitiesGRPCServer{svc: svc, repo: repo}
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toProtoAsset(a *facilities.HardwareAsset) *facilitiesv1.HardwareAsset {
	if a == nil {
		return nil
	}
	lastPing := ""
	if a.LastPingAt != nil {
		lastPing = a.LastPingAt.Format(time.RFC3339)
	}
	created := ""
	if !a.CreatedAt.IsZero() {
		created = a.CreatedAt.Format(time.RFC3339)
	}
	return &facilitiesv1.HardwareAsset{
		AssetID:            a.AssetID,
		SerialNumber:       a.SerialNumber,
		AssetType:          a.AssetType,
		Model:              a.Model,
		Status:             a.Status,
		ZoneID:             strVal(a.ZoneID),
		AssignedEmployeeID: strVal(a.AssignedEmployeeID),
		LastPingAt:         lastPing,
		CreatedAt:          created,
	}
}

func toProtoMaintenanceLog(l *facilities.MaintenanceLog) *facilitiesv1.MaintenanceLog {
	if l == nil {
		return nil
	}
	synced := ""
	if !l.SyncedAt.IsZero() {
		synced = l.SyncedAt.Format(time.RFC3339)
	}
	return &facilitiesv1.MaintenanceLog{
		LogID:         l.LogID,
		TicketID:      strVal(l.TicketID),
		AssetID:       l.AssetID,
		TechnicianID:  strVal(l.TechnicianID),
		QRCodeScanned: l.QRCodeScanned,
		ActionTaken:   l.ActionTaken,
		Notes:         strVal(l.Notes),
		SyncedAt:      synced,
	}
}

func toProtoFirmwareRelease(r *facilities.FirmwareRelease) *facilitiesv1.FirmwareRelease {
	if r == nil {
		return nil
	}
	return &facilitiesv1.FirmwareRelease{
		ReleaseID:          r.ReleaseID,
		DeviceType:         r.DeviceType,
		Version:            r.Version,
		FileURL:            r.FileURL,
		Checksum:           r.Checksum,
		MinHardwareVersion: strVal(r.MinHardwareVersion),
		Status:             r.Status,
		CreatedAt:          r.CreatedAt.Format(time.RFC3339),
	}
}

func toProtoOTAJob(j *facilities.DeviceOTAJob) *facilitiesv1.DeviceOTAJob {
	if j == nil {
		return nil
	}
	comp := ""
	if j.CompletedAt != nil {
		comp = j.CompletedAt.Format(time.RFC3339)
	}
	return &facilitiesv1.DeviceOTAJob{
		JobID:        j.JobID,
		AssetID:      j.AssetID,
		ReleaseID:    j.ReleaseID,
		Status:       j.Status,
		ErrorMessage: strVal(j.ErrorMessage),
		ScheduledAt:  j.ScheduledAt.Format(time.RFC3339),
		CompletedAt:  comp,
	}
}

func (s *FacilitiesGRPCServer) GetAsset(ctx context.Context, in *facilitiesv1.GetAssetRequest) (*facilitiesv1.GetAssetResponse, error) {
	a, err := s.svc.GetAsset(ctx, in.AssetID)
	if err != nil {
		return nil, err
	}
	return &facilitiesv1.GetAssetResponse{Asset: toProtoAsset(a)}, nil
}

func (s *FacilitiesGRPCServer) ListAssets(ctx context.Context, in *facilitiesv1.ListAssetsRequest) (*facilitiesv1.ListAssetsResponse, error) {
	var assetTypePtr *string
	if in.AssetType != "" {
		assetTypePtr = &in.AssetType
	}
	assets, err := s.svc.ListAssets(ctx, in.Limit, nil, nil, assetTypePtr)
	if err != nil {
		return nil, err
	}
	var out []*facilitiesv1.HardwareAsset
	for _, a := range assets {
		out = append(out, toProtoAsset(a))
	}
	return &facilitiesv1.ListAssetsResponse{Assets: out}, nil
}

func (s *FacilitiesGRPCServer) CreateAsset(ctx context.Context, in *facilitiesv1.CreateAssetRequest) (*facilitiesv1.CreateAssetResponse, error) {
	created, err := s.svc.CreateAsset(ctx, in.SerialNumber, in.AssetType, in.Model, in.ZoneID)
	if err != nil {
		return nil, err
	}
	return &facilitiesv1.CreateAssetResponse{Asset: toProtoAsset(created)}, nil
}

func (s *FacilitiesGRPCServer) CreateMaintenanceLog(ctx context.Context, in *facilitiesv1.CreateMaintenanceLogRequest) (*facilitiesv1.CreateMaintenanceLogResponse, error) {
	log := &facilities.MaintenanceLog{
		AssetID:       in.AssetID,
		TechnicianID:  strPtr(in.TechnicianID),
		QRCodeScanned: in.QRCodeScanned,
		ActionTaken:   in.ActionTaken,
		Notes:         strPtr(in.Notes),
		SyncedAt:      time.Now().UTC(),
	}
	synced, err := s.svc.SyncMaintenanceLogs(ctx, []*facilities.MaintenanceLog{log})
	if err != nil {
		return nil, err
	}
	if len(synced) == 0 {
		return nil, fmt.Errorf("no log synced")
	}
	return &facilitiesv1.CreateMaintenanceLogResponse{Log: toProtoMaintenanceLog(synced[0])}, nil
}

func (s *FacilitiesGRPCServer) ListMaintenanceLogs(ctx context.Context, in *facilitiesv1.ListMaintenanceLogsRequest) (*facilitiesv1.ListMaintenanceLogsResponse, error) {
	if s.repo == nil || in.AssetID == "" {
		return &facilitiesv1.ListMaintenanceLogsResponse{}, nil
	}
	logs, err := s.repo.ListMaintenanceLogsByAsset(ctx, in.AssetID)
	if err != nil {
		return nil, err
	}
	var out []*facilitiesv1.MaintenanceLog
	for _, l := range logs {
		out = append(out, toProtoMaintenanceLog(l))
	}
	return &facilitiesv1.ListMaintenanceLogsResponse{Logs: out}, nil
}

func (s *FacilitiesGRPCServer) SyncOfflineLogs(ctx context.Context, in *facilitiesv1.SyncOfflineLogsRequest) (*facilitiesv1.SyncOfflineLogsResponse, error) {
	var toSync []*facilities.MaintenanceLog
	for _, l := range in.Logs {
		if l == nil {
			continue
		}
		toSync = append(toSync, &facilities.MaintenanceLog{
			TicketID:      strPtr(l.TicketID),
			AssetID:       l.AssetID,
			TechnicianID:  strPtr(l.TechnicianID),
			QRCodeScanned: l.QRCodeScanned,
			ActionTaken:   l.ActionTaken,
			Notes:         strPtr(l.Notes),
			SyncedAt:      time.Now().UTC(),
		})
	}
	synced, err := s.svc.SyncMaintenanceLogs(ctx, toSync)
	if err != nil {
		return nil, err
	}
	var out []*facilitiesv1.MaintenanceLog
	for _, l := range synced {
		out = append(out, toProtoMaintenanceLog(l))
	}
	return &facilitiesv1.SyncOfflineLogsResponse{
		SyncedCount: int32(len(out)),
		Logs:        out,
	}, nil
}

func (s *FacilitiesGRPCServer) CreateFirmwareRelease(ctx context.Context, in *facilitiesv1.CreateFirmwareReleaseRequest) (*facilitiesv1.CreateFirmwareReleaseResponse, error) {
	rel, err := s.svc.CreateFirmwareRelease(ctx, &facilities.FirmwareRelease{
		DeviceType:         in.DeviceType,
		Version:            in.Version,
		FileURL:            in.FileURL,
		Checksum:           in.Checksum,
		MinHardwareVersion: strPtr(in.MinHardwareVersion),
		Status:             in.Status,
	})
	if err != nil {
		return nil, err
	}
	return &facilitiesv1.CreateFirmwareReleaseResponse{Release: toProtoFirmwareRelease(rel)}, nil
}

func (s *FacilitiesGRPCServer) ListFirmwareReleases(ctx context.Context, in *facilitiesv1.ListFirmwareReleasesRequest) (*facilitiesv1.ListFirmwareReleasesResponse, error) {
	releases, err := s.svc.ListFirmwareReleases(ctx, in.DeviceType)
	if err != nil {
		return nil, err
	}
	var out []*facilitiesv1.FirmwareRelease
	for _, r := range releases {
		out = append(out, toProtoFirmwareRelease(r))
	}
	return &facilitiesv1.ListFirmwareReleasesResponse{Releases: out}, nil
}

func (s *FacilitiesGRPCServer) CreateOTAJob(ctx context.Context, in *facilitiesv1.CreateOTAJobRequest) (*facilitiesv1.CreateOTAJobResponse, error) {
	jobs, err := s.svc.TriggerOTAUpdate(ctx, in.AssetIDs, in.ReleaseID)
	if err != nil {
		return nil, err
	}
	var out []*facilitiesv1.DeviceOTAJob
	for _, j := range jobs {
		out = append(out, toProtoOTAJob(j))
	}
	return &facilitiesv1.CreateOTAJobResponse{Jobs: out}, nil
}

func (s *FacilitiesGRPCServer) ListOTAJobs(ctx context.Context, in *facilitiesv1.ListOTAJobsRequest) (*facilitiesv1.ListOTAJobsResponse, error) {
	jobs, err := s.svc.ListOTAJobs(ctx, in.AssetID)
	if err != nil {
		return nil, err
	}
	var out []*facilitiesv1.DeviceOTAJob
	for _, j := range jobs {
		out = append(out, toProtoOTAJob(j))
	}
	return &facilitiesv1.ListOTAJobsResponse{Jobs: out}, nil
}

func (s *FacilitiesGRPCServer) UpdateOTAJobStatus(ctx context.Context, in *facilitiesv1.UpdateOTAJobStatusRequest) (*facilitiesv1.UpdateOTAJobStatusResponse, error) {
	if s.repo == nil {
		now := time.Now().UTC()
		return &facilitiesv1.UpdateOTAJobStatusResponse{
			Job: &facilitiesv1.DeviceOTAJob{
				JobID:        in.JobID,
				Status:       in.Status,
				ErrorMessage: in.ErrorMessage,
				CompletedAt:  now.Format(time.RFC3339),
			},
		}, nil
	}
	var comp *time.Time
	if in.Status == "completed" || in.Status == "failed" {
		now := time.Now().UTC()
		comp = &now
	}
	j, err := s.repo.UpdateOTAJobStatus(ctx, in.JobID, in.Status, strPtr(in.ErrorMessage), comp)
	if err != nil {
		return nil, err
	}
	return &facilitiesv1.UpdateOTAJobStatusResponse{Job: toProtoOTAJob(j)}, nil
}

func (s *FacilitiesGRPCServer) IngestEnvironmentalTelemetry(ctx context.Context, in *facilitiesv1.IngestTelemetryRequest) (*facilitiesv1.IngestTelemetryResponse, error) {
	payload := map[string]interface{}{
		"zone_id":               in.ZoneID,
		"temperature_c":         in.TemperatureC,
		"relative_humidity_pct": in.RelativeHumidityPct,
		"light_intensity_lux":   in.LightIntensityLux,
		"noise_level_db":        in.NoiseLevelDB,
	}
	res, err := s.svc.IngestEnvironmentalTelemetry(ctx, payload)
	if err != nil {
		return nil, err
	}
	telID, _ := res["telemetry_id"].(string)
	st, _ := res["status"].(string)
	return &facilitiesv1.IngestTelemetryResponse{
		TelemetryID: telID,
		Status:      st,
	}, nil
}

func (s *FacilitiesGRPCServer) StreamEnvironmentalTelemetry(req *facilitiesv1.StreamTelemetryRequest, stream facilitiesv1.FacilitiesService_StreamEnvironmentalTelemetryServer) error {
	zone := req.ZoneID
	if zone == "" {
		zone = "zone-alpha"
	}
	count := req.SampleCount
	if count <= 0 {
		count = 5
	} else if count > 100 {
		count = 100
	}

	samples, err := s.svc.GetLatestEnvironmentalTelemetry(stream.Context(), zone, int(count))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get environmental telemetry: %v", err)
	}
	if len(samples) == 0 {
		return status.Errorf(codes.NotFound, "no environmental telemetry found for zone %s", zone)
	}

	for _, sample := range samples {
		ts := &facilitiesv1.TelemetrySample{
			TelemetryID:         sample.TelemetryID,
			ZoneID:              sample.ZoneID,
			TemperatureC:        sample.TemperatureC,
			RelativeHumidityPct: sample.RelativeHumidityPct,
			LightIntensityLux:   sample.LightIntensityLux,
			NoiseLevelDB:        sample.NoiseLevelDB,
			Timestamp:           sample.RecordedAt.UTC().Format(time.RFC3339),
		}
		if err := stream.Send(ts); err != nil {
			return err
		}
	}
	return nil
}

func (s *FacilitiesGRPCServer) ControlPTZ(ctx context.Context, in *facilitiesv1.ControlPTZRequest) (*facilitiesv1.ControlPTZResponse, error) {
	res, err := s.svc.ControlPTZ(ctx, in.CameraID, map[string]interface{}{
		"pan":  in.Pan,
		"tilt": in.Tilt,
		"zoom": in.Zoom,
	})
	if err != nil {
		return nil, err
	}
	st, _ := res["status"].(string)
	return &facilitiesv1.ControlPTZResponse{
		CameraID: in.CameraID,
		Status:   st,
	}, nil
}

func (s *FacilitiesGRPCServer) CalibrateCameraLens(ctx context.Context, in *facilitiesv1.CalibrateCameraLensRequest) (*facilitiesv1.CalibrateCameraLensResponse, error) {
	res, err := s.svc.CalibrateCameraLens(ctx, in.CameraID)
	if err != nil {
		return nil, err
	}
	st, _ := res["status"].(string)
	return &facilitiesv1.CalibrateCameraLensResponse{
		CameraID: in.CameraID,
		Status:   st,
	}, nil
}
