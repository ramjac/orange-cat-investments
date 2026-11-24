package worker_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/internal/app/worker"
	"github.com/orange-cat-investments/oci/pkg/cosignverify"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCustomerBLL struct {
	customerbllv1.CustomerBLLServiceClient
	settleFunc    func(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error)
	claimFunc     func(ctx context.Context, in *customerbllv1.ClaimDueRecurringDepositsRequest) (*customerbllv1.ClaimDueRecurringDepositsResponse, error)
	batchStmtFunc        func(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error)
	completeBacktestFunc func(ctx context.Context, in *customerbllv1.CompleteBacktestRunRequest) (*customerbllv1.CompleteBacktestRunResponse, error)
	captureSnapshotsFunc func(ctx context.Context, in *customerbllv1.CaptureDailySnapshotsRequest) (*customerbllv1.CaptureDailySnapshotsResponse, error)
}

func (m *mockCustomerBLL) CaptureDailySnapshots(ctx context.Context, in *customerbllv1.CaptureDailySnapshotsRequest) (*customerbllv1.CaptureDailySnapshotsResponse, error) {
	if m.captureSnapshotsFunc != nil {
		return m.captureSnapshotsFunc(ctx, in)
	}
	return &customerbllv1.CaptureDailySnapshotsResponse{
		SnapshotDate:      in.SnapshotDate,
		SnapshotsRecorded: 0,
	}, nil
}

func (m *mockCustomerBLL) CompleteBacktestRun(ctx context.Context, in *customerbllv1.CompleteBacktestRunRequest) (*customerbllv1.CompleteBacktestRunResponse, error) {
	if m.completeBacktestFunc != nil {
		return m.completeBacktestFunc(ctx, in)
	}
	return &customerbllv1.CompleteBacktestRunResponse{}, nil
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
	updateOTAFunc      func(ctx context.Context, in *facilitiesv1.UpdateOTAJobStatusRequest) (*facilitiesv1.UpdateOTAJobStatusResponse, error)
}

func (m *mockInternalBLL) UpdateOTAJobStatus(ctx context.Context, in *facilitiesv1.UpdateOTAJobStatusRequest) (*facilitiesv1.UpdateOTAJobStatusResponse, error) {
	if m.updateOTAFunc != nil {
		return m.updateOTAFunc(ctx, in)
	}
	return &facilitiesv1.UpdateOTAJobStatusResponse{Job: &facilitiesv1.DeviceOTAJob{JobID: in.JobID, Status: in.Status}}, nil
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
		mockBLL := &mockCustomerBLL{}
		msg := message.NewMessage("test-uuid", []byte(`{"backtest_id":"bt-1","strategy_id":"alpha"}`))
		outMsgs, err := worker.HandleBacktestRun(context.Background(), logger, mockBLL, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)
		assert.Contains(t, string(outMsgs[0].Payload), "events.investment.backtest_completed.v1")
	})

	t.Run("HandleBacktestRun_NilClient", func(t *testing.T) {
		msg := message.NewMessage("test-uuid", []byte(`{"backtest_id":"bt-1"}`))
		_, err := worker.HandleBacktestRun(context.Background(), logger, nil, msg)
		assert.Error(t, err)
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

		_, err := worker.HandleACHTransferRequested(context.Background(), logger, nil, msg)
		assert.Error(t, err)
	})
}

