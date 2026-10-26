package main

import (
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
