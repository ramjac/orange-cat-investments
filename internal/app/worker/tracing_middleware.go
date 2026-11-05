package worker

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TraceContextMiddleware extracts distributed trace context from incoming messages
// and starts a child consumer span for the worker handler execution.
// It explicitly injects the current active trace ID into the correlation_id field.
func TraceContextMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		ctx := envelope.ExtractTraceContext(context.Background(), msg)

		tracer := otel.Tracer("watermill-worker")
		ctx, span := tracer.Start(ctx, "worker-consumer", trace.WithSpanKind(trace.SpanKindConsumer))
		defer span.End()

		if span.SpanContext().HasTraceID() {
			traceID := span.SpanContext().TraceID().String()

			// Set correlation_id without logging payload
			var payloadMap map[string]interface{}
			if err := json.Unmarshal(msg.Payload, &payloadMap); err == nil {
				payloadMap["correlation_id"] = traceID
				if newPayload, err := json.Marshal(payloadMap); err == nil {
					msg.Payload = newPayload
				}
			}
		}

		msg.SetContext(ctx)
		return h(msg)
	}
}
