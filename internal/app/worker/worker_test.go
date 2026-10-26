package worker_test

import (
	"context"
	"encoding/json"
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

type mockCustomerBLL struct {
	customerbllv1.CustomerBLLServiceClient
	settleFunc    func(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error)
	claimFunc     func(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error)
	batchStmtFunc func(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error)
}

func (m *mockCustomerBLL) SettleACHTransfer(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
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

func (m *mockCustomerBLL) ClaimDueRecurringDeposits(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error) {
	if m.claimFunc != nil {
		return m.claimFunc(ctx, in)
	}
	return &customerbllv1.ClaimDueRecurringDepositsResponse{}, nil
}

func (m *mockCustomerBLL) GenerateBatchStatements(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error) {
	if m.batchStmtFunc != nil {
		return m.batchStmtFunc(ctx, in)
	}
	return &customerbllv1.GenerateBatchStatementsResponse{
		Period:          in.Period,
		TotalPortfolios: 1,
		Status:          "completed",
	}, nil
}

type mockInternalBLL struct {
	internalbllv1.InternalBLLServiceClient
	generateReportFunc func(ctx context.Context, in *internalbllv1.GenerateMonthlyReportRequest) (*internalbllv1.GenerateMonthlyReportResponse, error)
	checkReportFunc    func(ctx context.Context, in *internalbllv1.CheckExecutiveReportRequest) (*internalbllv1.CheckExecutiveReportResponse, error)
	getFelineFunc      func(ctx context.Context, in *internalbllv1.GetFelineTargetRequest) (*internalbllv1.GetFelineTargetResponse, error)
	broadcastFunc      func(ctx context.Context, in *internalbllv1.BroadcastPebbleIncidentAlertRequest) (*internalbllv1.BroadcastPebbleIncidentAlertResponse, error)
}

func (m *mockInternalBLL) GenerateMonthlyReport(ctx context.Context, in *internalbllv1.GenerateMonthlyReportRequest) (*internalbllv1.GenerateMonthlyReportResponse, error) {
	if m.generateReportFunc != nil {
		return m.generateReportFunc(ctx, in)
	}
	return &internalbllv1.GenerateMonthlyReportResponse{
		Report: &internalbllv1.ExecutiveReportSummary{
			ReportID:        "rep-test-99",
			Period:          in.Period,
			Status:          "archived",
			TotalHeadcount:  30,
			NetProfit:       500000.0,
			OperationalBurn: 150000.0,
			CashReserves:    2000000.0,
		},
	}, nil
}

func (m *mockInternalBLL) CheckExecutiveReport(ctx context.Context, in *internalbllv1.CheckExecutiveReportRequest) (*internalbllv1.CheckExecutiveReportResponse, error) {
	if m.checkReportFunc != nil {
		return m.checkReportFunc(ctx, in)
	}
	return &internalbllv1.CheckExecutiveReportResponse{Exists: false}, nil
}

func (m *mockInternalBLL) GetFelineTarget(ctx context.Context, in *internalbllv1.GetFelineTargetRequest) (*internalbllv1.GetFelineTargetResponse, error) {
	if m.getFelineFunc != nil {
		return m.getFelineFunc(ctx, in)
	}
	return &internalbllv1.GetFelineTargetResponse{FelineID: "emp-feline-garfield"}, nil
}

func (m *mockInternalBLL) BroadcastPebbleIncidentAlert(ctx context.Context, in *internalbllv1.BroadcastPebbleIncidentAlertRequest) (*internalbllv1.BroadcastPebbleIncidentAlertResponse, error) {
	if m.broadcastFunc != nil {
		return m.broadcastFunc(ctx, in)
	}
	return &internalbllv1.BroadcastPebbleIncidentAlertResponse{TicketID: "tick-001", Status: "broadcasted"}, nil
}

func TestConsumersInvest(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	t.Run("HandleCatSpotted", func(t *testing.T) {
		msg := message.NewMessage("test-uuid", []byte(`{"payload":{"feline_id":"f-1","activity_type":"zooming"}}`))
		outMsgs, err := worker.HandleCatSpotted(logger, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)
		assert.JSONEq(t, `{"status":"brokerage_trade_executed"}`, string(outMsgs[0].Payload))
	})

	t.Run("HandleBacktestRun", func(t *testing.T) {
		msg := message.NewMessage("test-uuid", []byte(`{"strategy":"alpha"}`))
		outMsgs, err := worker.HandleBacktestRun(logger, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)
		assert.Contains(t, string(outMsgs[0].Payload), "backtest_completed")
	})

	t.Run("HandleACHTransferRequested", func(t *testing.T) {
		mockBLL := &mockCustomerBLL{}
		req := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
			EventID: "evt-1",
			Payload: worker.ACHTransferRequestedPayload{
				TransferID:   "ach-1",
				CustomerID:   "cust-1",
				PortfolioID:  "port-1",
				AmountUSD:    100.0,
				TransferType: "deposit",
			},
		}
		data, _ := json.Marshal(req)
		msg := message.NewMessage("msg-1", data)

		outMsgs, err := worker.HandleACHTransferRequested(context.Background(), logger, mockBLL, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)

		var settled worker.EventEnvelope[worker.ACHSettledPayload]
		err = json.Unmarshal(outMsgs[0].Payload, &settled)
		require.NoError(t, err)
		assert.Equal(t, "ach-1", settled.Payload.TransferID)
		assert.Equal(t, "settled", settled.Payload.Status)
	})

	t.Run("HandleACHTransferRequested_NilClient", func(t *testing.T) {
		req := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
			EventID: "evt-2",
			Payload: worker.ACHTransferRequestedPayload{
				TransferID: "ach-2",
				AmountUSD:  50.0,
			},
		}
		data, _ := json.Marshal(req)
		msg := message.NewMessage("msg-2", data)

		outMsgs, err := worker.HandleACHTransferRequested(context.Background(), logger, nil, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)
	})
}

