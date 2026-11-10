package internalbll

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetAsset(ctx context.Context, in *facilitiesv1.GetAssetRequest) (*facilitiesv1.GetAssetResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.GetAsset(ctx, in)
}

func (s *Server) ListAssets(ctx context.Context, in *facilitiesv1.ListAssetsRequest) (*facilitiesv1.ListAssetsResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListAssetsResponse{}, nil
	}
	return s.facClient.ListAssets(ctx, in)
}

func (s *Server) CreateAsset(ctx context.Context, in *facilitiesv1.CreateAssetRequest) (*facilitiesv1.CreateAssetResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.CreateAsset(ctx, in)
}

func (s *Server) CreateMaintenanceLog(ctx context.Context, in *facilitiesv1.CreateMaintenanceLogRequest) (*facilitiesv1.CreateMaintenanceLogResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.CreateMaintenanceLog(ctx, in)
}

func (s *Server) ListMaintenanceLogs(ctx context.Context, in *facilitiesv1.ListMaintenanceLogsRequest) (*facilitiesv1.ListMaintenanceLogsResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListMaintenanceLogsResponse{}, nil
	}
	return s.facClient.ListMaintenanceLogs(ctx, in)
}

func (s *Server) SyncOfflineLogs(ctx context.Context, in *facilitiesv1.SyncOfflineLogsRequest) (*facilitiesv1.SyncOfflineLogsResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.SyncOfflineLogs(ctx, in)
}

func (s *Server) CreateFirmwareRelease(ctx context.Context, in *facilitiesv1.CreateFirmwareReleaseRequest) (*facilitiesv1.CreateFirmwareReleaseResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.CreateFirmwareRelease(ctx, in)
}

func (s *Server) ListFirmwareReleases(ctx context.Context, in *facilitiesv1.ListFirmwareReleasesRequest) (*facilitiesv1.ListFirmwareReleasesResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListFirmwareReleasesResponse{}, nil
	}
	return s.facClient.ListFirmwareReleases(ctx, in)
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
				_ = s.publisher.Publish("events.facilities.ota_triggered.v1", envelope.NewEventEnvelope(
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
		return nil, err
	}
	if s.publisher != nil {
		for _, job := range resp.Jobs {
			_ = s.publisher.Publish("events.facilities.ota_triggered.v1", envelope.NewEventEnvelope(
				fmt.Sprintf("evt-ota-%d", time.Now().UnixNano()),
				"events.facilities.ota_triggered.v1",
				"",
				envelope.OTATriggeredPayload{
					JobID:     job.JobID,
					AssetID:   job.AssetID,
					ReleaseID: job.ReleaseID,
				},
			))
		}
	}
	return resp, nil
}

func (s *Server) ListOTAJobs(ctx context.Context, in *facilitiesv1.ListOTAJobsRequest) (*facilitiesv1.ListOTAJobsResponse, error) {
	if s.facClient == nil {
		return &facilitiesv1.ListOTAJobsResponse{}, nil
	}
	return s.facClient.ListOTAJobs(ctx, in)
}

func (s *Server) UpdateOTAJobStatus(ctx context.Context, in *facilitiesv1.UpdateOTAJobStatusRequest) (*facilitiesv1.UpdateOTAJobStatusResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.UpdateOTAJobStatus(ctx, in)
}

func (s *Server) IngestEnvironmentalTelemetry(ctx context.Context, in *facilitiesv1.IngestTelemetryRequest) (*facilitiesv1.IngestTelemetryResponse, error) {
	if s.facClient == nil {
		return nil, status.Error(codes.Unavailable, "facilities client not configured")
	}
	return s.facClient.IngestEnvironmentalTelemetry(ctx, in)
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
