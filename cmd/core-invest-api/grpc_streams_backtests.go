package main

import (
	"context"
	"fmt"
	"time"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *CoreInvestGRPCServer) RegisterCameraStream(ctx context.Context, in *coreinvestv1.RegisterCameraStreamRequest) (*coreinvestv1.RegisterCameraStreamResponse, error) {
	cs, err := s.svc.RegisterCameraStream(ctx, in.AssetID, in.StreamURL, in.Protocol)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.RegisterCameraStreamResponse{Stream: toProtoCameraStream(cs)}, nil
}

func (s *CoreInvestGRPCServer) GetCameraStream(ctx context.Context, in *coreinvestv1.GetCameraStreamRequest) (*coreinvestv1.GetCameraStreamResponse, error) {
	cs, err := s.svc.GetCameraStream(ctx, in.StreamID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetCameraStreamResponse{Stream: toProtoCameraStream(cs)}, nil
}

func (s *CoreInvestGRPCServer) ListCameraStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	streams, err := s.svc.ListCameraStreams(ctx, 50, 0)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.CameraStream
	for _, cs := range streams {
		if in.Status != "" && cs.Status != in.Status {
			continue
		}
		out = append(out, toProtoCameraStream(cs))
	}
	return &coreinvestv1.ListCameraStreamsResponse{Streams: out}, nil
}

func (s *CoreInvestGRPCServer) IngestObservation(ctx context.Context, in *coreinvestv1.IngestObservationRequest) (*coreinvestv1.IngestObservationResponse, error) {
	return &coreinvestv1.IngestObservationResponse{
		EventID: fmt.Sprintf("obs-%d", time.Now().UnixNano()),
		Status:  "ingested",
	}, nil
}

func (s *CoreInvestGRPCServer) ListObservations(ctx context.Context, in *coreinvestv1.ListObservationsRequest) (*coreinvestv1.ListObservationsResponse, error) {
	return &coreinvestv1.ListObservationsResponse{}, nil
}

func (s *CoreInvestGRPCServer) StartBacktestRun(ctx context.Context, in *coreinvestv1.StartBacktestRunRequest) (*coreinvestv1.StartBacktestRunResponse, error) {
	start, err := time.Parse("2006-01-02", in.StartDate)
	if err != nil {
		start = time.Now().UTC().AddDate(-1, 0, 0)
	}
	end, err := time.Parse("2006-01-02", in.EndDate)
	if err != nil {
		end = time.Now().UTC()
	}
	run, err := s.svc.StartBacktestRun(ctx, in.StrategyName, start, end, fmt.Sprintf(`{"initial_capital":%f}`, in.InitialCapital))
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.StartBacktestRunResponse{Run: toProtoBacktestRun(run)}, nil
}

func (s *CoreInvestGRPCServer) GetBacktestRun(ctx context.Context, in *coreinvestv1.GetBacktestRunRequest) (*coreinvestv1.GetBacktestRunResponse, error) {
	run, err := s.svc.GetBacktestRun(ctx, in.RunID)
	if err != nil {
		return nil, err
	}
	return &coreinvestv1.GetBacktestRunResponse{Run: toProtoBacktestRun(run)}, nil
}

func (s *CoreInvestGRPCServer) ListBacktestRuns(ctx context.Context, in *coreinvestv1.ListBacktestRunsRequest) (*coreinvestv1.ListBacktestRunsResponse, error) {
	runs, err := s.svc.ListBacktestRuns(ctx, 50, 0)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.BacktestRun
	for _, r := range runs {
		out = append(out, toProtoBacktestRun(r))
	}
	return &coreinvestv1.ListBacktestRunsResponse{Runs: out}, nil
}

func (s *CoreInvestGRPCServer) ListMLModels(ctx context.Context, in *coreinvestv1.ListMLModelsRequest) (*coreinvestv1.ListMLModelsResponse, error) {
	models, err := s.svc.ListMLModels(ctx)
	if err != nil {
		return nil, err
	}
	var out []*coreinvestv1.MLModelInfo
	for _, m := range models {
		id, _ := m["model_id"].(string)
		name, _ := m["name"].(string)
		ver, _ := m["version"].(string)
		fw, _ := m["framework"].(string)
		st, _ := m["status"].(string)
		acc, _ := m["accuracy"].(float64)
		out = append(out, &coreinvestv1.MLModelInfo{
			ModelID:   id,
			Name:      name,
			Version:   ver,
			Framework: fw,
			Status:    st,
			Accuracy:  acc,
		})
	}
	return &coreinvestv1.ListMLModelsResponse{Models: out}, nil
}

func (s *CoreInvestGRPCServer) LogModelDrift(ctx context.Context, in *coreinvestv1.LogModelDriftRequest) (*coreinvestv1.LogModelDriftResponse, error) {
	res, err := s.svc.LogModelDrift(ctx, map[string]interface{}{
		"model_id":       in.ModelID,
		"psi_score":      in.PSIScore,
		"accuracy_delta": in.AccuracyDelta,
		"notes":          in.Notes,
	})
	if err != nil {
		return nil, err
	}
	driftID, _ := res["drift_id"].(string)
	st, _ := res["status"].(string)
	return &coreinvestv1.LogModelDriftResponse{
		DriftID:  driftID,
		ModelID:  in.ModelID,
		Status:   st,
		LoggedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *CoreInvestGRPCServer) StreamPortfolioValuation(req *coreinvestv1.StreamPortfolioValuationRequest, stream coreinvestv1.CoreInvestService_StreamPortfolioValuationServer) error {
	if req.PortfolioID == "" {
		return status.Error(codes.InvalidArgument, "portfolio_id is required")
	}

	p, err := s.svc.GetPortfolio(stream.Context(), req.PortfolioID)
	if err != nil || p == nil {
		return status.Errorf(codes.NotFound, "portfolio %s not found: %v", req.PortfolioID, err)
	}

	count := req.UpdateCount
	if count <= 0 {
		count = 5
	}

	invested := p.TotalBalanceUSD - p.CashBalanceUSD
	changePct := 0.0
	if p.TotalBalanceUSD > 0 && invested > 0 {
		changePct = (invested / p.TotalBalanceUSD) * 100.0
	}

	for i := 0; i < int(count); i++ {
		if i > 0 {
			if latest, err := s.svc.GetPortfolio(stream.Context(), req.PortfolioID); err == nil && latest != nil {
				p = latest
				invested = p.TotalBalanceUSD - p.CashBalanceUSD
				if p.TotalBalanceUSD > 0 && invested > 0 {
					changePct = (invested / p.TotalBalanceUSD) * 100.0
				}
			}
		}

		err := stream.Send(&coreinvestv1.PortfolioValuationEvent{
			PortfolioID:        p.PortfolioID,
			TotalBalanceUSD:    p.TotalBalanceUSD,
			CashBalanceUSD:     p.CashBalanceUSD,
			InvestedBalanceUSD: invested,
			ChangePct:          changePct,
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
