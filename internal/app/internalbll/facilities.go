package internalbll

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetAsset(ctx context.Context, in *facilitiesv1.GetAssetRequest) (*facilitiesv1.GetAssetResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	res, err := s.facClient.GetAsset(ctx, in)
	return res, s.logDomainError("facilities-api", "GetAsset", err)
}

func (s *Server) ListAssets(ctx context.Context, in *facilitiesv1.ListAssetsRequest) (*facilitiesv1.ListAssetsResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListAssetsResponse{}, nil
	}
	res, err := s.facClient.ListAssets(ctx, in)
	return res, s.logDomainError("facilities-api", "ListAssets", err)
}

func (s *Server) CreateAsset(ctx context.Context, in *facilitiesv1.CreateAssetRequest) (*facilitiesv1.CreateAssetResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	res, err := s.facClient.CreateAsset(ctx, in)
	return res, s.logDomainError("facilities-api", "CreateAsset", err)
}

func (s *Server) CreateMaintenanceLog(ctx context.Context, in *facilitiesv1.CreateMaintenanceLogRequest) (*facilitiesv1.CreateMaintenanceLogResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	res, err := s.facClient.CreateMaintenanceLog(ctx, in)
	return res, s.logDomainError("facilities-api", "CreateMaintenanceLog", err)
}

func (s *Server) ListMaintenanceLogs(ctx context.Context, in *facilitiesv1.ListMaintenanceLogsRequest) (*facilitiesv1.ListMaintenanceLogsResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListMaintenanceLogsResponse{}, nil
	}
	res, err := s.facClient.ListMaintenanceLogs(ctx, in)
	return res, s.logDomainError("facilities-api", "ListMaintenanceLogs", err)
}

func (s *Server) SyncOfflineLogs(ctx context.Context, in *facilitiesv1.SyncOfflineLogsRequest) (*facilitiesv1.SyncOfflineLogsResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	res, err := s.facClient.SyncOfflineLogs(ctx, in)
	return res, s.logDomainError("facilities-api", "SyncOfflineLogs", err)
}

func (s *Server) CreateFirmwareRelease(ctx context.Context, in *facilitiesv1.CreateFirmwareReleaseRequest) (*facilitiesv1.CreateFirmwareReleaseResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	res, err := s.facClient.CreateFirmwareRelease(ctx, in)
	return res, s.logDomainError("facilities-api", "CreateFirmwareRelease", err)
}

func (s *Server) ListFirmwareReleases(ctx context.Context, in *facilitiesv1.ListFirmwareReleasesRequest) (*facilitiesv1.ListFirmwareReleasesResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListFirmwareReleasesResponse{}, nil
	}
	res, err := s.facClient.ListFirmwareReleases(ctx, in)
	return res, s.logDomainError("facilities-api", "ListFirmwareReleases", err)
}

func (s *Server) TriggerOTARollout(ctx context.Context, in *facilitiesv1.CreateOTAJobRequest) (*facilitiesv1.CreateOTAJobResponse, error) {
	if len(in.AssetIDs) == 0 || in.ReleaseID == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_ids and release_id are required")
	}
	if s.facClient == nil {
		var jobs []*facilitiesv1.DeviceOTAJob
		for i, aid := range in.AssetIDs {
			jobID := fmt.Sprintf("ota-job-%d-%d", time.Now().UnixNano(), i)
			jobs = append(jobs, &facilitiesv1.DeviceOTAJob{
				JobID:       jobID,
				AssetID:     aid,
				ReleaseID:   in.ReleaseID,
				Status:      "queued",
				ScheduledAt: time.Now().UTC().Format(time.RFC3339),
			})
			if s.publisher != nil {
				_ = s.publisher.Publish("events.facilities.ota_triggered.v1", envelope.NewEventEnvelope(ctx,
					fmt.Sprintf("evt-ota-%d-%d", time.Now().UnixNano(), i),
					"events.facilities.ota_triggered.v1",
					"",
					envelope.OTATriggeredPayload{
						JobID:     jobID,
						AssetID:   aid,
						ReleaseID: in.ReleaseID,
					},
				))
			}
		}
		return &facilitiesv1.CreateOTAJobResponse{Jobs: jobs}, nil
	}

	resp, err := s.facClient.CreateOTAJob(ctx, in)
	if err != nil {
		return nil, s.logDomainError("facilities-api", "CreateOTAJob", err)
	}
	// OTA rollout events (including Cosign signature metadata) are committed atomically
	// in facilities.outbox_events by facilities-api and relayed by oci-worker (Phase 10 / 7.40).
	return resp, nil
}

