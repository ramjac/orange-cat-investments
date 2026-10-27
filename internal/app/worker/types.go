package worker

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// PubSub defines message publishing and subscribing capabilities.
type PubSub interface {
	message.Publisher
	message.Subscriber
}

// EventEnvelope standardizes event messaging schema across the platform.
type EventEnvelope[T any] struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
	Payload       T         `json:"payload"`
}

// ACHTransferRequestedPayload is payload for events.banking.ach_transfer_requested.v1.
type ACHTransferRequestedPayload struct {
	TransferID   string  `json:"transfer_id"`
	CustomerID   string  `json:"customer_id"`
	PortfolioID  string  `json:"portfolio_id"`
	BankAccount  string  `json:"bank_account"`
	AmountUSD    float64 `json:"amount_usd"`
	TransferType string  `json:"transfer_type"`
}

// ACHSettledPayload is payload for events.banking.ach_settled.v1.
type ACHSettledPayload struct {
	TransferID  string    `json:"transfer_id"`
	CustomerID  string    `json:"customer_id"`
	PortfolioID string    `json:"portfolio_id"`
	AmountUSD   float64   `json:"amount_usd"`
	Status      string    `json:"status"`
	SettledAt   time.Time `json:"settled_at"`
}

// StatementGeneratedPayload is payload for events.reporting.statements_generated.v1.
type StatementGeneratedPayload struct {
	Period          string    `json:"period"`
	TotalPortfolios int       `json:"total_portfolios"`
	GeneratedAt     time.Time `json:"generated_at"`
	Status          string    `json:"status"`
}

// ExecutiveReportPayload is payload for events.reporting.report_generated.v1.
type ExecutiveReportPayload struct {
	ReportID        string  `json:"report_id"`
	Period          string  `json:"period"`
	Status          string  `json:"status"`
	WebDAVPathPDF   string  `json:"webdav_path_pdf"`
	WebDAVPathCSV   string  `json:"webdav_path_csv"`
	NetProfit       float64 `json:"net_profit"`
	OperationalBurn float64 `json:"operational_burn"`
	CashReserves    float64 `json:"cash_reserves"`
	Headcount       int     `json:"headcount"`
}

// CatSpottedPayload is payload for events.observation.cat_spotted.v1.
type CatSpottedPayload struct {
	CameraID        string  `json:"camera_id"`
	FelineID        string  `json:"feline_id"`
	ActivityType    string  `json:"activity_type"`
	ConfidenceScore float64 `json:"confidence_score"`
}

// OTATriggeredPayload is payload for events.facilities.ota_triggered.v1.
type OTATriggeredPayload struct {
	JobID      string `json:"job_id"`
	AssetID    string `json:"asset_id"`
	ReleaseID  string `json:"release_id"`
	DeviceType string `json:"device_type"`
	Version    string `json:"version"`
	FileURL    string `json:"file_url"`
	Checksum   string `json:"checksum"`
}

// OTACompletedPayload is payload for events.facilities.ota_completed.v1.
type OTACompletedPayload struct {
	JobID       string    `json:"job_id"`
	AssetID     string    `json:"asset_id"`
	ReleaseID   string    `json:"release_id"`
	Status      string    `json:"status"`
	CompletedAt time.Time `json:"completed_at"`
}

// CARotationPayload is payload for ops.ca_rotated.v1.
type CARotationPayload struct {
	IntermediateCAName       string    `json:"intermediate_ca_name"`
	NewSerialNumber           string    `json:"new_serial_number"`
	ExpiresAt                 time.Time `json:"expires_at"`
	ReissuedCertificatesCount int       `json:"reissued_certificates_count"`
	Status                    string    `json:"status"`
}

// BacktestRequestedPayload is payload for events.investment.backtest_requested.v1.
type BacktestRequestedPayload struct {
	BacktestID string `json:"backtest_id"`
	StrategyID string `json:"strategy_id"`
	RunID      string `json:"run_id,omitempty"`
}

// BacktestCompletedPayload is payload for events.investment.backtest_completed.v1.
type BacktestCompletedPayload struct {
	RunID       string    `json:"run_id"`
	StrategyID  string    `json:"strategy_id,omitempty"`
	Status      string    `json:"status"`
	SharpeRatio float64   `json:"sharpe_ratio"`
	Alpha       float64   `json:"alpha"`
	CompletedAt time.Time `json:"completed_at"`
}

// IncidentCreatedEventPayload is payload for events.workforce.incident_created.v1.
type IncidentCreatedEventPayload struct {
	IncidentID       string    `json:"incident_id"`
	Title            string    `json:"title"`
	Category         string    `json:"category"`
	InvolvedFelineID string    `json:"involved_feline_id,omitempty"`
	InvolvedHumanID  string    `json:"involved_human_id,omitempty"`
	Severity         string    `json:"severity"`
	RoomZone         string    `json:"room_zone"`
	Description      string    `json:"description"`
	ReportedAt       time.Time `json:"reported_at"`
}
