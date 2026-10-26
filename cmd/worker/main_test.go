package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/internal/app/worker"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCustomerBLLClient struct {
	customerbllv1.CustomerBLLServiceClient
	settleFunc    func(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error)
	claimFunc     func(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error)
	batchStmtFunc func(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error)
}

func (m *mockCustomerBLLClient) SettleACHTransfer(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
	if m.settleFunc != nil {
		return m.settleFunc(ctx, in)
	}
	return &customerbllv1.SettleACHTransferResponse{
		TransferID:  in.TransferID,
		CustomerID:  in.CustomerID,
		PortfolioID: in.PortfolioID,
		AmountUSD:   in.AmountUSD,
		Status:      "settled",
		SettledAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (m *mockCustomerBLLClient) ClaimDueRecurringDeposits(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error) {
	if m.claimFunc != nil {
		return m.claimFunc(ctx, in)
	}
	return &customerbllv1.ClaimDueRecurringDepositsResponse{}, nil
}

func (m *mockCustomerBLLClient) GenerateBatchStatements(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error) {
	if m.batchStmtFunc != nil {
		return m.batchStmtFunc(ctx, in)
	}
	return &customerbllv1.GenerateBatchStatementsResponse{
		Period:          in.Period,
		TotalPortfolios: 2,
		Status:          "completed",
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}

type mockInternalBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	generateReportFunc func(ctx context.Context, in *internalbllv1.GenerateMonthlyReportRequest) (*internalbllv1.GenerateMonthlyReportResponse, error)
	checkReportFunc    func(ctx context.Context, in *internalbllv1.CheckExecutiveReportRequest) (*internalbllv1.CheckExecutiveReportResponse, error)
	getFelineFunc      func(ctx context.Context, in *internalbllv1.GetFelineTargetRequest) (*internalbllv1.GetFelineTargetResponse, error)
	broadcastFunc      func(ctx context.Context, in *internalbllv1.BroadcastPebbleIncidentAlertRequest) (*internalbllv1.BroadcastPebbleIncidentAlertResponse, error)
}

func (m *mockInternalBLLClient) GenerateMonthlyReport(ctx context.Context, in *internalbllv1.GenerateMonthlyReportRequest) (*internalbllv1.GenerateMonthlyReportResponse, error) {
	if m.generateReportFunc != nil {
		return m.generateReportFunc(ctx, in)
	}
	return &internalbllv1.GenerateMonthlyReportResponse{
		Report: &internalbllv1.ExecutiveReportSummary{
			ReportID:        "rep-12345",
			Period:          in.Period,
			Status:          "archived",
			TotalHeadcount:  25,
			NetProfit:       450000.0,
			OperationalBurn: 120000.0,
			CashReserves:    1500000.0,
		},
	}, nil
}

func (m *mockInternalBLLClient) CheckExecutiveReport(ctx context.Context, in *internalbllv1.CheckExecutiveReportRequest) (*internalbllv1.CheckExecutiveReportResponse, error) {
	if m.checkReportFunc != nil {
		return m.checkReportFunc(ctx, in)
	}
	return &internalbllv1.CheckExecutiveReportResponse{Exists: false}, nil
}

func (m *mockInternalBLLClient) GetFelineTarget(ctx context.Context, in *internalbllv1.GetFelineTargetRequest) (*internalbllv1.GetFelineTargetResponse, error) {
	if m.getFelineFunc != nil {
		return m.getFelineFunc(ctx, in)
	}
	return &internalbllv1.GetFelineTargetResponse{FelineID: "emp-feline-garfield"}, nil
}

func (m *mockInternalBLLClient) BroadcastPebbleIncidentAlert(ctx context.Context, in *internalbllv1.BroadcastPebbleIncidentAlertRequest) (*internalbllv1.BroadcastPebbleIncidentAlertResponse, error) {
	if m.broadcastFunc != nil {
		return m.broadcastFunc(ctx, in)
	}
	return &internalbllv1.BroadcastPebbleIncidentAlertResponse{TicketID: "tick-001", Status: "broadcasted"}, nil
}

func TestEventEnvelopeJSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	event := worker.EventEnvelope[worker.CatSpottedPayload]{
		EventID:       "evt-123",
		EventType:     "observation.cat_spotted.v1",
		OccurredAt:    now,
		CorrelationID: "corr-456",
		Payload: worker.CatSpottedPayload{
			CameraID:        "CAM-01",
			FelineID:        "cat-01",
			ActivityType:    "napping",
			ConfidenceScore: 0.98,
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var unmarshaled worker.EventEnvelope[worker.CatSpottedPayload]
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
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
	require.NoError(t, err)
	require.NotNil(t, router)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	allocationCheckMessages, err := pubSub.Subscribe(ctx, "events.investment.allocation_check.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	event := worker.EventEnvelope[worker.CatSpottedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "observation.cat_spotted.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "correlation-test",
		Payload: worker.CatSpottedPayload{
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
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	allocationCheckMessages, err := pubSub.Subscribe(ctx, "events.investment.allocation_check.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

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
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
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
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	otaCompletedMessages, err := pubSub.Subscribe(ctx, "events.facilities.ota_completed.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	event := worker.EventEnvelope[worker.OTATriggeredPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_triggered.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "test-corr-ota",
		Payload: worker.OTATriggeredPayload{
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
		var completed worker.EventEnvelope[worker.OTACompletedPayload]
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

	outMsgs, err := worker.HandleCatSpotted(logger, msg)
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

	outMsgs, err := worker.HandleBacktestRun(logger, msg)
	require.NoError(t, err)
	require.Len(t, outMsgs, 1)

	logs := buf.String()
	assert.NotContains(t, logs, "CLASSIFIED_QUANT_MODEL")
	assert.Contains(t, logs, "test-msg-uuid-456")
}

func TestHandleOTATriggered(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	event := worker.EventEnvelope[worker.OTATriggeredPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_triggered.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "test-corr-id",
		Payload: worker.OTATriggeredPayload{
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
	outputMsgs, err := worker.HandleOTATriggered(logger, msg)

	assert.NoError(t, err)
	assert.Len(t, outputMsgs, 1)

	var completionEvent worker.EventEnvelope[worker.OTACompletedPayload]
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

	outMsgs, err := worker.HandleOTATriggered(logger, msg)
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
		router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
		require.NoError(t, err)
		handlers := router.Handlers()
		assert.Contains(t, handlers, "cat_spotted_handler")
		assert.Contains(t, handlers, "backtest_run_handler")
		assert.Contains(t, handlers, "ota_pipeline_handler")
	})

	t.Run("FelineDisabled", func(t *testing.T) {
		t.Setenv("ENABLE_FELINE_WORKFORCE", "false")
		router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
		require.NoError(t, err)
		handlers := router.Handlers()
		assert.NotContains(t, handlers, "cat_spotted_handler")
		assert.Contains(t, handlers, "backtest_run_handler")
		assert.Contains(t, handlers, "ota_pipeline_handler")
	})

	t.Run("CoreInvestDisabled", func(t *testing.T) {
		t.Setenv("ENABLE_CORE_INVEST", "false")
		router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
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
		router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, nil, "", "", nil)
		require.NoError(t, err)
		handlers := router.Handlers()
		assert.NotContains(t, handlers, "cat_spotted_handler")
		assert.NotContains(t, handlers, "backtest_run_handler")
		assert.Contains(t, handlers, "ota_pipeline_handler")
	})
}

func TestWorkerRouter_ExecutiveReportingHandler(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	mockBLL := &mockInternalBLLClient{}
	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, mockBLL, "", "", nil)
	require.NoError(t, err)
	require.NotNil(t, router)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = router.Run(ctx)
	}()

	select {
	case <-router.Running():
	case <-time.After(2 * time.Second):
		t.Fatal("router failed to start in time")
	}

	sub, err := pubSub.Subscribe(context.Background(), "events.reporting.report_generated.v1")
	require.NoError(t, err)

	triggerPayload := map[string]string{
		"period":       "2026-10",
		"generated_by": "unit-test-worker",
	}
	payloadBytes, err := json.Marshal(triggerPayload)
	require.NoError(t, err)

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	err = pubSub.Publish("events.reporting.generate_monthly_requested.v1", msg)
	require.NoError(t, err)

	select {
	case resultMsg := <-sub:
		require.NotNil(t, resultMsg)
		var envelope worker.EventEnvelope[worker.ExecutiveReportPayload]
		err := json.Unmarshal(resultMsg.Payload, &envelope)
		require.NoError(t, err)
		assert.Equal(t, "events.reporting.report_generated.v1", envelope.EventType)
		assert.Equal(t, "2026-10", envelope.Payload.Period)
		assert.Equal(t, "archived", envelope.Payload.Status)
		assert.Equal(t, "rep-12345", envelope.Payload.ReportID)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for report_generated event")
	}

	require.NoError(t, router.Close())
}

func TestWorkerRouter_ACHSettlementHandler(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	mockCustomerBLL := &mockCustomerBLLClient{}
	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, mockCustomerBLL, nil, "", "", nil)
	require.NoError(t, err)
	require.NotNil(t, router)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settledMessages, err := pubSub.Subscribe(ctx, "events.banking.ach_settled.v1")
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()

	select {
	case <-router.Running():
	case <-time.After(2 * time.Second):
		t.Fatal("router failed to start in time")
	}

	transferReq := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.banking.ach_transfer_requested.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "test-ach-trace",
		Payload: worker.ACHTransferRequestedPayload{
			TransferID:   "ach-transfer-test-1",
			CustomerID:   "cust-user-1",
			PortfolioID:  "port-user-1",
			BankAccount:  "act-bank-999",
			AmountUSD:    250.00,
			TransferType: "deposit",
		},
	}
	payloadBytes, err := json.Marshal(transferReq)
	require.NoError(t, err)

	msg := message.NewMessage(transferReq.EventID, payloadBytes)
	err = pubSub.Publish("events.banking.ach_transfer_requested.v1", msg)
	require.NoError(t, err)

	select {
	case outMsg := <-settledMessages:
		outMsg.Ack()
		var settled worker.EventEnvelope[worker.ACHSettledPayload]
		err := json.Unmarshal(outMsg.Payload, &settled)
		require.NoError(t, err)
		assert.Equal(t, "events.banking.ach_settled.v1", settled.EventType)
		assert.Equal(t, "ach-transfer-test-1", settled.Payload.TransferID)
		assert.Equal(t, "settled", settled.Payload.Status)
		assert.Equal(t, 250.00, settled.Payload.AmountUSD)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for events.banking.ach_settled.v1")
	}

	require.NoError(t, router.Close())
}

func TestWorkerRouter_IncidentAlertPipelineHandler(t *testing.T) {
	bffReceived := make(chan bool, 1)
	pebbleReceived := make(chan bool, 1)

	bffServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/employee/incidents/broadcast" {
			bffReceived <- true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"broadcasted"}`))
		}
	}))
	defer bffServer.Close()

	pebbleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/ops/pebble/incidents/broadcast" {
			pebbleReceived <- true
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"created"}`))
		}
	}))
	defer pebbleServer.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	mockInternalBLL := &mockInternalBLLClient{}
	router, err := worker.CreateRouter(pubSub, watermillLogger, logger, nil, mockInternalBLL, bffServer.URL, pebbleServer.URL, bffServer.Client())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	incidentEnv := worker.EventEnvelope[worker.IncidentCreatedEventPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.workforce.incident_created.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "corr-inc-001",
		Payload: worker.IncidentCreatedEventPayload{
			IncidentID:       "inc-workmill-test-01",
			Title:            "Treat Stash Raid in Progress",
			Category:         "facilities",
			InvolvedFelineID: "emp-feline-garfield",
			Severity:         "critical",
			RoomZone:         "Breakroom Cabinet",
			Description:      "Cabinet doors forced open, all salmon bites consumed",
			ReportedAt:       time.Now().UTC(),
		},
	}
	bodyBytes, err := json.Marshal(incidentEnv)
	require.NoError(t, err)

	msg := message.NewMessage(incidentEnv.EventID, bodyBytes)
	err = pubSub.Publish("events.workforce.incident_created.v1", msg)
	require.NoError(t, err)

	select {
	case <-bffReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for incident fanout to Employee BFF")
	}

	select {
	case <-pebbleReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for incident fanout to Pebble Proxy")
	}

	require.NoError(t, router.Close())
}

