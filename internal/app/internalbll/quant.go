package internalbll

import (
	"context"
	"fmt"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListStreams returns active camera streams from core-invest-api.
func (s *Server) ListStreams(ctx context.Context, in *coreinvestv1.ListCameraStreamsRequest) (*coreinvestv1.ListCameraStreamsResponse, error) {
	if s.ciClient == nil {
		return &coreinvestv1.ListCameraStreamsResponse{Streams: []*coreinvestv1.CameraStream{}}, nil
	}
	res, err := s.ciClient.ListCameraStreams(ctx, in)
	if err != nil {
		s.logger.Error("failed to list camera streams from core-invest-api", "error", err)
		return nil, status.Errorf(codes.Internal, "core-invest list streams failed: %v", err)
	}
	return res, nil
}

// RegisterStream registers a new camera stream in core-invest-api.
func (s *Server) RegisterStream(ctx context.Context, in *coreinvestv1.RegisterCameraStreamRequest) (*coreinvestv1.RegisterCameraStreamResponse, error) {
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest service client not configured")
	}
	res, err := s.ciClient.RegisterCameraStream(ctx, in)
	if err != nil {
		s.logger.Error("failed to register camera stream in core-invest-api", "error", err)
		return nil, fmt.Errorf("core-invest register stream failed: %w", err)
	}
	return res, nil
}

// ListBrokerageOrders lists trade orders from core-invest-api.
func (s *Server) ListBrokerageOrders(ctx context.Context, in *coreinvestv1.ListOrdersRequest) (*coreinvestv1.ListOrdersResponse, error) {
	if s.ciClient == nil {
		return &coreinvestv1.ListOrdersResponse{Orders: []*coreinvestv1.BrokerageOrder{}}, nil
	}
	res, err := s.ciClient.ListOrders(ctx, in)
	if err != nil {
		s.logger.Error("failed to list orders from core-invest-api", "error", err)
		return nil, status.Errorf(codes.Internal, "core-invest list orders failed: %v", err)
	}
	return res, nil
}

// ExecuteBrokerageOrder places a brokerage trade order via core-invest-api.
func (s *Server) ExecuteBrokerageOrder(ctx context.Context, in *coreinvestv1.CreateOrderRequest) (*coreinvestv1.CreateOrderResponse, error) {
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest service client not configured")
	}
	res, err := s.ciClient.CreateOrder(ctx, in)
	if err != nil {
		s.logger.Error("failed to create order in core-invest-api", "error", err)
		return nil, fmt.Errorf("core-invest create order failed: %w", err)
	}
	return res, nil
}

// ListBacktestRuns lists backtest runs from core-invest-api.
func (s *Server) ListBacktestRuns(ctx context.Context, in *coreinvestv1.ListBacktestRunsRequest) (*coreinvestv1.ListBacktestRunsResponse, error) {
	if s.ciClient == nil {
		return &coreinvestv1.ListBacktestRunsResponse{Runs: []*coreinvestv1.BacktestRun{}}, nil
	}
	res, err := s.ciClient.ListBacktestRuns(ctx, in)
	if err != nil {
		s.logger.Error("failed to list backtest runs from core-invest-api", "error", err)
		return nil, status.Errorf(codes.Internal, "core-invest list backtest runs failed: %v", err)
	}
	return res, nil
}

// StartBacktestRun initiates a backtesting execution in core-invest-api.
func (s *Server) StartBacktestRun(ctx context.Context, in *coreinvestv1.StartBacktestRunRequest) (*coreinvestv1.StartBacktestRunResponse, error) {
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest service client not configured")
	}
	res, err := s.ciClient.StartBacktestRun(ctx, in)
	if err != nil {
		s.logger.Error("failed to start backtest run in core-invest-api", "error", err)
		return nil, fmt.Errorf("core-invest start backtest run failed: %w", err)
	}
	return res, nil
}

// ListRecurringDeposits lists recurring DCA schedules from core-invest-api.
func (s *Server) ListRecurringDeposits(ctx context.Context, in *coreinvestv1.ListRecurringDepositsRequest) (*coreinvestv1.ListRecurringDepositsResponse, error) {
	if s.ciClient == nil {
		return &coreinvestv1.ListRecurringDepositsResponse{Deposits: []*coreinvestv1.RecurringDeposit{}}, nil
	}
	res, err := s.ciClient.ListRecurringDeposits(ctx, in)
	if err != nil {
		s.logger.Error("failed to list recurring deposits from core-invest-api", "error", err)
		return nil, status.Errorf(codes.Internal, "core-invest list recurring deposits failed: %v", err)
	}
	return res, nil
}

// CreateRecurringDeposit schedules a new recurring deposit in core-invest-api.
func (s *Server) CreateRecurringDeposit(ctx context.Context, in *coreinvestv1.CreateRecurringDepositRequest) (*coreinvestv1.CreateRecurringDepositResponse, error) {
	if s.ciClient == nil {
		return nil, status.Error(codes.Unavailable, "core-invest service client not configured")
	}
	res, err := s.ciClient.CreateRecurringDeposit(ctx, in)
	if err != nil {
		s.logger.Error("failed to create recurring deposit in core-invest-api", "error", err)
		return nil, fmt.Errorf("core-invest create recurring deposit failed: %w", err)
	}
	return res, nil
}
