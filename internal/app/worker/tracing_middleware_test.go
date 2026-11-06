package worker

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceContextMiddleware_GoChannelPubSub(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	ctx, span1 := tracer.Start(context.Background(), "publisher")
	defer span1.End()

	wmLogger := watermill.NewStdLogger(false, false)
	pubsub := gochannel.NewGoChannel(gochannel.Config{}, wmLogger)

	tracingPub := envelope.NewPublisherWithTracing(pubsub)

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	require.NoError(t, err)

	router.AddMiddleware(TraceContextMiddleware)

	receivedCh := make(chan struct{})

	router.AddHandler(
		"test_handler",
		"test_topic",
		pubsub,
		"test_output",
		tracingPub,
		func(msg *message.Message) ([]*message.Message, error) {
			extractedSpan := trace.SpanFromContext(msg.Context())

			assert.Equal(t, span1.SpanContext().TraceID(), extractedSpan.SpanContext().TraceID())
			close(receivedCh)
			return nil, nil
		},
	)

	go func() {
		_ = router.Run(context.Background())
	}()

	<-router.Running()

	msg := message.NewMessage("msg-id", []byte(`{}`))
	msg.SetContext(ctx)

	err = tracingPub.Publish("test_topic", msg)
	require.NoError(t, err)

	select {
	case <-receivedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	_ = router.Close()
}
