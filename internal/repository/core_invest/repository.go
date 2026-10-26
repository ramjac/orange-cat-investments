package core_invest

import (
	"context"
	"time"

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
	StatementType    string                 `json:"statement_type"`
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

type StreamBacktestRepository interface {
	GetCameraStreamByID(ctx context.Context, id string) (*CameraStream, error)
	ListCameraStreams(ctx context.Context, limit, offset int32) ([]*CameraStream, error)
	CreateCameraStream(ctx context.Context, stream *CameraStream) (*CameraStream, error)
	GetBacktestRunByID(ctx context.Context, id string) (*BacktestRun, error)
	ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*BacktestRun, error)
	CreateBacktestRun(ctx context.Context, run *BacktestRun) (*BacktestRun, error)
	UpdateBacktestRunResults(ctx context.Context, id string, status string, results string) (*BacktestRun, error)
}

type OrderDepositRepository interface {
	GetBrokerageOrderByID(ctx context.Context, id string) (*BrokerageOrder, error)
	ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*BrokerageOrder, error)
	CreateBrokerageOrder(ctx context.Context, order *BrokerageOrder) (*BrokerageOrder, error)
	UpdateBrokerageOrderStatus(ctx context.Context, id string, status string, executedAt *time.Time) (*BrokerageOrder, error)
	CreateRecurringDeposit(ctx context.Context, dep *RecurringDeposit) (*RecurringDeposit, error)
	ListRecurringDeposits(ctx context.Context, customerID string) ([]*RecurringDeposit, error)
	GetRecurringDepositByID(ctx context.Context, id string) (*RecurringDeposit, error)
	AdvanceRecurringDepositSchedule(ctx context.Context, id string, nextChargeAt time.Time) error
}

type PortfolioDeveloperRepository interface {
	GetPortfolioByCustomerID(ctx context.Context, customerID string) (*Portfolio, error)
	GetPortfolioByID(ctx context.Context, portfolioID string) (*Portfolio, error)
	ListPortfolios(ctx context.Context) ([]*Portfolio, error)
	CreatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error)
	UpdatePortfolio(ctx context.Context, p *Portfolio) (*Portfolio, error)
	GetHoldingsByPortfolioID(ctx context.Context, portfolioID string) ([]*PortfolioHolding, error)
	UpsertHolding(ctx context.Context, holding *PortfolioHolding) (*PortfolioHolding, error)
	CreateAPIKey(ctx context.Context, key *CustomerAPIKey) (*CustomerAPIKey, error)
	ListAPIKeys(ctx context.Context, customerID string) ([]*CustomerAPIKey, error)
	RevokeAPIKey(ctx context.Context, customerID, keyID string) error
	CreateWebhook(ctx context.Context, sub *WebhookSubscription) (*WebhookSubscription, error)
	ListWebhooks(ctx context.Context, customerID string) ([]*WebhookSubscription, error)
	DeleteWebhook(ctx context.Context, customerID, subscriptionID string) error
}

type StatementGDPRRepository interface {
	CreateInvestmentStatement(ctx context.Context, stmt *InvestmentStatement) (*InvestmentStatement, error)
	ListInvestmentStatements(ctx context.Context, customerID string) ([]*InvestmentStatement, error)
	GetInvestmentStatementByID(ctx context.Context, id string) (*InvestmentStatement, error)
	GetTaxLotSalesByPortfolioID(ctx context.Context, portfolioID string, taxYear int) ([]*TaxLotSale, error)
	CreateTaxLotSale(ctx context.Context, lot *TaxLotSale) (*TaxLotSale, error)
	CreateGDPRErasureCertificate(ctx context.Context, cert *GDPRCertificateRecord) (*GDPRCertificateRecord, error)
	GetGDPRErasureCertificateByID(ctx context.Context, id string) (*GDPRCertificateRecord, error)
	ExecuteGDPRErasure(ctx context.Context, customerID string) (*GDPRErasureStats, error)
}

type Repository interface {
	StreamBacktestRepository
	OrderDepositRepository
	PortfolioDeveloperRepository
	StatementGDPRRepository
}

type pgxRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepository{db: db}
}