func TestConsumersOpsWorkforce(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	t.Run("HandleOTATriggered", func(t *testing.T) {
		otaReq := worker.EventEnvelope[worker.OTATriggeredPayload]{
			EventID: "evt-ota-1",
			Payload: worker.OTATriggeredPayload{
				JobID:      "job-1",
				AssetID:    "asset-1",
				ReleaseID:  "rel-1",
				DeviceType: "camera",
				Version:    "1.2.0",
				FileURL:    "http://firmware/1.2.0.bin",
				Checksum:   "abc",
			},
		}
		data, _ := json.Marshal(otaReq)
		msg := message.NewMessage("msg-ota", data)

		outMsgs, err := worker.HandleOTATriggered(logger, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)

		var comp worker.EventEnvelope[worker.OTACompletedPayload]
		err = json.Unmarshal(outMsgs[0].Payload, &comp)
		require.NoError(t, err)
		assert.Equal(t, "job-1", comp.Payload.JobID)
		assert.Equal(t, "completed", comp.Payload.Status)
	})

	t.Run("HandleExecutiveReportTrigger", func(t *testing.T) {
		mockBLL := &mockInternalBLL{}
		msg := message.NewMessage("msg-rep", []byte(`{"period":"2026-10","generated_by":"tester"}`))
		outMsgs, err := worker.HandleExecutiveReportTrigger(context.Background(), logger, mockBLL, nil, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)

		var rep worker.EventEnvelope[worker.ExecutiveReportPayload]
		err = json.Unmarshal(outMsgs[0].Payload, &rep)
		require.NoError(t, err)
		assert.Equal(t, "rep-test-99", rep.Payload.ReportID)
		assert.Equal(t, "2026-10", rep.Payload.Period)
	})

	t.Run("HandleExecutiveReportTrigger_NilBLL", func(t *testing.T) {
		msg := message.NewMessage("msg-rep-nil", []byte(`{"period":"2026-10"}`))
		_, err := worker.HandleExecutiveReportTrigger(context.Background(), logger, nil, nil, msg)
		assert.Error(t, err)
	})

	t.Run("HandleIncidentAlert", func(t *testing.T) {
		var bffHit, pebbleHit bool
		srvBFF := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bffHit = true
			w.WriteHeader(http.StatusOK)
		}))
		defer srvBFF.Close()

		srvPebble := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pebbleHit = true
			w.WriteHeader(http.StatusCreated)
		}))
		defer srvPebble.Close()

		mockBLL := &mockInternalBLL{}
		inc := worker.EventEnvelope[worker.IncidentCreatedEventPayload]{
			EventID: "evt-inc-1",
			Payload: worker.IncidentCreatedEventPayload{
				IncidentID:  "inc-001",
				Title:       "Test incident",
				Category:    "facilities",
				Severity:    "high",
				RoomZone:    "Zone A",
				Description: "Water leak",
				ReportedAt:  time.Now().UTC(),
			},
		}
		data, _ := json.Marshal(inc)
		msg := message.NewMessage("msg-inc", data)

		err := worker.HandleIncidentAlert(context.Background(), logger, mockBLL, srvBFF.URL, srvPebble.URL, http.DefaultClient, msg)
		require.NoError(t, err)
		assert.True(t, bffHit)
		assert.True(t, pebbleHit)
	})
}

