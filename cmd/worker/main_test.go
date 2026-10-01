package main

import (
	"bytes"
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

func TestWorkerRouter_OTAPipelineHandler(t *testing.T) {
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

	otaCompletedMessages, err := pubSub.Subscribe(ctx, "events.facilities.ota_completed.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	event := EventEnvelope[OTATriggeredPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_triggered.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "test-corr-ota",
		Payload: OTATriggeredPayload{
			JobID:      "job-router-1",
			AssetID:    "asset-router-2",
			ReleaseID:  "release-router-3",
			DeviceType: "edge_camera",
			Version:    "2.1.0",
			FileURL:    "https://firmware.oci.local/v2.1.0.bin",
			Checksum:   "sha256checksum",
		},
	}
	payloadBytes, err := json.Marshal(event)
	require.NoError(t, err)

	msg := message.NewMessage(event.EventID, payloadBytes)
	err = pubSub.Publish("events.facilities.ota_triggered.v1", msg)
	require.NoError(t, err)

	select {
	case outMsg := <-otaCompletedMessages:
		outMsg.Ack()
		var completed EventEnvelope[OTACompletedPayload]
		err := json.Unmarshal(outMsg.Payload, &completed)
		require.NoError(t, err)
		assert.Equal(t, "job-router-1", completed.Payload.JobID)
		assert.Equal(t, "asset-router-2", completed.Payload.AssetID)
		assert.Equal(t, "completed", completed.Payload.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for output message on events.facilities.ota_completed.v1")
	}
}

func TestHandleCatSpotted_DoesNotLogRawPayload(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	secretPayloadStr := `{"secret_key":"TOP_SECRET_PAYLOAD_DATA","payload":{"camera_id":"CAM-1","feline_id":"garfield","activity_type":"zooming","confidence_score":0.99}}`
	msg := message.NewMessage("test-msg-uuid-123", []byte(secretPayloadStr))

	outMsgs, err := handleCatSpotted(logger, msg)
	require.NoError(t, err)
	require.Len(t, outMsgs, 1)

	logs := buf.String()
	assert.NotContains(t, logs, "TOP_SECRET_PAYLOAD_DATA")
	assert.Contains(t, logs, "test-msg-uuid-123")
}

func TestHandleBacktestRun_DoesNotLogRawPayload(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	secretPayloadStr := `{"secret_strategy":"CLASSIFIED_QUANT_MODEL"}`
	msg := message.NewMessage("test-msg-uuid-456", []byte(secretPayloadStr))

	outMsgs, err := handleBacktestRun(logger, msg)
	require.NoError(t, err)
	require.Len(t, outMsgs, 1)

	logs := buf.String()
	assert.NotContains(t, logs, "CLASSIFIED_QUANT_MODEL")
	assert.Contains(t, logs, "test-msg-uuid-456")
}

func TestHandleOTATriggered(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	event := EventEnvelope[OTATriggeredPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_triggered.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "test-corr-id",
		Payload: OTATriggeredPayload{
			JobID:      "job-123",
			AssetID:    "asset-456",
			ReleaseID:  "release-789",
			DeviceType: "edge_camera",
			Version:    "2.1.0",
			FileURL:    "https://firmware.oci.local/v2.1.0.bin",
			Checksum:   "sha256checksum",
		},
	}

	data, err := json.Marshal(event)
	assert.NoError(t, err)

	msg := message.NewMessage(event.EventID, data)
	outputMsgs, err := HandleOTATriggered(logger, msg)

	assert.NoError(t, err)
	assert.Len(t, outputMsgs, 1)

	var completionEvent EventEnvelope[OTACompletedPayload]
	err = json.Unmarshal(outputMsgs[0].Payload, &completionEvent)
	assert.NoError(t, err)
	assert.Equal(t, "job-123", completionEvent.Payload.JobID)
	assert.Equal(t, "asset-456", completionEvent.Payload.AssetID)
	assert.Equal(t, "completed", completionEvent.Payload.Status)
}

func TestHandleOTATriggered_DoesNotLogRawPayload(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	secretPayloadStr := `{"secret_ota_token":"SUPER_SECRET_OTA_KEY","payload":{"job_id":"job-1","asset_id":"asset-1","release_id":"rel-1","device_type":"edge_cam","version":"1.0","file_url":"http://url","checksum":"chk"}}`
	msg := message.NewMessage("test-msg-uuid-789", []byte(secretPayloadStr))

	outMsgs, err := HandleOTATriggered(logger, msg)
	require.NoError(t, err)
	require.Len(t, outMsgs, 1)

	logs := buf.String()
	assert.NotContains(t, logs, "SUPER_SECRET_OTA_KEY")
	assert.Contains(t, logs, "test-msg-uuid-789")
}

func TestWorkerRouter_FeatureFlags(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	t.Run("DefaultEnabled", func(t *testing.T) {
		router, err := createRouter(pubSub, watermillLogger, logger)
		require.NoError(t, err)
		handlers := router.Handlers()
		assert.Contains(t, handlers, "cat_spotted_handler")
		assert.Contains(t, handlers, "backtest_run_handler")
		assert.Contains(t, handlers, "ota_pipeline_handler")
	})

	t.Run("FelineDisabled", func(t *testing.T) {
		t.Setenv("ENABLE_FELINE_WORKFORCE", "false")
		router, err := createRouter(pubSub, watermillLogger, logger)
		require.NoError(t, err)
		handlers := router.Handlers()
		assert.NotContains(t, handlers, "cat_spotted_handler")
		assert.Contains(t, handlers, "backtest_run_handler")
		assert.Contains(t, handlers, "ota_pipeline_handler")
	})

	t.Run("CoreInvestDisabled", func(t *testing.T) {
		t.Setenv("ENABLE_CORE_INVEST", "false")
		router, err := createRouter(pubSub, watermillLogger, logger)
		require.NoError(t, err)
		handlers := router.Handlers()
		assert.NotContains(t, handlers, "cat_spotted_handler")
		assert.NotContains(t, handlers, "backtest_run_handler")
		assert.Contains(t, handlers, "ota_pipeline_handler")
	})

	t.Run("SmallBusinessMode", func(t *testing.T) {
		t.Setenv("ENABLE_SMALL_BUSINESS_MODE", "true")
		t.Setenv("ENABLE_CORE_INVEST", "")
		t.Setenv("ENABLE_FELINE_WORKFORCE", "")
		router, err := createRouter(pubSub, watermillLogger, logger)
		require.NoError(t, err)
		handlers := router.Handlers()
		assert.NotContains(t, handlers, "cat_spotted_handler")
		assert.NotContains(t, handlers, "backtest_run_handler")
		assert.Contains(t, handlers, "ota_pipeline_handler")
	})
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
