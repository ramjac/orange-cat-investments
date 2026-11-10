package envelope_test

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceContextPropagation(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	ctx, span1 := tracer.Start(context.Background(), "publisher")

	msg := message.NewMessage("123", []byte("payload"))
	envelope.InjectTraceContext(ctx, msg)
	span1.End()

	extractedCtx := envelope.ExtractTraceContext(context.Background(), msg)
	_, span2 := tracer.Start(extractedCtx, "subscriber")
	span2.End()

	spans := exp.GetSpans()
	require.Len(t, spans, 2)

	assert.Equal(t, spans[0].SpanContext.TraceID(), spans[1].SpanContext.TraceID())
	assert.Equal(t, spans[0].SpanContext.SpanID(), spans[1].Parent.SpanID())
}
