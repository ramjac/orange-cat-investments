package core_invest

import (
	"context"
	"encoding/json"
	"fmt"
)

func (r *pgxRepository) GetPortfolioByCustomerID(ctx context.Context, customerID string) (*Portfolio, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT portfolio_id, customer_id, account_name, total_balance_usd, cash_balance_usd, status, created_at, updated_at
              FROM core_invest.investment_portfolios
              WHERE customer_id::text = $1 OR portfolio_id::text = $1
              ORDER BY created_at ASC LIMIT 1`
	var p Portfolio
	err := r.db.QueryRow(ctx, query, customerID).Scan(&p.PortfolioID, &p.CustomerID, &p.AccountName, &p.TotalBalanceUSD, &p.CashBalanceUSD, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	holdings, _ := r.GetHoldingsByPortfolioID(ctx, p.PortfolioID)
	if len(holdings) > 0 {
		p.Holdings = make([]map[string]interface{}, 0, len(holdings))
		for _, h := range holdings {
			p.Holdings = append(p.Holdings, map[string]interface{}{
				"holding_id":          h.HoldingID,
				"symbol":              h.Symbol,
				"quantity":            h.Quantity,
				"avg_price":           h.AvgPriceUSD,
				"current_price":       h.CurrentPriceUSD,
				"cost_basis":          h.CostBasisUSD,
				"market_value":        h.MarketValueUSD,
				"unrealized_gain":     h.UnrealizedGainUSD,
				"unrealized_gain_pct": h.UnrealizedGainPct,
			})
		}
	}
	return &p, nil
}

func (r *pgxRepository) GetPortfolioByID(ctx context.Context, portfolioID string) (*Portfolio, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT portfolio_id, customer_id, account_name, total_balance_usd, cash_balance_usd, status, created_at, updated_at
              FROM core_invest.investment_portfolios
              WHERE portfolio_id::text = $1
              LIMIT 1`
	var p Portfolio
	err := r.db.QueryRow(ctx, query, portfolioID).Scan(&p.PortfolioID, &p.CustomerID, &p.AccountName, &p.TotalBalanceUSD, &p.CashBalanceUSD, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	holdings, _ := r.GetHoldingsByPortfolioID(ctx, p.PortfolioID)
	if len(holdings) > 0 {
		p.Holdings = make([]map[string]interface{}, 0, len(holdings))
		for _, h := range holdings {
			p.Holdings = append(p.Holdings, map[string]interface{}{
				"holding_id":          h.HoldingID,
				"symbol":              h.Symbol,
				"quantity":            h.Quantity,
				"avg_price":           h.AvgPriceUSD,
				"current_price":       h.CurrentPriceUSD,
				"cost_basis":          h.CostBasisUSD,
				"market_value":        h.MarketValueUSD,
				"unrealized_gain":     h.UnrealizedGainUSD,
				"unrealized_gain_pct": h.UnrealizedGainPct,
			})
		}
	}
	return &p, nil
}

func (r *pgxRepository) ListPortfolios(ctx context.Context) ([]*Portfolio, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT portfolio_id, customer_id, account_name, total_balance_usd, cash_balance_usd, status, created_at, updated_at
              FROM core_invest.investment_portfolios
              ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Portfolio
	for rows.Next() {
		var p Portfolio
		if err := rows.Scan(&p.PortfolioID, &p.CustomerID, &p.AccountName, &p.TotalBalanceUSD, &p.CashBalanceUSD, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, nil
}

func (r *pgxRepository) GetHoldingsByPortfolioID(ctx context.Context, portfolioID string) ([]*PortfolioHolding, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT holding_id, portfolio_id, symbol, quantity, cost_basis_usd, avg_price_usd, current_price_usd, market_value_usd, unrealized_gain_usd, unrealized_gain_pct, created_at, updated_at
              FROM core_invest.portfolio_holdings
              WHERE portfolio_id::text = $1 OR portfolio_id IN (
                  SELECT portfolio_id FROM core_invest.investment_portfolios WHERE customer_id::text = $1
              )
              ORDER BY market_value_usd DESC`
	rows, err := r.db.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio holdings: %w", err)
	}
	defer rows.Close()

	var list []*PortfolioHolding
	for rows.Next() {
		var h PortfolioHolding
		if err := rows.Scan(
			&h.HoldingID, &h.PortfolioID, &h.Symbol, &h.Quantity,
			&h.CostBasisUSD, &h.AvgPriceUSD, &h.CurrentPriceUSD,
			&h.MarketValueUSD, &h.UnrealizedGainUSD, &h.UnrealizedGainPct,
			&h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &h)
	}
	return list, rows.Err()
}

func (r *pgxRepository) UpsertHolding(ctx context.Context, holding *PortfolioHolding) (*PortfolioHolding, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.portfolio_holdings (
                  portfolio_id, symbol, quantity, cost_basis_usd, avg_price_usd, current_price_usd, market_value_usd, unrealized_gain_usd, unrealized_gain_pct, updated_at
              ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
              ON CONFLICT (portfolio_id, symbol) DO UPDATE SET
                  quantity = EXCLUDED.quantity,
                  cost_basis_usd = EXCLUDED.cost_basis_usd,
                  avg_price_usd = EXCLUDED.avg_price_usd,
                  current_price_usd = EXCLUDED.current_price_usd,
                  market_value_usd = EXCLUDED.market_value_usd,
                  unrealized_gain_usd = EXCLUDED.unrealized_gain_usd,
                  unrealized_gain_pct = EXCLUDED.unrealized_gain_pct,
                  updated_at = CURRENT_TIMESTAMP
              RETURNING holding_id, portfolio_id, symbol, quantity, cost_basis_usd, avg_price_usd, current_price_usd, market_value_usd, unrealized_gain_usd, unrealized_gain_pct, created_at, updated_at`
	var h PortfolioHolding
	err := r.db.QueryRow(ctx, query,
		holding.PortfolioID, holding.Symbol, holding.Quantity, holding.CostBasisUSD,
		holding.AvgPriceUSD, holding.CurrentPriceUSD, holding.MarketValueUSD,
		holding.UnrealizedGainUSD, holding.UnrealizedGainPct,
	).Scan(
		&h.HoldingID, &h.PortfolioID, &h.Symbol, &h.Quantity,
		&h.CostBasisUSD, &h.AvgPriceUSD, &h.CurrentPriceUSD,
		&h.MarketValueUSD, &h.UnrealizedGainUSD, &h.UnrealizedGainPct,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert portfolio holding: %w", err)
	}
	return &h, nil
}

func (r *pgxRepository) CreatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.investment_portfolios (customer_id, account_name, total_balance_usd, cash_balance_usd, status)
              VALUES ($1, $2, $3, $4, $5)
              RETURNING portfolio_id, customer_id, account_name, total_balance_usd, cash_balance_usd, status, created_at, updated_at`
	var created Portfolio
	status := p.Status
	if status == "" {
		status = "active"
	}
	err := r.db.QueryRow(ctx, query, p.CustomerID, p.AccountName, p.TotalBalanceUSD, p.CashBalanceUSD, status).
		Scan(&created.PortfolioID, &created.CustomerID, &created.AccountName, &created.TotalBalanceUSD, &created.CashBalanceUSD, &created.Status, &created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}
	return &created, nil
}

func (r *pgxRepository) CreateAPIKey(ctx context.Context, key *CustomerAPIKey) (*CustomerAPIKey, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.customer_api_keys (customer_id, key_prefix, key_hash, name, status)
              VALUES ($1, $2, $3, $4, $5)
              RETURNING key_id, customer_id, key_prefix, name, status, created_at`
	var created CustomerAPIKey
	status := key.Status
	if status == "" {
		status = "active"
	}
	err := r.db.QueryRow(ctx, query, key.CustomerID, key.KeyPrefix, key.KeyHash, key.Name, status).
		Scan(&created.KeyID, &created.CustomerID, &created.KeyPrefix, &created.Name, &created.Status, &created.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}
	created.Permissions = key.Permissions
	return &created, nil
}

func (r *pgxRepository) ListAPIKeys(ctx context.Context, customerID string) ([]*CustomerAPIKey, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT key_id, customer_id, key_prefix, name, status, created_at
              FROM core_invest.customer_api_keys
              WHERE customer_id = $1 AND status != 'revoked'
              ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query api keys: %w", err)
	}
	defer rows.Close()

	var list []*CustomerAPIKey
	for rows.Next() {
		var k CustomerAPIKey
		if err := rows.Scan(&k.KeyID, &k.CustomerID, &k.KeyPrefix, &k.Name, &k.Status, &k.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &k)
	}
	return list, nil
}

func (r *pgxRepository) RevokeAPIKey(ctx context.Context, customerID, keyID string) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	query := `UPDATE core_invest.customer_api_keys
              SET status = 'revoked'
              WHERE customer_id = $1 AND (key_id::text = $2 OR key_prefix = $2)`
	_, err := r.db.Exec(ctx, query, customerID, keyID)
	return err
}

