package worker

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TraceContextMiddleware extracts distributed trace context from incoming messages
// and starts a child consumer span for the worker handler execution.
// It sets the active trace ID into the message context, so that if the handler
// needs to publish another message or query a database, the parent trace context
// is correctly propagated. It does not overwrite the business correlation_id.
func TraceContextMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		ctx := envelope.ExtractTraceContext(msg.Context(), msg)

		tracer := otel.Tracer("watermill-worker")
		ctx, span := tracer.Start(ctx, "worker-consumer", trace.WithSpanKind(trace.SpanKindConsumer))
		defer span.End()

		// Do not mutate business payload correlation_id
		msg.SetContext(ctx)
		return h(msg)
	}
}
