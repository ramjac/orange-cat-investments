package core_invest

import (
	"context"
	"errors"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
)

func (s *coreInvestService) RegisterCameraStream(ctx context.Context, cameraAssetID, streamURL, protocol string) (*core_invest.CameraStream, error) {
	if cameraAssetID == "" || streamURL == "" {
		return nil, errors.New("camera_asset_id and stream_url are required")
	}
	if protocol != "rtsp" && protocol != "webrtc" {
		return nil, errors.New("protocol must be rtsp or webrtc")
	}

	stream := &core_invest.CameraStream{
		CameraAssetID:  cameraAssetID,
		StreamURL:      streamURL,
		Protocol:       protocol,
		Status:         "active",
		IngestSettings: "{}",
	}

	return s.repo.CreateCameraStream(ctx, stream)
}

func (s *coreInvestService) GetCameraStream(ctx context.Context, id string) (*core_invest.CameraStream, error) {
	if id == "" {
		return nil, errors.New("stream id cannot be empty")
	}
	return s.repo.GetCameraStreamByID(ctx, id)
}

func (s *coreInvestService) ListCameraStreams(ctx context.Context, limit, offset int32) ([]*core_invest.CameraStream, error) {
	return s.repo.ListCameraStreams(ctx, limit, offset)
}

func (s *coreInvestService) StartBacktestRun(ctx context.Context, strategyID string, startDate, endDate time.Time, parameters string) (*core_invest.BacktestRun, error) {
	if strategyID == "" {
		return nil, errors.New("strategy_id is required")
	}
	if endDate.Before(startDate) {
		return nil, errors.New("end_date must be after start_date")
	}
	if parameters == "" {
		parameters = "{}"
	}

	run := &core_invest.BacktestRun{
		StrategyID: strategyID,
		StartDate:  startDate,
		EndDate:    endDate,
		Parameters: parameters,
	}

	createdRun, err := s.repo.CreateBacktestRun(ctx, run)
	if err != nil {
		return nil, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish("events.investment.backtest_requested.v1", map[string]any{
			"backtest_id": createdRun.BacktestID,
			"strategy_id": createdRun.StrategyID,
		})
	}

	return createdRun, nil
}

func (s *coreInvestService) GetBacktestRun(ctx context.Context, id string) (*core_invest.BacktestRun, error) {
	if id == "" {
		return nil, errors.New("backtest id cannot be empty")
	}
	return s.repo.GetBacktestRunByID(ctx, id)
}

func (s *coreInvestService) ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*core_invest.BacktestRun, error) {
	return s.repo.ListBacktestRuns(ctx, limit, offset)
}

func (s *coreInvestService) ListMLModels(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{
			"model_id":   "mdl-yolov8-cat-pose-v3",
			"name":       "CatPose-YOLOv8-Alpha",
			"version":    "v3.2.0",
			"accuracy":   0.9845,
			"drift":      0.012,
			"status":     "active",
			"created_at": time.Now().AddDate(0, -2, 0).Format(time.RFC3339),
		},
	}, nil
}

func (s *coreInvestService) LogModelDrift(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"drift_id":         "drift-018f-99",
		"status":           "recorded",
		"confidence_drift": 0.012,
		"evaluated_at":     time.Now().Format(time.RFC3339),
		"payload":          payload,
	}, nil
}