func (r *pgxRepository) CreateWebhook(ctx context.Context, sub *WebhookSubscription) (*WebhookSubscription, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.webhook_subscriptions (customer_id, target_url, events, secret, status)
              VALUES ($1, $2, $3, $4, $5)
              RETURNING subscription_id, customer_id, target_url, events, status, created_at`
	var created WebhookSubscription
	status := sub.Status
	if status == "" {
		status = "active"
	}
	err := r.db.QueryRow(ctx, query, sub.CustomerID, sub.TargetURL, sub.Events, sub.Secret, status).
		Scan(&created.SubscriptionID, &created.CustomerID, &created.TargetURL, &created.Events, &created.Status, &created.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}
	return &created, nil
}

func (r *pgxRepository) ListWebhooks(ctx context.Context, customerID string) ([]*WebhookSubscription, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT subscription_id, customer_id, target_url, events, status, created_at
              FROM core_invest.webhook_subscriptions
              WHERE customer_id = $1 AND status != 'deleted'
              ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", err)
	}
	defer rows.Close()

	var list []*WebhookSubscription
	for rows.Next() {
		var w WebhookSubscription
		if err := rows.Scan(&w.SubscriptionID, &w.CustomerID, &w.TargetURL, &w.Events, &w.Status, &w.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &w)
	}
	return list, nil
}

func (r *pgxRepository) DeleteWebhook(ctx context.Context, customerID, subscriptionID string) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	query := `UPDATE core_invest.webhook_subscriptions
              SET status = 'deleted'
              WHERE customer_id = $1 AND subscription_id::text = $2`
	_, err := r.db.Exec(ctx, query, customerID, subscriptionID)
	return err
}

func (r *pgxRepository) UpdatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE core_invest.investment_portfolios
              SET account_name = $2, total_balance_usd = $3, cash_balance_usd = $4, status = $5, updated_at = CURRENT_TIMESTAMP
              WHERE portfolio_id::text = $1 OR customer_id::text = $1
              RETURNING portfolio_id, customer_id, account_name, total_balance_usd, cash_balance_usd, status, created_at, updated_at`
	var updated Portfolio
	err := r.db.QueryRow(ctx, query, p.PortfolioID, p.AccountName, p.TotalBalanceUSD, p.CashBalanceUSD, p.Status).
		Scan(&updated.PortfolioID, &updated.CustomerID, &updated.AccountName, &updated.TotalBalanceUSD, &updated.CashBalanceUSD, &updated.Status, &updated.CreatedAt, &updated.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update portfolio: %w", err)
	}
	if p.Status == "closed" {
		_, _ = r.db.Exec(ctx, `DELETE FROM core_invest.portfolio_holdings WHERE portfolio_id::text = $1`, p.PortfolioID)
	}
	updated.Holdings = p.Holdings
	updated.HistoricalPerf = p.HistoricalPerf
	return &updated, nil
}

