package worker

import (
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
)

// PubSub defines message publishing and subscribing capabilities.
type PubSub = envelope.PubSub

// EventEnvelope standardizes event messaging schema across the platform.
type EventEnvelope[T any] = envelope.EventEnvelope[T]

// ACHTransferRequestedPayload is payload for events.banking.ach_transfer_requested.v1.
type ACHTransferRequestedPayload = envelope.ACHTransferRequestedPayload

// ACHSettledPayload is payload for events.banking.ach_settled.v1.
type ACHSettledPayload = envelope.ACHSettledPayload

// StatementGeneratedPayload is payload for events.reporting.statements_generated.v1.
type StatementGeneratedPayload = envelope.StatementGeneratedPayload

// ExecutiveReportPayload is payload for events.reporting.report_generated.v1.
type ExecutiveReportPayload = envelope.ExecutiveReportPayload

// CatSpottedPayload is payload for events.observation.cat_spotted.v1.
type CatSpottedPayload = envelope.CatSpottedPayload

// OTATriggeredPayload is payload for events.facilities.ota_triggered.v1.
type OTATriggeredPayload = envelope.OTATriggeredPayload

// OTACompletedPayload is payload for events.facilities.ota_completed.v1.
type OTACompletedPayload = envelope.OTACompletedPayload

// CARotationPayload is payload for ops.ca_rotated.v1.
type CARotationPayload = envelope.CARotationPayload

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
