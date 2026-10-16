package core_invest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	// Seed initial dev mock fixtures
	repo.portfolios["cust-longterm-arthur"] = &Portfolio{
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

	repo.portfolios["cust-active-chloe"] = &Portfolio{
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
		{
			HoldingID:         "018f3a9a-9999-7000-8000-000000000001",
			PortfolioID:       "018f3a9a-2222-7000-8000-000000000002",
			Symbol:            "AAPL",
			Quantity:          120.0,
			CostBasisUSD:      21420.00,
			AvgPriceUSD:       178.50,
			CurrentPriceUSD:   224.30,
			MarketValueUSD:    26916.00,
			UnrealizedGainUSD: 5496.00,
			UnrealizedGainPct: 25.6583,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			HoldingID:         "018f3a9a-9999-7000-8000-000000000002",
			PortfolioID:       "018f3a9a-2222-7000-8000-000000000002",
			Symbol:            "NVDA",
			Quantity:          85.0,
			CostBasisUSD:      9367.00,
			AvgPriceUSD:       110.20,
			CurrentPriceUSD:   138.80,
			MarketValueUSD:    11798.00,
			UnrealizedGainUSD: 2431.00,
			UnrealizedGainPct: 25.9528,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			HoldingID:         "018f3a9a-9999-7000-8000-000000000003",
			PortfolioID:       "018f3a9a-2222-7000-8000-000000000002",
			Symbol:            "MSFT",
			Quantity:          90.0,
			CostBasisUSD:      34200.00,
			AvgPriceUSD:       380.00,
			CurrentPriceUSD:   448.20,
			MarketValueUSD:    40338.00,
			UnrealizedGainUSD: 6138.00,
			UnrealizedGainPct: 17.9474,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			HoldingID:         "018f3a9a-9999-7000-8000-000000000004",
			PortfolioID:       "018f3a9a-2222-7000-8000-000000000002",
			Symbol:            "TSLA",
			Quantity:          70.0,
			CostBasisUSD:      13300.00,
			AvgPriceUSD:       190.00,
			CurrentPriceUSD:   247.12,
			MarketValueUSD:    17298.40,
			UnrealizedGainUSD: 3998.40,
			UnrealizedGainPct: 30.0632,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
	repo.holdings["018f3a9a-2222-7000-8000-000000000002"] = arthurHoldings
	repo.holdings["cust-longterm-arthur"] = arthurHoldings
	repo.holdings["port-cust-longterm-arthur"] = arthurHoldings
	repo.holdings["018e0000-0000-7000-8000-000000000099"] = arthurHoldings

	chloeHoldings := []*PortfolioHolding{
		{
			HoldingID:         "018f3a9a-9999-7000-8000-000000000005",
			PortfolioID:       "018f3a9a-2222-7000-8000-000000000003",
			Symbol:            "NVDA",
			Quantity:          150.0,
			CostBasisUSD:      16860.00,
			AvgPriceUSD:       112.40,
			CurrentPriceUSD:   138.80,
			MarketValueUSD:    20820.00,
			UnrealizedGainUSD: 3960.00,
			UnrealizedGainPct: 23.4875,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			HoldingID:         "018f3a9a-9999-7000-8000-000000000006",
			PortfolioID:       "018f3a9a-2222-7000-8000-000000000003",
			Symbol:            "TSLA",
			Quantity:          120.0,
			CostBasisUSD:      22260.00,
			AvgPriceUSD:       185.50,
			CurrentPriceUSD:   247.12,
			MarketValueUSD:    29654.40,
			UnrealizedGainUSD: 7394.40,
			UnrealizedGainPct: 33.2183,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			HoldingID:         "018f3a9a-9999-7000-8000-000000000007",
			PortfolioID:       "018f3a9a-2222-7000-8000-000000000003",
			Symbol:            "AAPL",
			Quantity:          85.0,
			CostBasisUSD:      15308.50,
			AvgPriceUSD:       180.10,
			CurrentPriceUSD:   224.30,
			MarketValueUSD:    19065.50,
			UnrealizedGainUSD: 3757.00,
			UnrealizedGainPct: 24.5420,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
	repo.holdings["018f3a9a-2222-7000-8000-000000000003"] = chloeHoldings
	repo.holdings["cust-active-chloe"] = chloeHoldings
	repo.holdings["port-cust-active-chloe"] = chloeHoldings
	repo.holdings["018e0000-0000-7000-8000-000000000098"] = chloeHoldings

	arthurTaxLots := []*TaxLotSale{
		{
			LotID:                  "018f3a9a-aaaa-7000-8000-000000000001",
			PortfolioID:            "018f3a9a-2222-7000-8000-000000000002",
			Symbol:                 "NVDA",
			Description:            "50 shs NVDA",
			Quantity:               50.0,
			DateAcquired:           "2025-02-14",
			DateSold:               "2025-08-20",
			GrossProceedsUSD:       6940.00,
			CostBasisUSD:           5500.00,
			WashSaleLossDisallowed: 0.00,
			RealizedGainLossUSD:    1440.00,
			IsShortTerm:            true,
			TaxYear:                2025,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		{
			LotID:                  "018f3a9a-aaaa-7000-8000-000000000002",
			PortfolioID:            "018f3a9a-2222-7000-8000-000000000002",
			Symbol:                 "AAPL",
			Description:            "35 shs AAPL",
			Quantity:               35.0,
			DateAcquired:           "2025-01-10",
			DateSold:               "2025-09-15",
			GrossProceedsUSD:       7850.00,
			CostBasisUSD:           6200.00,
			WashSaleLossDisallowed: 0.00,
			RealizedGainLossUSD:    1650.00,
			IsShortTerm:            true,
			TaxYear:                2025,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		{
			LotID:                  "018f3a9a-aaaa-7000-8000-000000000003",
			PortfolioID:            "018f3a9a-2222-7000-8000-000000000002",
			Symbol:                 "MSFT",
			Description:            "20 shs MSFT",
			Quantity:               20.0,
			DateAcquired:           "2024-03-01",
			DateSold:               "2025-11-12",
			GrossProceedsUSD:       8960.00,
			CostBasisUSD:           7100.00,
			WashSaleLossDisallowed: 0.00,
			RealizedGainLossUSD:    1860.00,
			IsShortTerm:            false,
			TaxYear:                2025,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
	}
	repo.taxLots["018f3a9a-2222-7000-8000-000000000002"] = arthurTaxLots
	repo.taxLots["cust-longterm-arthur"] = arthurTaxLots
	repo.taxLots["port-cust-longterm-arthur"] = arthurTaxLots

	chloeTaxLots := []*TaxLotSale{
		{
			LotID:                  "018f3a9a-aaaa-7000-8000-000000000004",
			PortfolioID:            "018f3a9a-2222-7000-8000-000000000003",
			Symbol:                 "TSLA",
			Description:            "40 shs TSLA",
			Quantity:               40.0,
			DateAcquired:           "2025-03-15",
			DateSold:               "2025-07-22",
			GrossProceedsUSD:       9884.80,
			CostBasisUSD:           7420.00,
			WashSaleLossDisallowed: 0.00,
			RealizedGainLossUSD:    2464.80,
			IsShortTerm:            true,
			TaxYear:                2025,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		{
			LotID:                  "018f3a9a-aaaa-7000-8000-000000000005",
			PortfolioID:            "018f3a9a-2222-7000-8000-000000000003",
			Symbol:                 "NVDA",
			Description:            "60 shs NVDA",
			Quantity:               60.0,
			DateAcquired:           "2025-02-01",
			DateSold:               "2025-08-10",
			GrossProceedsUSD:       8328.00,
			CostBasisUSD:           6744.00,
			WashSaleLossDisallowed: 0.00,
			RealizedGainLossUSD:    1584.00,
			IsShortTerm:            true,
			TaxYear:                2025,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
	}
	repo.taxLots["018f3a9a-2222-7000-8000-000000000003"] = chloeTaxLots
	repo.taxLots["cust-active-chloe"] = chloeTaxLots
	repo.taxLots["port-cust-active-chloe"] = chloeTaxLots


	repo.webhooks["cust-longterm-arthur"] = []*WebhookSubscription{
		{
			SubscriptionID: "sub-018f-1111",
			CustomerID:     "cust-longterm-arthur",
			TargetURL:      "https://quant-bot.chloespark.io/api/v1/oci-callback",
			Events:         []string{"events.observation.cat_spotted.v1", "events.trade.executed.v1"},
			Status:         "active",
			CreatedAt:      now,
		},
	}

	repo.apiKeys["cust-active-chloe"] = []*CustomerAPIKey{
		{
			KeyID:       "key-018f-8888",
			CustomerID:  "cust-active-chloe",
			Name:        "Chloe Momentum Bot",
			KeyPrefix:   "oci_live_9a8f",
			Status:      "active",
			Permissions: []string{"read_streams", "execute_orders", "read_telemetry"},
			CreatedAt:   now,
		},
	}

	repo.statements["cust-longterm-arthur"] = []*InvestmentStatement{
		{
			StatementID:      "stmt-arthur-2026-03",
			PortfolioID:      "port-cust-longterm-arthur",
			CustomerID:       "cust-longterm-arthur",
			StatementType:    "monthly_summary",
			Period:           "March 2026",
			TaxYear:          2026,
			Title:            "Monthly Portfolio Summary - March 2026",
			FileName:         "monthly-statement-port-cust-longterm-arthur-2026-03.pdf",
			TotalProceedsUSD: 128450.75,
			NetGainUSD:       28450.75,
			Status:           "available",
			CreatedAt:        now.Add(-24 * time.Hour * 15),
		},
		{
			StatementID:      "stmt-arthur-1099b-2025",
			PortfolioID:      "port-cust-longterm-arthur",
			CustomerID:       "cust-longterm-arthur",
			StatementType:    "form_1099b",
			Period:           "Tax Year 2025",
			TaxYear:          2025,
			Title:            "IRS Form 1099-B Tax Report - 2025",
			FileName:         "form1099b-port-cust-longterm-arthur-2025.pdf",
			TotalProceedsUSD: 64200.00,
			NetGainUSD:       14300.00,
			Status:           "available",
			CreatedAt:        now.Add(-24 * time.Hour * 45),
		},
	}

	repo.statements["cust-active-chloe"] = []*InvestmentStatement{
		{
			StatementID:      "stmt-chloe-2026-03",
			PortfolioID:      "port-cust-active-chloe",
			CustomerID:       "cust-active-chloe",
			StatementType:    "monthly_summary",
			Period:           "March 2026",
			TaxYear:          2026,
			Title:            "Monthly Portfolio Summary - March 2026",
			FileName:         "monthly-statement-port-cust-active-chloe-2026-03.pdf",
			TotalProceedsUSD: 85200.00,
			NetGainUSD:       22800.00,
			Status:           "available",
			CreatedAt:        now.Add(-24 * time.Hour * 15),
		},
		{
			StatementID:      "stmt-chloe-1099b-2025",
			PortfolioID:      "port-cust-active-chloe",
			CustomerID:       "cust-active-chloe",
			StatementType:    "form_1099b",
			Period:           "Tax Year 2025",
			TaxYear:          2025,
			Title:            "IRS Form 1099-B Tax Report - 2025",
			FileName:         "form1099b-port-cust-active-chloe-2025.pdf",
			TotalProceedsUSD: 98400.00,
			NetGainUSD:       32100.00,
			Status:           "available",
			CreatedAt:        now.Add(-24 * time.Hour * 45),
		},
	}

	return repo
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

func (m *mockRepository) GetPortfolioByCustomerID(ctx context.Context, customerID string) (*Portfolio, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Direct match
	if p, ok := m.portfolios[customerID]; ok {
		return p, nil
	}

	// Search by CustomerID or PortfolioID
	for _, p := range m.portfolios {
		if p.CustomerID == customerID || p.PortfolioID == customerID {
			return p, nil
		}
	}

	// Dynamic portfolio fallback for unknown / new customers
	now := time.Now().UTC()
	created := &Portfolio{
		PortfolioID:     fmt.Sprintf("port-%s", customerID),
		CustomerID:      customerID,
		AccountName:     "OCI Investment Portfolio",
		TotalBalanceUSD: 0.00,
		CashBalanceUSD:  0.00,
		Status:          "active",
		Holdings:        []map[string]interface{}{},
		HistoricalPerf:  []map[string]interface{}{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	m.portfolios[customerID] = created
	return created, nil
}

func (m *mockRepository) GetPortfolioByID(ctx context.Context, portfolioID string) (*Portfolio, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.portfolios {
		if p.PortfolioID == portfolioID {
			return p, nil
		}
	}
	if p, ok := m.portfolios[portfolioID]; ok {
		return p, nil
	}

	now := time.Now().UTC()
	return &Portfolio{
		PortfolioID:     portfolioID,
		CustomerID:      portfolioID,
		AccountName:     "OCI Investment Portfolio",
		TotalBalanceUSD: 0.00,
		CashBalanceUSD:  0.00,
		Status:          "active",
		Holdings:        []map[string]interface{}{},
		HistoricalPerf:  []map[string]interface{}{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (m *mockRepository) CreatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if p.PortfolioID == "" {
		p.PortfolioID = fmt.Sprintf("port-%d", now.UnixNano())
	}
	p.CreatedAt = now
	p.UpdatedAt = now
	m.portfolios[p.PortfolioID] = p
	m.portfolios[p.CustomerID] = p
	return p, nil
}

func (m *mockRepository) CreateAPIKey(ctx context.Context, key *CustomerAPIKey) (*CustomerAPIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if key.KeyID == "" {
		key.KeyID = fmt.Sprintf("key-%d", now.UnixNano())
	}
	key.CreatedAt = now
	key.Status = "active"
	m.apiKeys[key.CustomerID] = append(m.apiKeys[key.CustomerID], key)
	return key, nil
}

func (m *mockRepository) ListAPIKeys(ctx context.Context, customerID string) ([]*CustomerAPIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*CustomerAPIKey
	for _, k := range m.apiKeys[customerID] {
		if k.Status != "revoked" {
			active = append(active, k)
		}
	}
	return active, nil
}

func (m *mockRepository) RevokeAPIKey(ctx context.Context, customerID, keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := m.apiKeys[customerID]
	for _, k := range keys {
		if k.KeyID == keyID || k.KeyPrefix == keyID {
			k.Status = "revoked"
		}
	}
	return nil
}

func (m *mockRepository) CreateWebhook(ctx context.Context, sub *WebhookSubscription) (*WebhookSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if sub.SubscriptionID == "" {
		sub.SubscriptionID = fmt.Sprintf("sub-%d", now.UnixNano())
	}
	sub.CreatedAt = now
	sub.Status = "active"
	m.webhooks[sub.CustomerID] = append(m.webhooks[sub.CustomerID], sub)
	return sub, nil
}

func (m *mockRepository) ListWebhooks(ctx context.Context, customerID string) ([]*WebhookSubscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*WebhookSubscription
	for _, w := range m.webhooks[customerID] {
		if w.Status != "deleted" {
			active = append(active, w)
		}
	}
	return active, nil
}

func (m *mockRepository) DeleteWebhook(ctx context.Context, customerID, subscriptionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.webhooks[customerID]
	for _, w := range subs {
		if w.SubscriptionID == subscriptionID {
			w.Status = "deleted"
		}
	}
	return nil
}

func (m *mockRepository) UpdatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p.UpdatedAt = time.Now().UTC()
	m.portfolios[p.PortfolioID] = p
	if p.CustomerID != "" {
		m.portfolios[p.CustomerID] = p
	}
	if p.Status == "closed" {
		delete(m.holdings, p.PortfolioID)
		if p.CustomerID != "" {
			delete(m.holdings, p.CustomerID)
		}
	}
	return p, nil
}

func (m *mockRepository) GetHoldingsByPortfolioID(ctx context.Context, portfolioID string) ([]*PortfolioHolding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if list, ok := m.holdings[portfolioID]; ok {
		res := make([]*PortfolioHolding, len(list))
		copy(res, list)
		return res, nil
	}
	for id, p := range m.portfolios {
		if id == portfolioID || p.PortfolioID == portfolioID || p.CustomerID == portfolioID {
			if list, ok := m.holdings[p.PortfolioID]; ok {
				res := make([]*PortfolioHolding, len(list))
				copy(res, list)
				return res, nil
			}
			if list, ok := m.holdings[p.CustomerID]; ok {
				res := make([]*PortfolioHolding, len(list))
				copy(res, list)
				return res, nil
			}
		}
	}
	return []*PortfolioHolding{}, nil
}

func (m *mockRepository) UpsertHolding(ctx context.Context, holding *PortfolioHolding) (*PortfolioHolding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if holding.HoldingID == "" {
		holding.HoldingID = fmt.Sprintf("holding-%d", time.Now().UnixNano())
	}
	now := time.Now().UTC()
	holding.UpdatedAt = now
	if holding.CreatedAt.IsZero() {
		holding.CreatedAt = now
	}

	list := m.holdings[holding.PortfolioID]
	found := false
	for i, h := range list {
		if h.Symbol == holding.Symbol {
			list[i] = holding
			found = true
			break
		}
	}
	if !found {
		list = append(list, holding)
	}
	m.holdings[holding.PortfolioID] = list
	return holding, nil
}

func (m *mockRepository) CreateInvestmentStatement(ctx context.Context, stmt *InvestmentStatement) (*InvestmentStatement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if stmt.StatementID == "" {
		stmt.StatementID = fmt.Sprintf("stmt-%d", now.UnixNano())
	}
	stmt.CreatedAt = now
	stmt.Status = "available"
	m.statements[stmt.CustomerID] = append([]*InvestmentStatement{stmt}, m.statements[stmt.CustomerID]...)
	return stmt, nil
}

func (m *mockRepository) ListInvestmentStatements(ctx context.Context, customerID string) ([]*InvestmentStatement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stmts := m.statements[customerID]
	if stmts == nil {
		return []*InvestmentStatement{}, nil
	}
	return stmts, nil
}

func (m *mockRepository) GetInvestmentStatementByID(ctx context.Context, id string) (*InvestmentStatement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, list := range m.statements {
		for _, s := range list {
			if s.StatementID == id {
				return s, nil
			}
		}
	}
	return nil, fmt.Errorf("investment statement not found: %s", id)
}

func (m *mockRepository) CreateGDPRErasureCertificate(ctx context.Context, cert *GDPRCertificateRecord) (*GDPRCertificateRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if cert.CertificateID == "" {
		cert.CertificateID = fmt.Sprintf("cert-gdpr-%d", now.UnixNano())
	}
	cert.CreatedAt = now
	m.certificates[cert.CertificateID] = cert
	return cert, nil
}

func (m *mockRepository) GetGDPRErasureCertificateByID(ctx context.Context, id string) (*GDPRCertificateRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if cert, ok := m.certificates[id]; ok {
		return cert, nil
	}
	return nil, fmt.Errorf("gdpr certificate not found: %s", id)
}

func (m *mockRepository) ExecuteGDPRErasure(ctx context.Context, customerID string) (*GDPRErasureStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	h := sha256.Sum256([]byte(customerID))
	custHash := hex.EncodeToString(h[:])

	recordsPurged := 0
	tables := []string{}

	if keys, ok := m.apiKeys[customerID]; ok {
		recordsPurged += len(keys)
		delete(m.apiKeys, customerID)
		tables = append(tables, "core_invest.customer_api_keys")
	}

	if subs, ok := m.webhooks[customerID]; ok {
		recordsPurged += len(subs)
		delete(m.webhooks, customerID)
		tables = append(tables, "core_invest.webhook_subscriptions")
	}

	depCount := 0
	for id, dep := range m.deposits {
		if dep.CustomerID == customerID {
			delete(m.deposits, id)
			depCount++
		}
	}
	if depCount > 0 {
		recordsPurged += depCount
		tables = append(tables, "core_invest.recurring_deposits")
	}

	for id, p := range m.portfolios {
		if id == customerID || p.CustomerID == customerID || p.PortfolioID == customerID {
			p.AccountName = fmt.Sprintf("Anonymized Investor (%s)", custHash[:8])
			p.CustomerName = "Anonymized Subject"
			p.Status = "closed"
			p.TotalBalanceUSD = 0.00
			p.CashBalanceUSD = 0.00
			p.Holdings = []map[string]interface{}{}
			p.UpdatedAt = time.Now().UTC()
			delete(m.holdings, p.PortfolioID)
			delete(m.holdings, p.CustomerID)
			delete(m.holdings, id)
			delete(m.taxLots, p.PortfolioID)
			delete(m.taxLots, p.CustomerID)
			delete(m.taxLots, id)
			tables = append(tables, "core_invest.investment_portfolios", "core_invest.portfolio_holdings", "core_invest.tax_lot_sales")
		}
	}

	tables = append(tables, "ops.it_tickets", "ops.ticket_comments")

	return &GDPRErasureStats{
		CustomerIDHash:       custHash,
		CustomerEmailHash:    custHash,
		TablesScrubbed:       tables,
		RecordsScrubbedCount: recordsPurged,
	}, nil
}

func (m *mockRepository) GetTaxLotSalesByPortfolioID(ctx context.Context, portfolioID string, taxYear int) ([]*TaxLotSale, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []*TaxLotSale
	if list, ok := m.taxLots[portfolioID]; ok {
		matched = list
	} else {
		for id, p := range m.portfolios {
			if id == portfolioID || p.PortfolioID == portfolioID || p.CustomerID == portfolioID {
				if list, ok := m.taxLots[p.PortfolioID]; ok {
					matched = list
					break
				}
				if list, ok := m.taxLots[p.CustomerID]; ok {
					matched = list
					break
				}
			}
		}
	}

	var res []*TaxLotSale
	for _, l := range matched {
		if taxYear == 0 || l.TaxYear == taxYear {
			copyLot := *l
			res = append(res, &copyLot)
		}
	}
	if res == nil {
		res = []*TaxLotSale{}
	}
	return res, nil
}

func (m *mockRepository) CreateTaxLotSale(ctx context.Context, lot *TaxLotSale) (*TaxLotSale, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if lot.LotID == "" {
		lot.LotID = fmt.Sprintf("lot-%d", now.UnixNano())
	}
	lot.CreatedAt = now
	lot.UpdatedAt = now
	m.taxLots[lot.PortfolioID] = append(m.taxLots[lot.PortfolioID], lot)
	return lot, nil
}