func TestConsumersOpsWorkforce(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	t.Run("HandleOTATriggered", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		privDER, err := x509.MarshalPKCS8PrivateKey(priv)
		require.NoError(t, err)
		pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		require.NoError(t, err)
		privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
		pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

		pubKeyPath := filepath.Join(t.TempDir(), "cosign.pub")
		require.NoError(t, os.WriteFile(pubKeyPath, pubPEM, 0o600))
		t.Setenv(cosignverify.DefaultPublicKeyEnv, pubKeyPath)

		firmwareBytes := []byte("OCI-SIGNED-EDGE-FIRMWARE-v1.2.0")
		sum := sha256.Sum256(firmwareBytes)
		checksum := hex.EncodeToString(sum[:])
		sig, err := cosignverify.SignBlob(privPEM, firmwareBytes)
		require.NoError(t, err)

		pkgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/1.2.0.bin":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(firmwareBytes)
			case "/tampered.bin":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("TAMPERED-FIRMWARE-BYTES"))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer pkgServer.Close()

		var lastStatus string
		mockBLL := &mockInternalBLL{
			updateOTAFunc: func(ctx context.Context, in *facilitiesv1.UpdateOTAJobStatusRequest) (*facilitiesv1.UpdateOTAJobStatusResponse, error) {
				lastStatus = in.Status
				return &facilitiesv1.UpdateOTAJobStatusResponse{
					Job: &facilitiesv1.DeviceOTAJob{JobID: in.JobID, Status: in.Status, ErrorMessage: in.ErrorMessage},
				}, nil
			},
		}
		otaReq := worker.EventEnvelope[worker.OTATriggeredPayload]{
			EventID: "evt-ota-1",
			Payload: worker.OTATriggeredPayload{
				JobID:           "job-1",
				AssetID:         "asset-1",
				ReleaseID:       "rel-1",
				DeviceType:      "edge_camera",
				Version:         "1.2.0",
				FileURL:         pkgServer.URL + "/1.2.0.bin",
				Checksum:        checksum,
				CosignSignature: sig,
			},
		}
		data, _ := json.Marshal(otaReq)
		msg := message.NewMessage("msg-ota", data)

		outMsgs, err := worker.HandleOTATriggered(context.Background(), logger, mockBLL, msg)
		require.NoError(t, err)
		require.Len(t, outMsgs, 1)
		assert.Equal(t, "completed", lastStatus)

		var comp worker.EventEnvelope[worker.OTACompletedPayload]
		err = json.Unmarshal(outMsgs[0].Payload, &comp)
		require.NoError(t, err)
		assert.Equal(t, "job-1", comp.Payload.JobID)
		assert.Equal(t, "completed", comp.Payload.Status)

		// Reject tampered firmware payload -> marks job failed and errors for .dlq routing.
		lastStatus = ""
		tamperedReq := otaReq
		tamperedReq.Payload.FileURL = pkgServer.URL + "/tampered.bin"
		tamperedData, _ := json.Marshal(tamperedReq)
		_, err = worker.HandleOTATriggered(context.Background(), logger, mockBLL, message.NewMessage("msg-ota-tamper", tamperedData))
		assert.Error(t, err)
		assert.Equal(t, "failed", lastStatus)

		// Reject unsigned firmware payload -> marks job failed and errors for .dlq routing.
		lastStatus = ""
		unsignedReq := otaReq
		unsignedReq.Payload.CosignSignature = ""
		unsignedData, _ := json.Marshal(unsignedReq)
		_, err = worker.HandleOTATriggered(context.Background(), logger, mockBLL, message.NewMessage("msg-ota-unsigned", unsignedData))
		assert.Error(t, err)
		assert.Equal(t, "failed", lastStatus)

		// Reject forged/invalid Cosign signature -> marks job failed and errors for .dlq routing.
		lastStatus = ""
		badSigReq := otaReq
		badSigReq.Payload.CosignSignature = "aW52YWxpZC1zaWduYXR1cmU="
		badSigData, _ := json.Marshal(badSigReq)
		_, err = worker.HandleOTATriggered(context.Background(), logger, mockBLL, message.NewMessage("msg-ota-badsig", badSigData))
		assert.Error(t, err)
		assert.Equal(t, "failed", lastStatus)
	})

	t.Run("HandleOTATriggered_NilClient", func(t *testing.T) {
		msg := message.NewMessage("msg-ota-nil", []byte(`{}`))
		_, err := worker.HandleOTATriggered(context.Background(), logger, nil, msg)
		assert.Error(t, err)
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

	t.Run("RunDailyNAVSnapshotBatch_NilClient", func(t *testing.T) {
		err := worker.RunDailyNAVSnapshotBatch(context.Background(), nil, logger, time.Now().UTC())
		assert.Error(t, err)
	})

	t.Run("RunDailyNAVSnapshotBatch_Success", func(t *testing.T) {
		var capturedDate string
		mockBLL := &mockCustomerBLL{
			captureSnapshotsFunc: func(ctx context.Context, in *customerbllv1.CaptureDailySnapshotsRequest) (*customerbllv1.CaptureDailySnapshotsResponse, error) {
				capturedDate = in.SnapshotDate
				return &customerbllv1.CaptureDailySnapshotsResponse{
					SnapshotDate:      in.SnapshotDate,
					SnapshotsRecorded: 2,
				}, nil
			},
		}
		runAt := time.Date(2026, time.November, 17, 21, 0, 0, 0, time.UTC)
		err := worker.RunDailyNAVSnapshotBatch(context.Background(), mockBLL, logger, runAt)
		require.NoError(t, err)
		assert.Equal(t, "2026-11-17", capturedDate)
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
