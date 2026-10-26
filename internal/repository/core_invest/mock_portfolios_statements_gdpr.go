package core_invest

import (
	"context"
	"fmt"
	"time"
)

func (m *mockRepository) seedTaxLotsAndStatements(now time.Time) {
	arthurTaxLots := []*TaxLotSale{
		{LotID: "018f3a9a-aaaa-7000-8000-000000000001", PortfolioID: "018f3a9a-2222-7000-8000-000000000002", Symbol: "NVDA", Description: "50 shs NVDA", Quantity: 50.0, DateAcquired: "2025-02-14", DateSold: "2025-08-20", GrossProceedsUSD: 6940.00, CostBasisUSD: 5500.00, WashSaleLossDisallowed: 0.00, RealizedGainLossUSD: 1440.00, IsShortTerm: true, TaxYear: 2025, CreatedAt: now, UpdatedAt: now},
		{LotID: "018f3a9a-aaaa-7000-8000-000000000002", PortfolioID: "018f3a9a-2222-7000-8000-000000000002", Symbol: "AAPL", Description: "35 shs AAPL", Quantity: 35.0, DateAcquired: "2025-01-10", DateSold: "2025-09-15", GrossProceedsUSD: 7850.00, CostBasisUSD: 6200.00, WashSaleLossDisallowed: 0.00, RealizedGainLossUSD: 1650.00, IsShortTerm: true, TaxYear: 2025, CreatedAt: now, UpdatedAt: now},
		{LotID: "018f3a9a-aaaa-7000-8000-000000000003", PortfolioID: "018f3a9a-2222-7000-8000-000000000002", Symbol: "MSFT", Description: "20 shs MSFT", Quantity: 20.0, DateAcquired: "2024-03-01", DateSold: "2025-11-12", GrossProceedsUSD: 8960.00, CostBasisUSD: 7100.00, WashSaleLossDisallowed: 0.00, RealizedGainLossUSD: 1860.00, IsShortTerm: false, TaxYear: 2025, CreatedAt: now, UpdatedAt: now},
	}
	m.taxLots["018f3a9a-2222-7000-8000-000000000002"] = arthurTaxLots
	m.taxLots["cust-longterm-arthur"] = arthurTaxLots
	m.taxLots["port-cust-longterm-arthur"] = arthurTaxLots

	chloeTaxLots := []*TaxLotSale{
		{LotID: "018f3a9a-aaaa-7000-8000-000000000004", PortfolioID: "018f3a9a-2222-7000-8000-000000000003", Symbol: "TSLA", Description: "40 shs TSLA", Quantity: 40.0, DateAcquired: "2025-03-15", DateSold: "2025-07-22", GrossProceedsUSD: 9884.80, CostBasisUSD: 7420.00, WashSaleLossDisallowed: 0.00, RealizedGainLossUSD: 2464.80, IsShortTerm: true, TaxYear: 2025, CreatedAt: now, UpdatedAt: now},
		{LotID: "018f3a9a-aaaa-7000-8000-000000000005", PortfolioID: "018f3a9a-2222-7000-8000-000000000003", Symbol: "NVDA", Description: "60 shs NVDA", Quantity: 60.0, DateAcquired: "2025-02-01", DateSold: "2025-08-10", GrossProceedsUSD: 8328.00, CostBasisUSD: 6744.00, WashSaleLossDisallowed: 0.00, RealizedGainLossUSD: 1584.00, IsShortTerm: true, TaxYear: 2025, CreatedAt: now, UpdatedAt: now},
	}
	m.taxLots["018f3a9a-2222-7000-8000-000000000003"] = chloeTaxLots
	m.taxLots["cust-active-chloe"] = chloeTaxLots
	m.taxLots["port-cust-active-chloe"] = chloeTaxLots

	m.webhooks["cust-longterm-arthur"] = []*WebhookSubscription{
		{SubscriptionID: "sub-018f-1111", CustomerID: "cust-longterm-arthur", TargetURL: "https://quant-bot.chloespark.io/api/v1/oci-callback", Events: []string{"events.observation.cat_spotted.v1", "events.trade.executed.v1"}, Status: "active", CreatedAt: now},
	}

	m.apiKeys["cust-active-chloe"] = []*CustomerAPIKey{
		{KeyID: "key-018f-8888", CustomerID: "cust-active-chloe", Name: "Chloe Momentum Bot", KeyPrefix: "oci_live_9a8f", Status: "active", Permissions: []string{"read_streams", "execute_orders", "read_telemetry"}, CreatedAt: now},
	}

	m.statements["cust-longterm-arthur"] = []*InvestmentStatement{
		{StatementID: "stmt-arthur-2026-03", PortfolioID: "port-cust-longterm-arthur", CustomerID: "cust-longterm-arthur", StatementType: "monthly_summary", Period: "March 2026", TaxYear: 2026, Title: "Monthly Portfolio Summary - March 2026", FileName: "monthly-statement-port-cust-longterm-arthur-2026-03.pdf", TotalProceedsUSD: 128450.75, NetGainUSD: 28450.75, Status: "available", CreatedAt: now.Add(-24 * time.Hour * 15)},
		{StatementID: "stmt-arthur-1099b-2025", PortfolioID: "port-cust-longterm-arthur", CustomerID: "cust-longterm-arthur", StatementType: "form_1099b", Period: "Tax Year 2025", TaxYear: 2025, Title: "IRS Form 1099-B Tax Report - 2025", FileName: "form1099b-port-cust-longterm-arthur-2025.pdf", TotalProceedsUSD: 64200.00, NetGainUSD: 14300.00, Status: "available", CreatedAt: now.Add(-24 * time.Hour * 45)},
	}

	m.statements["cust-active-chloe"] = []*InvestmentStatement{
		{StatementID: "stmt-chloe-2026-03", PortfolioID: "port-cust-active-chloe", CustomerID: "cust-active-chloe", StatementType: "monthly_summary", Period: "March 2026", TaxYear: 2026, Title: "Monthly Portfolio Summary - March 2026", FileName: "monthly-statement-port-cust-active-chloe-2026-03.pdf", TotalProceedsUSD: 85200.00, NetGainUSD: 22800.00, Status: "available", CreatedAt: now.Add(-24 * time.Hour * 15)},
		{StatementID: "stmt-chloe-1099b-2025", PortfolioID: "port-cust-active-chloe", CustomerID: "cust-active-chloe", StatementType: "form_1099b", Period: "Tax Year 2025", TaxYear: 2025, Title: "IRS Form 1099-B Tax Report - 2025", FileName: "form1099b-port-cust-active-chloe-2025.pdf", TotalProceedsUSD: 98400.00, NetGainUSD: 32100.00, Status: "available", CreatedAt: now.Add(-24 * time.Hour * 45)},
	}
}

func (m *mockRepository) GetPortfolioByCustomerID(ctx context.Context, customerID string) (*Portfolio, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.portfolios[customerID]; ok {
		return p, nil
	}

	for _, p := range m.portfolios {
		if p.CustomerID == customerID || p.PortfolioID == customerID {
			return p, nil
		}
	}

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

func (m *mockRepository) ListPortfolios(ctx context.Context) ([]*Portfolio, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Portfolio
	for _, p := range m.portfolios {
		list = append(list, p)
	}
	return list, nil
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
