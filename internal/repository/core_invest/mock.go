package core_invest

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type mockRepository struct {
	mu           sync.RWMutex
	streams      map[string]*CameraStream
	orders       map[string]*BrokerageOrder
	runs         map[string]*BacktestRun
	deposits     map[string]*RecurringDeposit
	portfolios   map[string]*Portfolio
	apiKeys      map[string][]*CustomerAPIKey
	webhooks     map[string][]*WebhookSubscription
	statements   map[string][]*InvestmentStatement
	certificates map[string]*GDPRCertificateRecord
	holdings     map[string][]*PortfolioHolding
	taxLots      map[string][]*TaxLotSale
}

func NewMockRepository() Repository {
	now := time.Now().UTC()
	repo := &mockRepository{
		streams:      make(map[string]*CameraStream),
		orders:       make(map[string]*BrokerageOrder),
		runs:         make(map[string]*BacktestRun),
		deposits:     make(map[string]*RecurringDeposit),
		portfolios:   make(map[string]*Portfolio),
		apiKeys:      make(map[string][]*CustomerAPIKey),
		webhooks:     make(map[string][]*WebhookSubscription),
		statements:   make(map[string][]*InvestmentStatement),
		certificates: make(map[string]*GDPRCertificateRecord),
		holdings:     make(map[string][]*PortfolioHolding),
		taxLots:      make(map[string][]*TaxLotSale),
	}

	repo.seedPortfoliosAndHoldings(now)
	repo.seedTaxLotsAndStatements(now)
	return repo
}

