package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventEnvelopeJSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	event := EventEnvelope[CatSpottedPayload]{
		EventID:       "evt-123",
		EventType:     "observation.cat_spotted.v1",
		OccurredAt:    now,
		CorrelationID: "corr-456",
		Payload: CatSpottedPayload{
			CameraID:        "CAM-01",
			FelineID:        "cat-01",
			ActivityType:    "napping",
			ConfidenceScore: 0.98,
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var unmarshaled EventEnvelope[CatSpottedPayload]
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, event.EventID, unmarshaled.EventID)
	assert.Equal(t, event.EventType, unmarshaled.EventType)
	assert.Equal(t, event.CorrelationID, unmarshaled.CorrelationID)
	assert.Equal(t, event.Payload, unmarshaled.Payload)
	assert.True(t, event.OccurredAt.Equal(unmarshaled.OccurredAt))
}

func TestWorkerRouter_CatSpottedHandler(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{OutputChannelBuffer: 10},
		watermillLogger,
	)

	router, err := createRouter(pubSub, watermillLogger, logger)
	require.NoError(t, err)
	require.NotNil(t, router)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to output topic
	allocationCheckMessages, err := pubSub.Subscribe(ctx, "events.investment.allocation_check.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	event := EventEnvelope[CatSpottedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "observation.cat_spotted.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "correlation-test",
		Payload: CatSpottedPayload{
			CameraID:        "CAM-ORANGE-01",
			FelineID:        "emp-feline-garfield",
			ActivityType:    "zooming",
			ConfidenceScore: 0.992,
		},
	}
	payloadBytes, err := json.Marshal(event)
	require.NoError(t, err)

	msg := message.NewMessage(event.EventID, payloadBytes)
	err = pubSub.Publish("events.observation.cat_spotted.v1", msg)
	require.NoError(t, err)

	select {
	case outMsg := <-allocationCheckMessages:
		outMsg.Ack()
		assert.JSONEq(t, `{"status":"brokerage_trade_executed"}`, string(outMsg.Payload))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for output message on events.investment.allocation_check.v1")
	}
}

func TestWorkerRouter_CatSpottedHandler_InvalidPayload(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{OutputChannelBuffer: 10},
		watermillLogger,
	)

	router, err := createRouter(pubSub, watermillLogger, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	allocationCheckMessages, err := pubSub.Subscribe(ctx, "events.investment.allocation_check.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	// Send malformed non-JSON payload
	msg := message.NewMessage(watermill.NewUUID(), []byte("invalid-json"))
	err = pubSub.Publish("events.observation.cat_spotted.v1", msg)
	require.NoError(t, err)

	select {
	case outMsg := <-allocationCheckMessages:
		outMsg.Ack()
		assert.JSONEq(t, `{"status":"brokerage_trade_executed"}`, string(outMsg.Payload))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for output message on events.investment.allocation_check.v1")
	}
}

func TestWorkerRouter_BacktestRunHandler(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{OutputChannelBuffer: 10},
		watermillLogger,
	)

	router, err := createRouter(pubSub, watermillLogger, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backtestCompletedMessages, err := pubSub.Subscribe(ctx, "events.investment.backtest_completed.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	msg := message.NewMessage(watermill.NewUUID(), []byte(`{"strategy_id":"strat-1"}`))
	err = pubSub.Publish("events.investment.backtest_requested.v1", msg)
	require.NoError(t, err)

	select {
	case outMsg := <-backtestCompletedMessages:
		outMsg.Ack()
		assert.Contains(t, string(outMsg.Payload), "backtest_completed")
		assert.Contains(t, string(outMsg.Payload), "sharpe_ratio")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for output message on events.investment.backtest_completed.v1")
	}
}

func BenchmarkEventPublishingOriginal(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event := EventEnvelope[CatSpottedPayload]{
			EventID:       watermill.NewUUID(),
			EventType:     "observation.cat_spotted.v1",
			OccurredAt:    time.Now().UTC(),
			CorrelationID: "correlation-trace-001",
			Payload: CatSpottedPayload{
				CameraID:        "CAM-ORANGE-01",
				FelineID:        "emp-feline-garfield",
				ActivityType:    "zooming",
				ConfidenceScore: 0.992,
			},
		}
		data, _ := json.Marshal(event)
		_ = message.NewMessage(event.EventID, data)
	}
}

func BenchmarkEventPublishingOptimized(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	event := EventEnvelope[CatSpottedPayload]{
		EventType:     "observation.cat_spotted.v1",
		CorrelationID: "correlation-trace-001",
		Payload: CatSpottedPayload{
			CameraID:        "CAM-ORANGE-01",
			FelineID:        "emp-feline-garfield",
			ActivityType:    "zooming",
			ConfidenceScore: 0.992,
		},
	}

	for i := 0; i < b.N; i++ {
		event.EventID = watermill.NewUUID()
		event.OccurredAt = time.Now().UTC()

		data, _ := json.Marshal(event)
		_ = message.NewMessage(event.EventID, data)
	}
}