func (s *Server) ListOTAJobs(ctx context.Context, in *facilitiesv1.ListOTAJobsRequest) (*facilitiesv1.ListOTAJobsResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListOTAJobsResponse{}, nil
	}
	res, err := s.facClient.ListOTAJobs(ctx, in)
	return res, s.logDomainError("facilities-api", "ListOTAJobs", err)
}

func (s *Server) UpdateOTAJobStatus(ctx context.Context, in *facilitiesv1.UpdateOTAJobStatusRequest) (*facilitiesv1.UpdateOTAJobStatusResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	res, err := s.facClient.UpdateOTAJobStatus(ctx, in)
	return res, s.logDomainError("facilities-api", "UpdateOTAJobStatus", err)
}

func (s *Server) IngestEnvironmentalTelemetry(ctx context.Context, in *facilitiesv1.IngestTelemetryRequest) (*facilitiesv1.IngestTelemetryResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	res, err := s.facClient.IngestEnvironmentalTelemetry(ctx, in)
	return res, s.logDomainError("facilities-api", "IngestEnvironmentalTelemetry", err)
}

func (s *Server) StreamEnvironmentalTelemetry(in *facilitiesv1.StreamTelemetryRequest, stream facilitiesv1.FacilitiesService_StreamEnvironmentalTelemetryServer) error {
	if s.facClient == nil {
		return status.Error(codes.Unavailable, "facilities client not configured")
	}
	upstream, err := s.facClient.StreamEnvironmentalTelemetry(stream.Context(), in)
	if err != nil {
		return err
	}
	for {
		sample, err := upstream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(sample); err != nil {
			return err
		}
	}
}

func (s *Server) ControlPTZ(ctx context.Context, in *facilitiesv1.ControlPTZRequest) (*facilitiesv1.ControlPTZResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.ControlPTZ(ctx, in)
}

func (s *Server) CalibrateCameraLens(ctx context.Context, in *facilitiesv1.CalibrateCameraLensRequest) (*facilitiesv1.CalibrateCameraLensResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.CalibrateCameraLens(ctx, in)
}

func (s *Server) ReportDeviceHeartbeat(ctx context.Context, in *internalbllv1.ReportDeviceHeartbeatRequest) (*internalbllv1.ReportDeviceHeartbeatResponse, error) {
	if in == nil || in.SerialNumber == "" {
		return nil, status.Error(codes.InvalidArgument, "serial_number is required")
	}
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}

	hbResp, err := s.facClient.RecordAssetHeartbeat(ctx, &facilitiesv1.RecordAssetHeartbeatRequest{
		SerialNumber: in.SerialNumber,
		ObservedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, s.logDomainError("facilities-api", "RecordAssetHeartbeat", err)
	}

	var stream *coreinvestv1.CameraStream
	if in.CameraStatus != "" {
		if s.ciClient == nil {
			return nil, status.Error(codes.Unavailable, "core_invest client not configured")
		}
		if hbResp.Asset == nil || hbResp.Asset.AssetID == "" {
			return nil, status.Error(codes.Internal, "facilities RecordAssetHeartbeat returned empty asset")
		}
		streamResp, err := s.ciClient.UpdateCameraStreamStatus(ctx, &coreinvestv1.UpdateCameraStreamStatusRequest{
			CameraAssetID: hbResp.Asset.AssetID,
			Status:        in.CameraStatus,
		})
		if err != nil {
			return nil, s.logDomainError("core-invest-api", "UpdateCameraStreamStatus", err)
		}
		stream = streamResp.Stream
	}

	return &internalbllv1.ReportDeviceHeartbeatResponse{
		Asset:  hbResp.Asset,
		Stream: stream,
	}, nil
}

func (s *Server) ListDeviceConnectivity(ctx context.Context, in *internalbllv1.ListDeviceConnectivityRequest) (*internalbllv1.ListDeviceConnectivityResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	var zoneID string
	if in != nil {
		zoneID = in.ZoneID
	}
	resp, err := s.facClient.ListAssetConnectivity(ctx, &facilitiesv1.ListAssetConnectivityRequest{
		ZoneID: zoneID,
	})
	if err != nil {
		return nil, s.logDomainError("facilities-api", "ListAssetConnectivity", err)
	}
	return &internalbllv1.ListDeviceConnectivityResponse{
		Assets: resp.Assets,
	}, nil
}
