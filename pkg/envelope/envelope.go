package envelope

import (
	"time"
)

// EventEnvelope standardizes event messaging across RabbitMQ and internal event brokers.
// It wraps arbitrary domain event payloads with distributed tracing metadata.
type EventEnvelope[T any] struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
	Payload       T         `json:"payload"`
}

// NewEventEnvelope constructs a new EventEnvelope with a given ID, event type, correlation trace ID, and payload.
func NewEventEnvelope[T any](eventID, eventType, correlationID string, payload T) EventEnvelope[T] {
	return EventEnvelope[T]{
		EventID:       eventID,
		EventType:     eventType,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
		Payload:       payload,
	}
}
