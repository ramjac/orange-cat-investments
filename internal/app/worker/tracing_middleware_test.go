package worker

import (
	"testing"
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"github.com/orange-cat-investments/oci/pkg/envelope"
)

func TestTraceContextMiddleware(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	ctx, span1 := tracer.Start(context.Background(), "publisher")

	payload := `{"correlation_id":"business-123","data":"test"}`
	msg := message.NewMessage("123", []byte(payload))
	msg.SetContext(ctx)
	envelope.InjectTraceContext(ctx, msg)
	span1.End()

	handler := TraceContextMiddleware(func(m *message.Message) ([]*message.Message, error) {
		assert.Equal(t, `{"correlation_id":"business-123","data":"test"}`, string(m.Payload))

		span := trace.SpanFromContext(m.Context())
		require.NotNil(t, span)

		assert.Equal(t, span1.SpanContext().TraceID(), span.SpanContext().TraceID())

		return nil, nil
	})

	_, err := handler(msg)
	assert.NoError(t, err)
}
