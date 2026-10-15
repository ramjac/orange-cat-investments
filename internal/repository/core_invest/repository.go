package core_invest

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CameraStream struct {
	StreamID       string    `json:"stream_id"`
	CameraAssetID  string    `json:"camera_asset_id"`
	StreamURL      string    `json:"stream_url"`
	Protocol       string    `json:"protocol"`
	Status         string    `json:"status"`
	IngestSettings string    `json:"ingest_settings"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BrokerageOrder struct {
	OrderID     string     `json:"order_id"`
	PortfolioID string     `json:"portfolio_id"`
	StrategyID  *string    `json:"strategy_id,omitempty"`
	EventID     *string    `json:"event_id,omitempty"`
	BrokerName  string     `json:"broker_name"`
	Symbol      string     `json:"symbol"`
	Side        string     `json:"side"`
	Quantity    float64    `json:"quantity"`
	Price       float64    `json:"price"`
	Status      string     `json:"status"`
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type BacktestRun struct {
	BacktestID string    `json:"backtest_id"`
	StrategyID string    `json:"strategy_id"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Parameters string    `json:"parameters"`
	Results    string    `json:"results"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type RecurringDeposit struct {
	DepositID    string    `json:"deposit_id"`
	CustomerID   string    `json:"customer_id"`
	PortfolioID  string    `json:"portfolio_id"`
	AmountUSD    float64   `json:"amount_usd"`
	Frequency    string    `json:"frequency"`
	BankAccount  string    `json:"bank_account"`
	Status       string    `json:"status"`
	NextChargeAt time.Time `json:"next_charge_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type Portfolio struct {
	PortfolioID     string                   `json:"portfolio_id"`
	CustomerID      string                   `json:"customer_id"`
	CustomerName    string                   `json:"customer_name,omitempty"`
	AccountName     string                   `json:"account_name"`
	TotalBalanceUSD float64                  `json:"total_balance_usd"`
	CashBalanceUSD  float64                  `json:"cash_balance_usd"`
	Status          string                   `json:"status"`
	Holdings        []map[string]interface{} `json:"holdings,omitempty"`
	HistoricalPerf  []map[string]interface{} `json:"historical_performance,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

type CustomerAPIKey struct {
	KeyID       string    `json:"key_id"`
	CustomerID  string    `json:"customer_id"`
	KeyPrefix   string    `json:"key_prefix"`
	KeyHash     string    `json:"key_hash,omitempty"`
	APIKey      string    `json:"api_key,omitempty"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type WebhookSubscription struct {
	SubscriptionID string    `json:"subscription_id"`
	CustomerID     string    `json:"customer_id"`
	TargetURL      string    `json:"target_url"`
	Events         []string  `json:"events"`
	Secret         string    `json:"secret,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type Repository interface {
	GetCameraStreamByID(ctx context.Context, id string) (*CameraStream, error)
	ListCameraStreams(ctx context.Context, limit, offset int32) ([]*CameraStream, error)
	CreateCameraStream(ctx context.Context, stream *CameraStream) (*CameraStream, error)

	GetBrokerageOrderByID(ctx context.Context, id string) (*BrokerageOrder, error)
	ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*BrokerageOrder, error)
	CreateBrokerageOrder(ctx context.Context, order *BrokerageOrder) (*BrokerageOrder, error)
	UpdateBrokerageOrderStatus(ctx context.Context, id string, status string, executedAt *time.Time) (*BrokerageOrder, error)

	GetBacktestRunByID(ctx context.Context, id string) (*BacktestRun, error)
	ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*BacktestRun, error)
	CreateBacktestRun(ctx context.Context, run *BacktestRun) (*BacktestRun, error)
	UpdateBacktestRunResults(ctx context.Context, id string, status string, results string) (*BacktestRun, error)

	CreateRecurringDeposit(ctx context.Context, dep *RecurringDeposit) (*RecurringDeposit, error)
	ListRecurringDeposits(ctx context.Context, customerID string) ([]*RecurringDeposit, error)
	GetRecurringDepositByID(ctx context.Context, id string) (*RecurringDeposit, error)

	// Portfolios
	GetPortfolioByCustomerID(ctx context.Context, customerID string) (*Portfolio, error)
	GetPortfolioByID(ctx context.Context, portfolioID string) (*Portfolio, error)
	CreatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error)

	// API Keys
	CreateAPIKey(ctx context.Context, key *CustomerAPIKey) (*CustomerAPIKey, error)
	ListAPIKeys(ctx context.Context, customerID string) ([]*CustomerAPIKey, error)
	RevokeAPIKey(ctx context.Context, customerID, keyID string) error

	// Webhooks
	CreateWebhook(ctx context.Context, sub *WebhookSubscription) (*WebhookSubscription, error)
	ListWebhooks(ctx context.Context, customerID string) ([]*WebhookSubscription, error)
	DeleteWebhook(ctx context.Context, customerID, subscriptionID string) error
}

type pgxRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepository{db: db}
}

func (r *pgxRepository) GetCameraStreamByID(ctx context.Context, id string) (*CameraStream, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var s CameraStream
	query := `SELECT stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at
              FROM core_invest.camera_streams WHERE stream_id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&s.StreamID, &s.CameraAssetID, &s.StreamURL, &s.Protocol, &s.Status, &s.IngestSettings, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *pgxRepository) ListCameraStreams(ctx context.Context, limit, offset int32) ([]*CameraStream, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at
              FROM core_invest.camera_streams ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []*CameraStream
	for rows.Next() {
		var s CameraStream
		if err := rows.Scan(&s.StreamID, &s.CameraAssetID, &s.StreamURL, &s.Protocol, &s.Status, &s.IngestSettings, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		streams = append(streams, &s)
	}
	return streams, rows.Err()
}

func (r *pgxRepository) CreateCameraStream(ctx context.Context, stream *CameraStream) (*CameraStream, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.camera_streams (
                  camera_asset_id, stream_url, protocol, status, ingest_settings
              ) VALUES ($1, $2, $3, $4, $5)
              RETURNING stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at`
	var s CameraStream
	err := r.db.QueryRow(ctx, query, stream.CameraAssetID, stream.StreamURL, stream.Protocol, stream.Status, stream.IngestSettings).
		Scan(&s.StreamID, &s.CameraAssetID, &s.StreamURL, &s.Protocol, &s.Status, &s.IngestSettings, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *pgxRepository) GetBrokerageOrderByID(ctx context.Context, id string) (*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var o BrokerageOrder
	query := `SELECT order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at
              FROM core_invest.brokerage_orders WHERE order_id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *pgxRepository) ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at
              FROM core_invest.brokerage_orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*BrokerageOrder
	for rows.Next() {
		var o BrokerageOrder
		if err := rows.Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, rows.Err()
}

func (r *pgxRepository) CreateBrokerageOrder(ctx context.Context, order *BrokerageOrder) (*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.brokerage_orders (
                  portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status
              ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
              RETURNING order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at`
	var o BrokerageOrder
	err := r.db.QueryRow(ctx, query, order.PortfolioID, order.StrategyID, order.EventID, order.BrokerName, order.Symbol, order.Side, order.Quantity, order.Price, order.Status).
		Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *pgxRepository) UpdateBrokerageOrderStatus(ctx context.Context, id string, status string, executedAt *time.Time) (*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE core_invest.brokerage_orders
              SET status = $2, executed_at = $3, updated_at = CURRENT_TIMESTAMP
              WHERE order_id = $1
              RETURNING order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at`
	var o BrokerageOrder
	err := r.db.QueryRow(ctx, query, id, status, executedAt).
		Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *pgxRepository) GetBacktestRunByID(ctx context.Context, id string) (*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var b BacktestRun
	query := `SELECT backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at
              FROM core_invest.backtest_runs WHERE backtest_id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *pgxRepository) ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at
              FROM core_invest.backtest_runs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*BacktestRun
	for rows.Next() {
		var b BacktestRun
		if err := rows.Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, &b)
	}
	return runs, rows.Err()
}

func (r *pgxRepository) CreateBacktestRun(ctx context.Context, run *BacktestRun) (*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.backtest_runs (
                  strategy_id, start_date, end_date, parameters, status
              ) VALUES ($1, $2, $3, $4, 'pending')
              RETURNING backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at`
	var b BacktestRun
	err := r.db.QueryRow(ctx, query, run.StrategyID, run.StartDate, run.EndDate, run.Parameters).
		Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *pgxRepository) CreateRecurringDeposit(ctx context.Context, dep *RecurringDeposit) (*RecurringDeposit, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.recurring_deposits (customer_id, portfolio_id, amount_usd, frequency, bank_account, status, next_charge_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)
              RETURNING deposit_id, customer_id, portfolio_id, amount_usd, frequency, bank_account, status, next_charge_at, created_at`
	var created RecurringDeposit
	err := r.db.QueryRow(ctx, query, dep.CustomerID, dep.PortfolioID, dep.AmountUSD, dep.Frequency, dep.BankAccount, dep.Status, dep.NextChargeAt).
		Scan(&created.DepositID, &created.CustomerID, &created.PortfolioID, &created.AmountUSD, &created.Frequency, &created.BankAccount, &created.Status, &created.NextChargeAt, &created.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create recurring deposit: %w", err)
	}
	return &created, nil
}

func (r *pgxRepository) ListRecurringDeposits(ctx context.Context, customerID string) ([]*RecurringDeposit, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var (
		rows pgx.Rows
		err  error
	)
	if customerID == "" {
		query := `SELECT deposit_id, customer_id, portfolio_id, amount_usd, frequency, bank_account, status, next_charge_at, created_at
              FROM core_invest.recurring_deposits ORDER BY created_at DESC`
		rows, err = r.db.Query(ctx, query)
	} else {
		query := `SELECT deposit_id, customer_id, portfolio_id, amount_usd, frequency, bank_account, status, next_charge_at, created_at
              FROM core_invest.recurring_deposits WHERE customer_id = $1 ORDER BY created_at DESC`
		rows, err = r.db.Query(ctx, query, customerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query recurring deposits: %w", err)
	}
	defer rows.Close()

	var list []*RecurringDeposit
	for rows.Next() {
		var d RecurringDeposit
		if err := rows.Scan(&d.DepositID, &d.CustomerID, &d.PortfolioID, &d.AmountUSD, &d.Frequency, &d.BankAccount, &d.Status, &d.NextChargeAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &d)
	}
	return list, nil
}

func (r *pgxRepository) GetRecurringDepositByID(ctx context.Context, id string) (*RecurringDeposit, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT deposit_id, customer_id, portfolio_id, amount_usd, frequency, bank_account, status, next_charge_at, created_at
              FROM core_invest.recurring_deposits WHERE deposit_id = $1`
	var d RecurringDeposit
	err := r.db.QueryRow(ctx, query, id).Scan(&d.DepositID, &d.CustomerID, &d.PortfolioID, &d.AmountUSD, &d.Frequency, &d.BankAccount, &d.Status, &d.NextChargeAt, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring deposit: %w", err)
	}
	return &d, nil
}

func (r *pgxRepository) UpdateBacktestRunResults(ctx context.Context, id string, status string, results string) (*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE core_invest.backtest_runs
              SET status = $2, results = $3, updated_at = CURRENT_TIMESTAMP
              WHERE backtest_id = $1
              RETURNING backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at`
	var b BacktestRun
	err := r.db.QueryRow(ctx, query, id, status, results).
		Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

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
	return &p, nil
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

