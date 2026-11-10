package envelope

import (
	"context"
	"time"
	"go.opentelemetry.io/otel/trace"
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
// If the context contains a trace ID and correlationID is empty, it uses the trace ID as the correlationID.
func NewEventEnvelope[T any](ctx context.Context, eventID, eventType, correlationID string, payload T) EventEnvelope[T] {
	if correlationID == "" && ctx != nil {
		spanCtx := trace.SpanFromContext(ctx).SpanContext()
		if spanCtx.HasTraceID() {
			correlationID = spanCtx.TraceID().String()
		}
	}
	return EventEnvelope[T]{
		EventID:       eventID,
		EventType:     eventType,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
		Payload:       payload,
	}
}
