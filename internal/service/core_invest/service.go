package core_invest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/core_invest"
)

type EventPublisher interface {
	Publish(topic string, payload any) error
}

type Service interface {
	// Camera Stream Ingestion Pipeline
	RegisterCameraStream(ctx context.Context, cameraAssetID, streamURL, protocol string) (*core_invest.CameraStream, error)
	GetCameraStream(ctx context.Context, id string) (*core_invest.CameraStream, error)
	ListCameraStreams(ctx context.Context, limit, offset int32) ([]*core_invest.CameraStream, error)

	// Automated Brokerage Execution
	ExecuteBrokerageOrder(ctx context.Context, portfolioID, brokerName, symbol, side string, quantity, price float64, strategyID, eventID *string) (*core_invest.BrokerageOrder, error)
	GetBrokerageOrder(ctx context.Context, id string) (*core_invest.BrokerageOrder, error)
	ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*core_invest.BrokerageOrder, error)

	// Dynamic Backtesting Workbench
	StartBacktestRun(ctx context.Context, strategyID string, startDate, endDate time.Time, parameters string) (*core_invest.BacktestRun, error)
	GetBacktestRun(ctx context.Context, id string) (*core_invest.BacktestRun, error)
	ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*core_invest.BacktestRun, error)
}

type coreInvestService struct {
	repo      core_invest.Repository
	publisher EventPublisher
}

func NewService(repo core_invest.Repository) Service {
	return &coreInvestService{repo: repo}
}

func NewServiceWithPublisher(repo core_invest.Repository, publisher EventPublisher) Service {
	return &coreInvestService{repo: repo, publisher: publisher}
}

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

func (s *coreInvestService) ExecuteBrokerageOrder(ctx context.Context, portfolioID, brokerName, symbol, side string, quantity, price float64, strategyID, eventID *string) (*core_invest.BrokerageOrder, error) {
	if portfolioID == "" || brokerName == "" || symbol == "" {
		return nil, errors.New("portfolio_id, broker_name, and symbol are required")
	}
	if side != "buy" && side != "sell" {
		return nil, errors.New("side must be buy or sell")
	}
	if quantity <= 0 || price <= 0 {
		return nil, errors.New("quantity and price must be greater than zero")
	}

	order := &core_invest.BrokerageOrder{
		PortfolioID: portfolioID,
		StrategyID:  strategyID,
		EventID:     eventID,
		BrokerName:  brokerName,
		Symbol:      symbol,
		Side:        side,
		Quantity:    quantity,
		Price:       price,
		Status:      "pending",
	}

	createdOrder, err := s.repo.CreateBrokerageOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Simulate trade execution with brokerage API (FIX/REST)
	now := time.Now().UTC()
	return s.repo.UpdateBrokerageOrderStatus(ctx, createdOrder.OrderID, "executed", &now)
}

func (s *coreInvestService) GetBrokerageOrder(ctx context.Context, id string) (*core_invest.BrokerageOrder, error) {
	if id == "" {
		return nil, errors.New("order id cannot be empty")
	}
	return s.repo.GetBrokerageOrderByID(ctx, id)
}

func (s *coreInvestService) ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*core_invest.BrokerageOrder, error) {
	return s.repo.ListBrokerageOrders(ctx, limit, offset)
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
