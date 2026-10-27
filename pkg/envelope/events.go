package envelope

import "time"

const (
	TopicOrderPlaced          = "events.investment.order_placed.v1"
	TopicTradeExecuted        = "events.investment.trade_executed.v1"
	TopicACHTransferRequested = "events.banking.ach_transfer_requested.v1"
	TopicACHSettled           = "events.banking.ach_settled.v1"
	TopicStatementGenerated   = "events.reporting.statements_generated.v1"
	TopicCatSpotted           = "events.observation.cat_spotted.v1"
	TopicOTATriggered         = "events.facilities.ota_triggered.v1"
	TopicOTACompleted         = "events.facilities.ota_completed.v1"
	TopicBacktestRun          = "events.investment.backtest_requested.v1"
	TopicBacktestCompleted    = "events.investment.backtest_completed.v1"
	TopicIncidentCreated      = "events.workforce.incident_created.v1"
	TopicIncidentAlerted      = "events.ops.incident_alerted.v1"
	TopicCareScheduleUpdated  = "events.workforce.care_schedule_updated.v1"
	TopicReportGenerated      = "events.reporting.report_generated.v1"
	TopicCARotated            = "events.ops.ca_rotated.v1"
)

type ACHTransferRequestedPayload struct {
	TransferID   string  `json:"transfer_id"`
	CustomerID   string  `json:"customer_id"`
	PortfolioID  string  `json:"portfolio_id"`
	BankAccount  string  `json:"bank_account"`
	AmountUSD    float64 `json:"amount_usd"`
	TransferType string  `json:"transfer_type"`
}

type ACHSettledPayload struct {
	TransferID  string    `json:"transfer_id"`
	CustomerID  string    `json:"customer_id"`
	PortfolioID string    `json:"portfolio_id"`
	AmountUSD   float64   `json:"amount_usd"`
	Status      string    `json:"status"`
	SettledAt   time.Time `json:"settled_at"`
}

type StatementGeneratedPayload struct {
	Period          string    `json:"period"`
	TotalPortfolios int       `json:"total_portfolios"`
	GeneratedAt     time.Time `json:"generated_at"`
	Status          string    `json:"status"`
}

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

type CatSpottedPayload struct {
	CameraID        string  `json:"camera_id"`
	FelineID        string  `json:"feline_id"`
	ActivityType    string  `json:"activity_type"`
	ConfidenceScore float64 `json:"confidence_score"`
}

type OTATriggeredPayload struct {
	JobID      string `json:"job_id"`
	AssetID    string `json:"asset_id"`
	ReleaseID  string `json:"release_id"`
	DeviceType string `json:"device_type"`
	Version    string `json:"version"`
	FileURL    string `json:"file_url"`
	Checksum   string `json:"checksum"`
}

type OTACompletedPayload struct {
	JobID       string    `json:"job_id"`
	AssetID     string    `json:"asset_id"`
	ReleaseID   string    `json:"release_id"`
	Status      string    `json:"status"`
	CompletedAt time.Time `json:"completed_at"`
}

type CARotationPayload struct {
	IntermediateCAName        string    `json:"intermediate_ca_name"`
	NewSerialNumber           string    `json:"new_serial_number"`
	ExpiresAt                 time.Time `json:"expires_at"`
	ReissuedCertificatesCount int       `json:"reissued_certificates_count"`
	Status                    string    `json:"status"`
}

type IncidentCreatedPayload struct {
	IncidentID       string    `json:"incident_id"`
	Title            string    `json:"title"`
	Category         string    `json:"category"`
	InvolvedFelineID string    `json:"involved_feline_id,omitempty"`
	InvolvedHumanID  string    `json:"involved_human_id,omitempty"`
	Severity         string    `json:"severity"`
	Status           string    `json:"status,omitempty"`
	RoomZone         string    `json:"room_zone"`
	Description      string    `json:"description"`
	ReportedAt       time.Time `json:"reported_at"`
}

type EmployeeOnboardedPayload struct {
	EmployeeID   string `json:"employee_id"`
	EmployeeType string `json:"employee_type"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Email        string `json:"email,omitempty"`
	RoleTitle    string `json:"role_title"`
	Department   string `json:"department"`
}

type GenerateMonthlyReportRequestedPayload struct {
	Period      string `json:"period"`
	RequestedBy string `json:"requested_by"`
}