func (m *mockRepository) seedPortfoliosAndHoldings(now time.Time) {
	m.portfolios["cust-longterm-arthur"] = &Portfolio{
		PortfolioID:     "port-cust-longterm-arthur",
		CustomerID:      "cust-longterm-arthur",
		CustomerName:    "Arthur Pendelton",
		AccountName:     "OCI Alpha Growth Fund",
		TotalBalanceUSD: 128450.75,
		CashBalanceUSD:  32100.25,
		Status:          "active",
		Holdings: []map[string]interface{}{
			{"symbol": "AAPL", "quantity": 120.0, "avg_price": 178.50, "current_price": 224.30, "market_value": 26916.00},
			{"symbol": "NVDA", "quantity": 85.0, "avg_price": 110.20, "current_price": 138.80, "market_value": 11798.00},
			{"symbol": "MSFT", "quantity": 90.0, "avg_price": 380.00, "current_price": 448.20, "market_value": 40338.00},
			{"symbol": "TSLA", "quantity": 70.0, "avg_price": 190.00, "current_price": 247.12, "market_value": 17298.40},
		},
		HistoricalPerf: []map[string]interface{}{
			{"date": "2026-01-01", "valuation": 100000.00},
			{"date": "2026-02-01", "valuation": 108400.00},
			{"date": "2026-03-01", "valuation": 114200.00},
			{"date": "2026-04-01", "valuation": 128450.75},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.portfolios["cust-active-chloe"] = &Portfolio{
		PortfolioID:     "port-cust-active-chloe",
		CustomerID:      "cust-active-chloe",
		CustomerName:    "Chloe Spark",
		AccountName:     "Chloe Spark High-Velocity Alpha",
		TotalBalanceUSD: 85200.00,
		CashBalanceUSD:  15400.00,
		Status:          "active",
		Holdings: []map[string]interface{}{
			{"symbol": "NVDA", "quantity": 150.0, "avg_price": 112.40, "current_price": 138.80, "market_value": 20820.00},
			{"symbol": "TSLA", "quantity": 120.0, "avg_price": 185.50, "current_price": 247.12, "market_value": 29654.40},
			{"symbol": "AAPL", "quantity": 85.0, "avg_price": 180.10, "current_price": 224.30, "market_value": 19065.50},
		},
		HistoricalPerf: []map[string]interface{}{
			{"date": "2026-01-01", "valuation": 50000.00},
			{"date": "2026-02-01", "valuation": 62400.00},
			{"date": "2026-03-01", "valuation": 74100.00},
			{"date": "2026-04-01", "valuation": 85200.00},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	arthurHoldings := []*PortfolioHolding{
		{HoldingID: "018f3a9a-9999-7000-8000-000000000001", PortfolioID: "018f3a9a-2222-7000-8000-000000000002", Symbol: "AAPL", Quantity: 120.0, CostBasisUSD: 21420.00, AvgPriceUSD: 178.50, CurrentPriceUSD: 224.30, MarketValueUSD: 26916.00, UnrealizedGainUSD: 5496.00, UnrealizedGainPct: 25.6583, CreatedAt: now, UpdatedAt: now},
		{HoldingID: "018f3a9a-9999-7000-8000-000000000002", PortfolioID: "018f3a9a-2222-7000-8000-000000000002", Symbol: "NVDA", Quantity: 85.0, CostBasisUSD: 9367.00, AvgPriceUSD: 110.20, CurrentPriceUSD: 138.80, MarketValueUSD: 11798.00, UnrealizedGainUSD: 2431.00, UnrealizedGainPct: 25.9528, CreatedAt: now, UpdatedAt: now},
		{HoldingID: "018f3a9a-9999-7000-8000-000000000003", PortfolioID: "018f3a9a-2222-7000-8000-000000000002", Symbol: "MSFT", Quantity: 90.0, CostBasisUSD: 34200.00, AvgPriceUSD: 380.00, CurrentPriceUSD: 448.20, MarketValueUSD: 40338.00, UnrealizedGainUSD: 6138.00, UnrealizedGainPct: 17.9474, CreatedAt: now, UpdatedAt: now},
		{HoldingID: "018f3a9a-9999-7000-8000-000000000004", PortfolioID: "018f3a9a-2222-7000-8000-000000000002", Symbol: "TSLA", Quantity: 70.0, CostBasisUSD: 13300.00, AvgPriceUSD: 190.00, CurrentPriceUSD: 247.12, MarketValueUSD: 17298.40, UnrealizedGainUSD: 3998.40, UnrealizedGainPct: 30.0632, CreatedAt: now, UpdatedAt: now},
	}
	m.holdings["018f3a9a-2222-7000-8000-000000000002"] = arthurHoldings
	m.holdings["cust-longterm-arthur"] = arthurHoldings
	m.holdings["port-cust-longterm-arthur"] = arthurHoldings
	m.holdings["018e0000-0000-7000-8000-000000000099"] = arthurHoldings

	chloeHoldings := []*PortfolioHolding{
		{HoldingID: "018f3a9a-9999-7000-8000-000000000005", PortfolioID: "018f3a9a-2222-7000-8000-000000000003", Symbol: "NVDA", Quantity: 150.0, CostBasisUSD: 16860.00, AvgPriceUSD: 112.40, CurrentPriceUSD: 138.80, MarketValueUSD: 20820.00, UnrealizedGainUSD: 3960.00, UnrealizedGainPct: 23.4875, CreatedAt: now, UpdatedAt: now},
		{HoldingID: "018f3a9a-9999-7000-8000-000000000006", PortfolioID: "018f3a9a-2222-7000-8000-000000000003", Symbol: "TSLA", Quantity: 120.0, CostBasisUSD: 22260.00, AvgPriceUSD: 185.50, CurrentPriceUSD: 247.12, MarketValueUSD: 29654.40, UnrealizedGainUSD: 7394.40, UnrealizedGainPct: 33.2183, CreatedAt: now, UpdatedAt: now},
		{HoldingID: "018f3a9a-9999-7000-8000-000000000007", PortfolioID: "018f3a9a-2222-7000-8000-000000000003", Symbol: "AAPL", Quantity: 85.0, CostBasisUSD: 15308.50, AvgPriceUSD: 180.10, CurrentPriceUSD: 224.30, MarketValueUSD: 19065.50, UnrealizedGainUSD: 3757.00, UnrealizedGainPct: 24.5420, CreatedAt: now, UpdatedAt: now},
	}
	m.holdings["018f3a9a-2222-7000-8000-000000000003"] = chloeHoldings
	m.holdings["cust-active-chloe"] = chloeHoldings
	m.holdings["port-cust-active-chloe"] = chloeHoldings
	m.holdings["018e0000-0000-7000-8000-000000000098"] = chloeHoldings
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

func (m *mockRepository) CreateRecurringDeposit(ctx context.Context, dep *RecurringDeposit) (*RecurringDeposit, error) {
	now := time.Now().UTC()
	dep.DepositID = "deposit-mock-001"
	dep.CreatedAt = now
	m.deposits[dep.DepositID] = dep
	return dep, nil
}

func (m *mockRepository) ListRecurringDeposits(ctx context.Context, customerID string) ([]*RecurringDeposit, error) {
	var list []*RecurringDeposit
	for _, d := range m.deposits {
		if customerID == "" || d.CustomerID == customerID {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockRepository) GetRecurringDepositByID(ctx context.Context, id string) (*RecurringDeposit, error) {
	if dep, ok := m.deposits[id]; ok {
		return dep, nil
	}
	return nil, fmt.Errorf("recurring deposit not found: %s", id)
}

func (m *mockRepository) AdvanceRecurringDepositSchedule(ctx context.Context, id string, nextChargeAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, ok := m.deposits[id]
	if !ok {
		return fmt.Errorf("recurring deposit not found: %s", id)
	}
	d.NextChargeAt = nextChargeAt
	return nil
}
