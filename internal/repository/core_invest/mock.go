package core_invest

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type mockRepository struct {
	mu         sync.RWMutex
	streams    map[string]*CameraStream
	orders     map[string]*BrokerageOrder
	runs       map[string]*BacktestRun
	deposits   map[string]*RecurringDeposit
	portfolios map[string]*Portfolio
	apiKeys    map[string][]*CustomerAPIKey
	webhooks   map[string][]*WebhookSubscription
}

func NewMockRepository() Repository {
	now := time.Now().UTC()
	repo := &mockRepository{
		streams:    make(map[string]*CameraStream),
		orders:     make(map[string]*BrokerageOrder),
		runs:       make(map[string]*BacktestRun),
		deposits:   make(map[string]*RecurringDeposit),
		portfolios: make(map[string]*Portfolio),
		apiKeys:    make(map[string][]*CustomerAPIKey),
		webhooks:   make(map[string][]*WebhookSubscription),
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


