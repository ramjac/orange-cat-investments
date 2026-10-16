package reporting

import "time"

// StatementHolding represents an equity holding in a monthly customer portfolio statement.
type StatementHolding struct {
	Symbol             string  `json:"symbol"`
	Quantity           float64 `json:"quantity"`
	CostBasisUSD       float64 `json:"cost_basis_usd"`
	CurrentPriceUSD    float64 `json:"current_price_usd"`
	MarketValueUSD     float64 `json:"market_value_usd"`
	UnrealizedGainUSD  float64 `json:"unrealized_gain_usd"`
	UnrealizedGainPct  float64 `json:"unrealized_gain_pct"`
}

// StatementTrade represents a recent trade logged in the statement period.
type StatementTrade struct {
	OrderID         string  `json:"order_id"`
	Symbol          string  `json:"symbol"`
	Side            string  `json:"side"`
	Quantity        float64 `json:"quantity"`
	PriceUSD        float64 `json:"price_usd"`
	TriggerActivity string  `json:"trigger_activity"`
	ExecutedAt      string  `json:"executed_at"`
}

// CustomerMonthlyStatement contains aggregated monthly performance, balances, and holdings.
type CustomerMonthlyStatement struct {
	StatementID               string             `json:"statement_id"`
	PortfolioID               string             `json:"portfolio_id"`
	CustomerID                string             `json:"customer_id"`
	CustomerName              string             `json:"customer_name"`
	AccountName               string             `json:"account_name"`
	Period                    string             `json:"period"` // e.g., "March 2026"
	TaxYear                   int                `json:"tax_year"`
	OpeningBalanceUSD         float64            `json:"opening_balance_usd"`
	EndingBalanceUSD          float64            `json:"ending_balance_usd"`
	CashBalanceUSD            float64            `json:"cash_balance_usd"`
	NetContributionsUSD       float64            `json:"net_contributions_usd"`
	TotalRealizedGainUSD      float64            `json:"total_realized_gain_usd"`
	TotalUnrealizedGainUSD    float64            `json:"total_unrealized_gain_usd"`
	GrossReturnPct            float64            `json:"gross_return_pct"`
	SharpeRatio               float64            `json:"sharpe_ratio"`
	SortinoRatio              float64            `json:"sortino_ratio"`
	ZoomieAlphaAttributionUSD float64            `json:"zoomie_alpha_attribution_usd"`
	Holdings                  []StatementHolding `json:"holdings"`
	RecentTrades              []StatementTrade   `json:"recent_trades"`
	CreatedAt                 time.Time          `json:"created_at"`
}

// Form1099BTransaction represents an itemized taxable broker barter/exchange transaction.
type Form1099BTransaction struct {
	Description             string  `json:"description"` // e.g. "120 shs AAPL"
	Symbol                  string  `json:"symbol"`
	Quantity                float64 `json:"quantity"`
	DateAcquired            string  `json:"date_acquired"` // YYYY-MM-DD
	DateSold                string  `json:"date_sold"`     // YYYY-MM-DD
	GrossProceedsUSD        float64 `json:"gross_proceeds_usd"`
	CostBasisUSD            float64 `json:"cost_basis_usd"`
	WashSaleLossDisallowed  float64 `json:"wash_sale_loss_disallowed"`
	RealizedGainLossUSD     float64 `json:"realized_gain_loss_usd"`
	IsShortTerm             bool    `json:"is_short_term"`
}

// Form1099BStatement represents IRS Form 1099-B (Proceeds from Broker and Barter Exchange Transactions).
type Form1099BStatement struct {
	StatementID             string                  `json:"statement_id"`
	PortfolioID             string                  `json:"portfolio_id"`
	CustomerID              string                  `json:"customer_id"`
	CustomerName            string                  `json:"customer_name"`
	TaxYear                 int                     `json:"tax_year"`
	PayerName               string                  `json:"payer_name"`
	PayerTIN                string                  `json:"payer_tin"`
	RecipientTIN            string                  `json:"recipient_tin"`
	RecipientAddress        string                  `json:"recipient_address"`
	TotalGrossProceedsUSD   float64                 `json:"total_gross_proceeds_usd"`
	TotalCostBasisUSD       float64                 `json:"total_cost_basis_usd"`
	TotalRealizedGainUSD    float64                 `json:"total_realized_gain_usd"`
	ShortTermGainUSD        float64                 `json:"short_term_gain_usd"`
	LongTermGainUSD         float64                 `json:"long_term_gain_usd"`
	Transactions            []Form1099BTransaction  `json:"transactions"`
	CreatedAt               time.Time               `json:"created_at"`
}

// GDPRDeletionCertificate represents an official cryptographic compliance certificate under GDPR Article 17.
type GDPRDeletionCertificate struct {
	CertificateID          string    `json:"certificate_id"`
	CustomerIDHash         string    `json:"customer_id_hash"`
	CustomerEmailHash      string    `json:"customer_email_hash"`
	ErasureTimestamp       time.Time `json:"erasure_timestamp"`
	LegalBasis             string    `json:"legal_basis"`
	SupervisoryAuthority   string    `json:"supervisory_authority"`
	TablesScrubbed         []string  `json:"tables_scrubbed"`
	SessionsPurgedCount    int       `json:"sessions_purged_count"`
	RecordsScrubbedCount   int       `json:"records_scrubbed_count"`
	DigitalSignature       string    `json:"digital_signature"`
	ComplianceStatement    string    `json:"compliance_statement"`
}