func TestHandleACHTransferRequested_Success(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	var bllCalled bool
	mockBLL := &mockCustomerBLLClient{
		settleFunc: func(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
			assert.Equal(t, "ach-trans-123", in.TransferID)
			assert.Equal(t, 250.0, in.AmountUSD)
			assert.Equal(t, "port-001", in.PortfolioID)
			bllCalled = true
			return &customerbllv1.SettleACHTransferResponse{
				TransferID:  in.TransferID,
				CustomerID:  in.CustomerID,
				PortfolioID: in.PortfolioID,
				AmountUSD:   in.AmountUSD,
				Status:      "settled",
				SettledAt:   time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}

	payload := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "events.banking.ach_transfer_requested.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "test-corr-ach",
		Payload: worker.ACHTransferRequestedPayload{
			TransferID:   "ach-trans-123",
			CustomerID:   "cust-001",
			PortfolioID:  "port-001",
			BankAccount:  "bank-act-999",
			AmountUSD:    250.0,
			TransferType: "deposit",
		},
	}
	data, _ := json.Marshal(payload)
	msg := message.NewMessage(payload.EventID, data)

	outMsgs, err := worker.HandleACHTransferRequested(context.Background(), logger, mockBLL, msg)
	require.NoError(t, err)
	require.Len(t, outMsgs, 1)
	assert.True(t, bllCalled, "Customer BLL SettleACHTransfer should be called")

	var settled worker.EventEnvelope[worker.ACHSettledPayload]
	err = json.Unmarshal(outMsgs[0].Payload, &settled)
	require.NoError(t, err)
	assert.Equal(t, "ach-trans-123", settled.Payload.TransferID)
	assert.Equal(t, "settled", settled.Payload.Status)
}

func TestHandleACHTransferRequested_BankFailure(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	mockBLL := &mockCustomerBLLClient{
		settleFunc: func(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
			return nil, errors.New("bank clearing rejected: insufficient funds")
		},
	}

	payload := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
		EventID:   watermill.NewUUID(),
		EventType: "events.banking.ach_transfer_requested.v1",
		Payload: worker.ACHTransferRequestedPayload{
			TransferID:   "ach-fail-001",
			PortfolioID:  "port-001",
			AmountUSD:    100.0,
			TransferType: "deposit",
		},
	}
	data, _ := json.Marshal(payload)
	msg := message.NewMessage(payload.EventID, data)

	outMsgs, err := worker.HandleACHTransferRequested(context.Background(), logger, mockBLL, msg)
	assert.Error(t, err)
	assert.Nil(t, outMsgs)
	assert.Contains(t, err.Error(), "bank clearing rejected")
}

