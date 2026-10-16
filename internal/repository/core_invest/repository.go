package core_invest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

type PortfolioHolding struct {
	HoldingID         string    `json:"holding_id"`
	PortfolioID       string    `json:"portfolio_id"`
	Symbol            string    `json:"symbol"`
	Quantity          float64   `json:"quantity"`
	CostBasisUSD      float64   `json:"cost_basis_usd"`
	AvgPriceUSD       float64   `json:"avg_price_usd"`
	CurrentPriceUSD   float64   `json:"current_price_usd"`
	MarketValueUSD    float64   `json:"market_value_usd"`
	UnrealizedGainUSD float64   `json:"unrealized_gain_usd"`
	UnrealizedGainPct float64   `json:"unrealized_gain_pct"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type TaxLotSale struct {
	LotID                  string    `json:"lot_id"`
	PortfolioID            string    `json:"portfolio_id"`
	Symbol                 string    `json:"symbol"`
	Description            string    `json:"description"`
	Quantity               float64   `json:"quantity"`
	DateAcquired           string    `json:"date_acquired"`
	DateSold               string    `json:"date_sold"`
	GrossProceedsUSD       float64   `json:"gross_proceeds_usd"`
	CostBasisUSD           float64   `json:"cost_basis_usd"`
	WashSaleLossDisallowed float64   `json:"wash_sale_loss_disallowed"`
	RealizedGainLossUSD    float64   `json:"realized_gain_loss_usd"`
	IsShortTerm            bool      `json:"is_short_term"`
	TaxYear                int       `json:"tax_year"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
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

type InvestmentStatement struct {
	StatementID      string                 `json:"statement_id"`
	PortfolioID      string                 `json:"portfolio_id"`
	CustomerID       string                 `json:"customer_id"`
	StatementType    string                 `json:"statement_type"` // "monthly_summary", "form_1099b"
	Period           string                 `json:"period"`
	TaxYear          int                    `json:"tax_year"`
	Title            string                 `json:"title"`
	FileName         string                 `json:"file_name"`
	TotalProceedsUSD float64                `json:"total_proceeds_usd"`
	NetGainUSD       float64                `json:"net_gain_usd"`
	Status           string                 `json:"status"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

type GDPRCertificateRecord struct {
	CertificateID        string    `json:"certificate_id"`
	CustomerIDHash       string    `json:"customer_id_hash"`
	CustomerEmailHash    string    `json:"customer_email_hash"`
	LegalBasis           string    `json:"legal_basis"`
	SupervisoryAuthority string    `json:"supervisory_authority"`
	TablesScrubbed       []string  `json:"tables_scrubbed"`
	SessionsPurgedCount  int       `json:"sessions_purged_count"`
	RecordsScrubbedCount int       `json:"records_scrubbed_count"`
	DigitalSignature     string    `json:"digital_signature"`
	ComplianceStatement  string    `json:"compliance_statement"`
	CreatedAt            time.Time `json:"created_at"`
}

type GDPRErasureStats struct {
	CustomerIDHash       string   `json:"customer_id_hash"`
	CustomerEmailHash    string   `json:"customer_email_hash"`
	TablesScrubbed       []string `json:"tables_scrubbed"`
	RecordsScrubbedCount int      `json:"records_scrubbed_count"`
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
	UpdatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error)
	GetHoldingsByPortfolioID(ctx context.Context, portfolioID string) ([]*PortfolioHolding, error)
	UpsertHolding(ctx context.Context, holding *PortfolioHolding) (*PortfolioHolding, error)

	// API Keys
	CreateAPIKey(ctx context.Context, key *CustomerAPIKey) (*CustomerAPIKey, error)
	ListAPIKeys(ctx context.Context, customerID string) ([]*CustomerAPIKey, error)
	RevokeAPIKey(ctx context.Context, customerID, keyID string) error

	// Webhooks
	CreateWebhook(ctx context.Context, sub *WebhookSubscription) (*WebhookSubscription, error)
	ListWebhooks(ctx context.Context, customerID string) ([]*WebhookSubscription, error)
	DeleteWebhook(ctx context.Context, customerID, subscriptionID string) error

	// Statements & Tax Documents
	CreateInvestmentStatement(ctx context.Context, stmt *InvestmentStatement) (*InvestmentStatement, error)
	ListInvestmentStatements(ctx context.Context, customerID string) ([]*InvestmentStatement, error)
	GetInvestmentStatementByID(ctx context.Context, id string) (*InvestmentStatement, error)
	GetTaxLotSalesByPortfolioID(ctx context.Context, portfolioID string, taxYear int) ([]*TaxLotSale, error)
	CreateTaxLotSale(ctx context.Context, lot *TaxLotSale) (*TaxLotSale, error)

	// GDPR Right to be Forgotten Erasure
	CreateGDPRErasureCertificate(ctx context.Context, cert *GDPRCertificateRecord) (*GDPRCertificateRecord, error)
	GetGDPRErasureCertificateByID(ctx context.Context, id string) (*GDPRCertificateRecord, error)
	ExecuteGDPRErasure(ctx context.Context, customerID string) (*GDPRErasureStats, error)
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

func (r *pgxRepository) CreateGDPRErasureCertificate(ctx context.Context, cert *GDPRCertificateRecord) (*GDPRCertificateRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.gdpr_erasure_certificates (
                  customer_id_hash, customer_email_hash, legal_basis, supervisory_authority, tables_scrubbed, sessions_purged_count, records_scrubbed_count, digital_signature, compliance_statement
              ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
              RETURNING certificate_id, customer_id_hash, customer_email_hash, legal_basis, supervisory_authority, tables_scrubbed, sessions_purged_count, records_scrubbed_count, digital_signature, compliance_statement, created_at`
	var c GDPRCertificateRecord
	err := r.db.QueryRow(ctx, query, cert.CustomerIDHash, cert.CustomerEmailHash, cert.LegalBasis, cert.SupervisoryAuthority, cert.TablesScrubbed, cert.SessionsPurgedCount, cert.RecordsScrubbedCount, cert.DigitalSignature, cert.ComplianceStatement).
		Scan(&c.CertificateID, &c.CustomerIDHash, &c.CustomerEmailHash, &c.LegalBasis, &c.SupervisoryAuthority, &c.TablesScrubbed, &c.SessionsPurgedCount, &c.RecordsScrubbedCount, &c.DigitalSignature, &c.ComplianceStatement, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create gdpr erasure certificate: %w", err)
	}
	return &c, nil
}

func (r *pgxRepository) GetGDPRErasureCertificateByID(ctx context.Context, id string) (*GDPRCertificateRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT certificate_id, customer_id_hash, customer_email_hash, legal_basis, supervisory_authority, tables_scrubbed, sessions_purged_count, records_scrubbed_count, digital_signature, compliance_statement, created_at
              FROM core_invest.gdpr_erasure_certificates
              WHERE certificate_id::text = $1`
	var c GDPRCertificateRecord
	err := r.db.QueryRow(ctx, query, id).
		Scan(&c.CertificateID, &c.CustomerIDHash, &c.CustomerEmailHash, &c.LegalBasis, &c.SupervisoryAuthority, &c.TablesScrubbed, &c.SessionsPurgedCount, &c.RecordsScrubbedCount, &c.DigitalSignature, &c.ComplianceStatement, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get gdpr certificate: %w", err)
	}
	return &c, nil
}

func (r *pgxRepository) ExecuteGDPRErasure(ctx context.Context, customerID string) (*GDPRErasureStats, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	h := sha256.Sum256([]byte(customerID))
	custHash := hex.EncodeToString(h[:])

	scrubbedTables := []string{}
	totalPurged := 0

	// 1. Purge customer_api_keys
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM core_invest.customer_api_keys WHERE customer_id = $1`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.customer_api_keys")
	}

	// 2. Purge webhook_subscriptions
	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.webhook_subscriptions WHERE customer_id = $1`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.webhook_subscriptions")
	}

	// 3. Purge recurring_deposits
	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.recurring_deposits WHERE customer_id = $1`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.recurring_deposits")
	}

	// 4. Purge portfolio holdings, tax lot sales, and anonymize investment_portfolios
	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.portfolio_holdings WHERE portfolio_id IN (
		SELECT portfolio_id FROM core_invest.investment_portfolios WHERE customer_id::text = $1
	)`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.portfolio_holdings")
	}

	cmdTag, err = r.db.Exec(ctx, `DELETE FROM core_invest.tax_lot_sales WHERE portfolio_id IN (
		SELECT portfolio_id FROM core_invest.investment_portfolios WHERE customer_id::text = $1
	)`, customerID)
	if err == nil {
		totalPurged += int(cmdTag.RowsAffected())
		scrubbedTables = append(scrubbedTables, "core_invest.tax_lot_sales")
	}

	anonName := fmt.Sprintf("Anonymized Investor (%s)", custHash[:8])
	_, err = r.db.Exec(ctx, `UPDATE core_invest.investment_portfolios
                             SET account_name = $1, status = 'closed', updated_at = CURRENT_TIMESTAMP
                             WHERE customer_id::text = $2`, anonName, customerID)
	if err == nil {
		scrubbedTables = append(scrubbedTables, "core_invest.investment_portfolios")
	}

	// 5. Anonymize ops.it_tickets and comments
	_, _ = r.db.Exec(ctx, `UPDATE ops.it_tickets
                           SET author_username = 'ANONYMIZED_GDPR_USER',
                               body = '[Content redacted pursuant to GDPR Article 17 erasure request]'
                           WHERE author_username = $1`, customerID)
	scrubbedTables = append(scrubbedTables, "ops.it_tickets")

	_, _ = r.db.Exec(ctx, `UPDATE ops.ticket_comments
                           SET author_username = 'ANONYMIZED_GDPR_USER',
                               comment = '[Content redacted pursuant to GDPR Article 17 erasure request]'
                           WHERE author_username = $1`, customerID)
	scrubbedTables = append(scrubbedTables, "ops.ticket_comments")

	return &GDPRErasureStats{
		CustomerIDHash:       custHash,
		CustomerEmailHash:    custHash,
		TablesScrubbed:       scrubbedTables,
		RecordsScrubbedCount: totalPurged,
	}, nil
}