func (r *pgxRepository) CreateInvestmentStatement(ctx context.Context, stmt *InvestmentStatement) (*InvestmentStatement, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.investment_statements (
                  portfolio_id, customer_id, statement_type, period, tax_year, title, file_name, total_proceeds_usd, net_gain_usd, status, metadata
              ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
              RETURNING statement_id, portfolio_id, customer_id, statement_type, period, tax_year, title, file_name, total_proceeds_usd, net_gain_usd, status, created_at`
	var s InvestmentStatement
	metaJSON, _ := json.Marshal(stmt.Metadata)
	err := r.db.QueryRow(ctx, query, stmt.PortfolioID, stmt.CustomerID, stmt.StatementType, stmt.Period, stmt.TaxYear, stmt.Title, stmt.FileName, stmt.TotalProceedsUSD, stmt.NetGainUSD, stmt.Status, metaJSON).
		Scan(&s.StatementID, &s.PortfolioID, &s.CustomerID, &s.StatementType, &s.Period, &s.TaxYear, &s.Title, &s.FileName, &s.TotalProceedsUSD, &s.NetGainUSD, &s.Status, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create investment statement: %w", err)
	}
	s.Metadata = stmt.Metadata
	return &s, nil
}

func (r *pgxRepository) ListInvestmentStatements(ctx context.Context, customerID string) ([]*InvestmentStatement, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT statement_id, portfolio_id, customer_id, statement_type, period, tax_year, title, file_name, total_proceeds_usd, net_gain_usd, status, created_at
              FROM core_invest.investment_statements
              WHERE customer_id = $1
              ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list investment statements: %w", err)
	}
	defer rows.Close()

	var list []*InvestmentStatement
	for rows.Next() {
		var s InvestmentStatement
		if err := rows.Scan(&s.StatementID, &s.PortfolioID, &s.CustomerID, &s.StatementType, &s.Period, &s.TaxYear, &s.Title, &s.FileName, &s.TotalProceedsUSD, &s.NetGainUSD, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}

func (r *pgxRepository) GetInvestmentStatementByID(ctx context.Context, id string) (*InvestmentStatement, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT statement_id, portfolio_id, customer_id, statement_type, period, tax_year, title, file_name, total_proceeds_usd, net_gain_usd, status, created_at
              FROM core_invest.investment_statements
              WHERE statement_id::text = $1`
	var s InvestmentStatement
	err := r.db.QueryRow(ctx, query, id).
		Scan(&s.StatementID, &s.PortfolioID, &s.CustomerID, &s.StatementType, &s.Period, &s.TaxYear, &s.Title, &s.FileName, &s.TotalProceedsUSD, &s.NetGainUSD, &s.Status, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get investment statement: %w", err)
	}
	return &s, nil
}

func (r *pgxRepository) GetTaxLotSalesByPortfolioID(ctx context.Context, portfolioID string, taxYear int) ([]*TaxLotSale, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT lot_id, portfolio_id, symbol, description, quantity, date_acquired::text, date_sold::text, gross_proceeds_usd, cost_basis_usd, wash_sale_loss_disallowed, realized_gain_loss_usd, is_short_term, tax_year, created_at, updated_at
              FROM core_invest.tax_lot_sales
              WHERE (portfolio_id::text = $1 OR portfolio_id IN (
                  SELECT portfolio_id FROM core_invest.investment_portfolios WHERE customer_id::text = $1
              )) AND ($2 = 0 OR tax_year = $2)
              ORDER BY date_sold ASC`
	rows, err := r.db.Query(ctx, query, portfolioID, taxYear)
	if err != nil {
		return nil, fmt.Errorf("failed to query tax lot sales: %w", err)
	}
	defer rows.Close()

	var list []*TaxLotSale
	for rows.Next() {
		var l TaxLotSale
		if err := rows.Scan(
			&l.LotID, &l.PortfolioID, &l.Symbol, &l.Description, &l.Quantity,
			&l.DateAcquired, &l.DateSold, &l.GrossProceedsUSD, &l.CostBasisUSD,
			&l.WashSaleLossDisallowed, &l.RealizedGainLossUSD, &l.IsShortTerm,
			&l.TaxYear, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &l)
	}
	return list, rows.Err()
}

func (r *pgxRepository) CreateTaxLotSale(ctx context.Context, lot *TaxLotSale) (*TaxLotSale, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.tax_lot_sales (
                  portfolio_id, symbol, description, quantity, date_acquired, date_sold, gross_proceeds_usd, cost_basis_usd, wash_sale_loss_disallowed, realized_gain_loss_usd, is_short_term, tax_year
              ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
              RETURNING lot_id, portfolio_id, symbol, description, quantity, date_acquired::text, date_sold::text, gross_proceeds_usd, cost_basis_usd, wash_sale_loss_disallowed, realized_gain_loss_usd, is_short_term, tax_year, created_at, updated_at`
	var l TaxLotSale
	err := r.db.QueryRow(ctx, query,
		lot.PortfolioID, lot.Symbol, lot.Description, lot.Quantity,
		lot.DateAcquired, lot.DateSold, lot.GrossProceedsUSD, lot.CostBasisUSD,
		lot.WashSaleLossDisallowed, lot.RealizedGainLossUSD, lot.IsShortTerm, lot.TaxYear,
	).Scan(
		&l.LotID, &l.PortfolioID, &l.Symbol, &l.Description, &l.Quantity,
		&l.DateAcquired, &l.DateSold, &l.GrossProceedsUSD, &l.CostBasisUSD,
		&l.WashSaleLossDisallowed, &l.RealizedGainLossUSD, &l.IsShortTerm,
		&l.TaxYear, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tax lot sale: %w", err)
	}
	return &l, nil
}