func TestRunACHRecurringBatch(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	mockBLL := &mockCustomerBLLClient{
		claimFunc: func(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error) {
			return &customerbllv1.ClaimDueRecurringDepositsResponse{
				DueDeposits: []*customerbllv1.RecurringDepositSummary{
					{
						DepositID:   "dep-due",
						CustomerID:  "cust-01",
						PortfolioID: "port-01",
						AmountUSD:   500.0,
						BankAccount: "act-01",
					},
				},
			}, nil
		},
	}

	ch, err := pubSub.Subscribe(context.Background(), "events.banking.ach_transfer_requested.v1")
	require.NoError(t, err)

	err = worker.RunACHRecurringBatch(context.Background(), mockBLL, pubSub, logger)
	require.NoError(t, err)

	select {
	case msg := <-ch:
		msg.Ack()
		assert.Contains(t, string(msg.Payload), "dep-due")
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for recurring ACH event")
	}
}

func TestRunEOMStatementBatch(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	var bllBatchCalled bool
	mockBLL := &mockCustomerBLLClient{
		batchStmtFunc: func(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error) {
			bllBatchCalled = true
			assert.Equal(t, int32(2026), in.Year)
			assert.Equal(t, "2026-10", in.Period)
			return &customerbllv1.GenerateBatchStatementsResponse{
				Period:          in.Period,
				TotalPortfolios: 2,
				Status:          "completed",
				GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}

	ch, err := pubSub.Subscribe(context.Background(), "events.reporting.statements_generated.v1")
	require.NoError(t, err)

	now := time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC)
	err = worker.RunEOMStatementBatch(context.Background(), mockBLL, pubSub, logger, now)
	require.NoError(t, err)
	assert.True(t, bllBatchCalled, "Customer BLL GenerateBatchStatements should be called")

	select {
	case msg := <-ch:
		msg.Ack()
		var env worker.EventEnvelope[worker.StatementGeneratedPayload]
		_ = json.Unmarshal(msg.Payload, &env)
		assert.Equal(t, "2026-10", env.Payload.Period, "should target previous month")
		assert.Equal(t, 2, env.Payload.TotalPortfolios, "derived total should match actual generated count")
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for statement generated event")
	}
}
