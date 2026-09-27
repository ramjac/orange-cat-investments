package core_invest

import (
	"context"
	"time"
)

type mockRepository struct {
	streams map[string]*CameraStream
	orders  map[string]*BrokerageOrder
	runs    map[string]*BacktestRun
}

func NewMockRepository() Repository {
	return &mockRepository{
		streams: make(map[string]*CameraStream),
		orders:  make(map[string]*BrokerageOrder),
		runs:    make(map[string]*BacktestRun),
	}
}

func (m *mockRepository) GetCameraStreamByID(ctx context.Context, id string) (*CameraStream, error) {
	if stream, ok := m.streams[id]; ok {
		return stream, nil
	}
	now := time.Now().UTC()
	return &CameraStream{
		StreamID:      id,
		CameraAssetID: "cam-asset-001",
		StreamURL:     "rtsp://edge-camera-01.local/live",
		Protocol:      "rtsp",
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *mockRepository) ListCameraStreams(ctx context.Context, limit, offset int32) ([]*CameraStream, error) {
	var list []*CameraStream
	for _, s := range m.streams {
		list = append(list, s)
	}
	if len(list) == 0 {
		now := time.Now().UTC()
		list = append(list, &CameraStream{
			StreamID:      "stream-uuid-001",
			CameraAssetID: "cam-asset-001",
			StreamURL:     "rtsp://edge-camera-01.local/live",
			Protocol:      "rtsp",
			Status:        "active",
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return list, nil
}

func (m *mockRepository) CreateCameraStream(ctx context.Context, stream *CameraStream) (*CameraStream, error) {
	now := time.Now().UTC()
	stream.StreamID = "stream-uuid-created"
	stream.CreatedAt = now
	stream.UpdatedAt = now
	m.streams[stream.StreamID] = stream
	return stream, nil
}

func (m *mockRepository) GetBrokerageOrderByID(ctx context.Context, id string) (*BrokerageOrder, error) {
	if order, ok := m.orders[id]; ok {
		return order, nil
	}
	now := time.Now().UTC()
	return &BrokerageOrder{
		OrderID:     id,
		PortfolioID: "portfolio-001",
		BrokerName:  "InteractiveBrokers",
		Symbol:      "ORNG",
		Side:        "buy",
		Quantity:    100.0,
		Price:       42.50,
		Status:      "executed",
		ExecutedAt:  &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (m *mockRepository) ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*BrokerageOrder, error) {
	var list []*BrokerageOrder
	for _, o := range m.orders {
		list = append(list, o)
	}
	if len(list) == 0 {
		now := time.Now().UTC()
		list = append(list, &BrokerageOrder{
			OrderID:     "order-uuid-001",
			PortfolioID: "portfolio-001",
			BrokerName:  "InteractiveBrokers",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    100.0,
			Price:       42.50,
			Status:      "executed",
			ExecutedAt:  &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return list, nil
}

func (m *mockRepository) CreateBrokerageOrder(ctx context.Context, order *BrokerageOrder) (*BrokerageOrder, error) {
	now := time.Now().UTC()
	order.OrderID = "order-uuid-created"
	order.CreatedAt = now
	order.UpdatedAt = now
	m.orders[order.OrderID] = order
	return order, nil
}

func (m *mockRepository) UpdateBrokerageOrderStatus(ctx context.Context, id string, status string, executedAt *time.Time) (*BrokerageOrder, error) {
	order, ok := m.orders[id]
	if !ok {
		now := time.Now().UTC()
		order = &BrokerageOrder{
			OrderID:     id,
			PortfolioID: "portfolio-001",
			BrokerName:  "InteractiveBrokers",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    100.0,
			Price:       42.50,
			CreatedAt:   now,
		}
	}
	order.Status = status
	order.ExecutedAt = executedAt
	order.UpdatedAt = time.Now().UTC()
	m.orders[id] = order
	return order, nil
}

func (m *mockRepository) GetBacktestRunByID(ctx context.Context, id string) (*BacktestRun, error) {
	if run, ok := m.runs[id]; ok {
		return run, nil
	}
	now := time.Now().UTC()
	return &BacktestRun{
		BacktestID: id,
		StrategyID: "strategy-001",
		StartDate:  now.Add(-24 * time.Hour * 30),
		EndDate:    now,
		Parameters: "{}",
		Results:    `{"sharpe_ratio":1.85, "alpha":0.12}`,
		Status:     "completed",
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (m *mockRepository) ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*BacktestRun, error) {
	var list []*BacktestRun
	for _, r := range m.runs {
		list = append(list, r)
	}
	if len(list) == 0 {
		now := time.Now().UTC()
		list = append(list, &BacktestRun{
			BacktestID: "backtest-uuid-001",
			StrategyID: "strategy-001",
			StartDate:  now.Add(-24 * time.Hour * 30),
			EndDate:    now,
			Parameters: "{}",
			Results:    `{"sharpe_ratio":1.85, "alpha":0.12}`,
			Status:     "completed",
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	return list, nil
}

func (m *mockRepository) CreateBacktestRun(ctx context.Context, run *BacktestRun) (*BacktestRun, error) {
	now := time.Now().UTC()
	run.BacktestID = "backtest-uuid-created"
	run.Status = "pending"
	run.CreatedAt = now
	run.UpdatedAt = now
	m.runs[run.BacktestID] = run
	return run, nil
}

func (m *mockRepository) UpdateBacktestRunResults(ctx context.Context, id string, status string, results string) (*BacktestRun, error) {
	run, ok := m.runs[id]
	if !ok {
		now := time.Now().UTC()
		run = &BacktestRun{
			BacktestID: id,
			StrategyID: "strategy-001",
			StartDate:  now.Add(-24 * time.Hour * 30),
			EndDate:    now,
			CreatedAt:  now,
		}
	}
	run.Status = status
	run.Results = results
	run.UpdatedAt = time.Now().UTC()
	m.runs[id] = run
	return run, nil
}
