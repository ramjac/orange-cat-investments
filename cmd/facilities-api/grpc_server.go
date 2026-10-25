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
	svc facilities_svc.Service
}

func NewFacilitiesGRPCServer(svc facilities_svc.Service) *FacilitiesGRPCServer {
	return &FacilitiesGRPCServer{svc: svc}
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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
	var notesPtr *string
	if in.Notes != "" {
		notesPtr = &in.Notes
	}
	var techPtr *string
	if in.TechnicianID != "" {
		techPtr = &in.TechnicianID
	}
	log := &facilities.MaintenanceLog{
		AssetID:       in.AssetID,
		TechnicianID:  techPtr,
		QRCodeScanned: in.QRCodeScanned,
		ActionTaken:   in.ActionTaken,
		Notes:         notesPtr,
		SyncedAt:      time.Now().UTC(),
	}
	synced, err := s.svc.SyncMaintenanceLogs(ctx, []*facilities.MaintenanceLog{log})
	if err != nil {
		return nil, err
	}
	if len(synced) == 0 {
		return nil, fmt.Errorf("no log synced")
	}
	resLog := synced[0]
	return &facilitiesv1.CreateMaintenanceLogResponse{Log: &facilitiesv1.MaintenanceLog{
		LogID:         resLog.LogID,
		AssetID:       resLog.AssetID,
		TechnicianID:  strVal(resLog.TechnicianID),
		QRCodeScanned: resLog.QRCodeScanned,
		ActionTaken:   resLog.ActionTaken,
		Notes:         strVal(resLog.Notes),
		SyncedAt:      resLog.SyncedAt.Format(time.RFC3339),
	}}, nil
}

func (s *FacilitiesGRPCServer) ListMaintenanceLogs(ctx context.Context, in *facilitiesv1.ListMaintenanceLogsRequest) (*facilitiesv1.ListMaintenanceLogsResponse, error) {
	return &facilitiesv1.ListMaintenanceLogsResponse{}, nil
}

func (s *FacilitiesGRPCServer) ListFirmwareReleases(ctx context.Context, in *facilitiesv1.ListFirmwareReleasesRequest) (*facilitiesv1.ListFirmwareReleasesResponse, error) {
	releases, err := s.svc.ListFirmwareReleases(ctx, in.DeviceType)
	if err != nil {
		return nil, err
	}
	var out []*facilitiesv1.FirmwareRelease
	for _, r := range releases {
		out = append(out, &facilitiesv1.FirmwareRelease{
			ReleaseID:          r.ReleaseID,
			DeviceType:         r.DeviceType,
			Version:            r.Version,
			FileURL:            r.FileURL,
			Checksum:           r.Checksum,
			MinHardwareVersion: strVal(r.MinHardwareVersion),
			Status:             r.Status,
			CreatedAt:          r.CreatedAt.Format(time.RFC3339),
		})
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
		out = append(out, &facilitiesv1.DeviceOTAJob{
			JobID:        j.JobID,
			AssetID:      j.AssetID,
			ReleaseID:    j.ReleaseID,
			Status:       j.Status,
			ErrorMessage: strVal(j.ErrorMessage),
			ScheduledAt:  j.ScheduledAt.Format(time.RFC3339),
		})
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
		out = append(out, &facilitiesv1.DeviceOTAJob{
			JobID:        j.JobID,
			AssetID:      j.AssetID,
			ReleaseID:    j.ReleaseID,
			Status:       j.Status,
			ErrorMessage: strVal(j.ErrorMessage),
			ScheduledAt:  j.ScheduledAt.Format(time.RFC3339),
		})
	}
	return &facilitiesv1.ListOTAJobsResponse{Jobs: out}, nil
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
	status, _ := res["status"].(string)
	return &facilitiesv1.IngestTelemetryResponse{
		TelemetryID: telID,
		Status:      status,
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