func TestBatchJobs(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermill.NopLogger{})

	t.Run("RunACHRecurringBatch_NilClient", func(t *testing.T) {
		err := worker.RunACHRecurringBatch(context.Background(), nil, pubSub, logger)
		assert.Error(t, err)
	})

	t.Run("RunACHRecurringBatch_Success", func(t *testing.T) {
		mockBLL := &mockCustomerBLL{
			claimFunc: func(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error) {
				return &customerbllv1.ClaimDueRecurringDepositsResponse{
					DueDeposits: []*customerbllv1.RecurringDepositSummary{
						{
							DepositID:   "dep-1",
							CustomerID:  "cust-1",
							PortfolioID: "port-1",
							AmountUSD:   200.0,
							BankAccount: "act-1",
						},
					},
				}, nil
			},
		}
		err := worker.RunACHRecurringBatch(context.Background(), mockBLL, pubSub, logger)
		require.NoError(t, err)
	})

	t.Run("RunEOMStatementBatch_NilClient", func(t *testing.T) {
		err := worker.RunEOMStatementBatch(context.Background(), nil, pubSub, logger, time.Now().UTC())
		assert.Error(t, err)
	})

	t.Run("RunEOMStatementBatch_Success", func(t *testing.T) {
		mockBLL := &mockCustomerBLL{
			batchStmtFunc: func(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error) {
				return &customerbllv1.GenerateBatchStatementsResponse{
					Period:          in.Period,
					TotalPortfolios: 5,
					Status:          "completed",
				}, nil
			},
		}
		err := worker.RunEOMStatementBatch(context.Background(), mockBLL, pubSub, logger, time.Now().UTC())
		require.NoError(t, err)
	})
}

func TestSchedulers(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermillLogger)

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = router.Run(ctx)
	}()

	mockCustomerBLL := &mockCustomerBLL{}
	mockInternalBLL := &mockInternalBLL{}

	worker.StartCatSpottedPublisher(ctx, router, pubSub, mockInternalBLL)
	worker.StartExecutiveReportingWorker(ctx, router, pubSub, mockInternalBLL, logger)
	worker.StartRecurringACHProcessor(ctx, router, pubSub, mockCustomerBLL, logger)
	worker.StartEOMStatementBatchGenerator(ctx, router, pubSub, mockCustomerBLL, logger)

	select {
	case <-router.Running():
	case <-time.After(2 * time.Second):
		t.Fatal("router failed to start")
	}

	time.Sleep(100 * time.Millisecond)
	_ = router.Close()
}
