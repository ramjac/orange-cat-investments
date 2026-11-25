package customerbll

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubWorkforceForOrders struct {
	workforcev1.WorkforceServiceClient
}

func (s *stubWorkforceForOrders) GetCareSchedule(_ context.Context, _ *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	return &workforcev1.GetCareScheduleResponse{
		CareSchedule: &workforcev1.CareSchedule{
			EmergencyMedicalHold: false,
		},
	}, nil
}

// stubCIPortfolioClient serves an active portfolio with the configured cash
// balance so pre-trade account-state verification passes and the tests can
// exercise downstream broker faults.
type stubCIPortfolioClient struct {
	coreinvestv1.CoreInvestServiceClient
	cashBalanceUSD float64
}

func (c *stubCIPortfolioClient) GetPortfolio(_ context.Context, in *coreinvestv1.GetPortfolioRequest) (*coreinvestv1.GetPortfolioResponse, error) {
	return &coreinvestv1.GetPortfolioResponse{
		Portfolio: &coreinvestv1.Portfolio{
			PortfolioID:    in.PortfolioID,
			Status:         "active",
			CashBalanceUSD: c.cashBalanceUSD,
		},
	}, nil
}

// serveQuote answers trusted market-quote requests with a fixed NVDA quote so
// market-order notional estimation succeeds while other routes keep the
// subtest's fault behavior.
func serveQuote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"symbol":"NVDA","quote":{"bp":99.5,"bs":100,"ap":100.0,"as":100}}`))
}

func assertNoVendorNames(t *testing.T, msg string) {
	t.Helper()
	lower := strings.ToLower(msg)
	assert.NotContains(t, lower, "alpaca", "error message must not leak broker vendor name")
	assert.NotContains(t, lower, "plaid", "error message must not leak bank vendor name")
}

func TestValidateAndExecuteOrder_UpstreamFaultMapping(t *testing.T) {
	t.Run("broker 503 outage maps to Unavailable without vendor name", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v2/stocks/") {
				serveQuote(w, r)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "mock broker service outage injected"})
		}))
		defer ts.Close()

		srv := NewServer(Config{
			DefaultFelineID: "feline-1",
			BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: ts.URL}),
		})
		srv.SetWorkforceClient(&stubWorkforceForOrders{})
		srv.SetCoreInvestClient(&stubCIPortfolioClient{cashBalanceUSD: 1_000_000_000})

		_, err := srv.ValidateAndExecuteOrder(context.Background(), &customerbllv1.ValidateAndExecuteOrderRequest{
			PortfolioID: "port-1",
			Symbol:      "NVDA",
			Side:        "buy",
			Quantity:    1,
			OrderType:   "market",
		})
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
		assertNoVendorNames(t, err.Error())
	})

	t.Run("broker 412 reject_orders maps to FailedPrecondition without vendor name", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v2/stocks/") {
				serveQuote(w, r)
				return
			}
			w.WriteHeader(http.StatusPreconditionFailed)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "order rejected by injected fault"})
		}))
		defer ts.Close()

		srv := NewServer(Config{
			DefaultFelineID: "feline-1",
			BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: ts.URL}),
		})
		srv.SetWorkforceClient(&stubWorkforceForOrders{})
		srv.SetCoreInvestClient(&stubCIPortfolioClient{cashBalanceUSD: 1_000_000_000})

		_, err := srv.ValidateAndExecuteOrder(context.Background(), &customerbllv1.ValidateAndExecuteOrderRequest{
			PortfolioID: "port-1",
			Symbol:      "NVDA",
			Side:        "buy",
			Quantity:    1,
			OrderType:   "market",
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, err.Error(), "order rejected")
		assertNoVendorNames(t, err.Error())
	})

	t.Run("broker 422 halted symbol maps to FailedPrecondition without vendor name", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v2/stocks/") {
				serveQuote(w, r)
				return
			}
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "trading halted for symbol by injected fault"})
		}))
		defer ts.Close()

		srv := NewServer(Config{
			DefaultFelineID: "feline-1",
			BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: ts.URL}),
		})
		srv.SetWorkforceClient(&stubWorkforceForOrders{})
		srv.SetCoreInvestClient(&stubCIPortfolioClient{cashBalanceUSD: 1_000_000_000})

		_, err := srv.ValidateAndExecuteOrder(context.Background(), &customerbllv1.ValidateAndExecuteOrderRequest{
			PortfolioID: "port-1",
			Symbol:      "NVDA",
			Side:        "buy",
			Quantity:    1,
			OrderType:   "market",
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, err.Error(), "trading halted")
		assertNoVendorNames(t, err.Error())
	})

	t.Run("broker latency exceeding bounded deadline maps to DeadlineExceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v2/stocks/") {
				// Quotes stay fast so the flow reaches the slow order route.
				serveQuote(w, r)
				return
			}
			select {
			case <-time.After(500 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
			}
		}))
		defer ts.Close()

		srv := NewServer(Config{
			DefaultFelineID: "feline-1",
			BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: ts.URL}),
			UpstreamTimeout: 50 * time.Millisecond,
		})
		srv.SetWorkforceClient(&stubWorkforceForOrders{})
		srv.SetCoreInvestClient(&stubCIPortfolioClient{cashBalanceUSD: 1_000_000_000})

		start := time.Now()
		_, err := srv.ValidateAndExecuteOrder(context.Background(), &customerbllv1.ValidateAndExecuteOrderRequest{
			PortfolioID: "port-1",
			Symbol:      "NVDA",
			Side:        "buy",
			Quantity:    1,
			OrderType:   "market",
		})
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
		assert.Less(t, elapsed, 400*time.Millisecond, "bounded deadline must abort before slow server completes")
		assertNoVendorNames(t, err.Error())
	})
}

func TestGetTickerSnapshot_UpstreamFaultMapping(t *testing.T) {
	t.Run("broker 503 maps to Unavailable", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "outage"})
		}))
		defer ts.Close()

		srv := NewServer(Config{
			BrokerClient: alpaca.NewClient(alpaca.Config{BaseURL: ts.URL}),
		})

		_, err := srv.GetTickerSnapshot(context.Background(), &customerbllv1.GetTickerSnapshotRequest{
			Symbols: []string{"NVDA"},
		})
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
		assertNoVendorNames(t, err.Error())
	})

	t.Run("broker latency timeout maps to DeadlineExceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(500 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
			}
		}))
		defer ts.Close()

		srv := NewServer(Config{
			BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: ts.URL}),
			UpstreamTimeout: 50 * time.Millisecond,
		})

		_, err := srv.GetTickerSnapshot(context.Background(), &customerbllv1.GetTickerSnapshotRequest{
			Symbols: []string{"NVDA"},
		})
		require.Error(t, err)
		assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
		assertNoVendorNames(t, err.Error())
	})
}

func TestSettleACHTransfer_UpstreamFaultMapping(t *testing.T) {
	t.Run("bank 503 outage maps to Unavailable without vendor name", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "mock bank service outage injected"})
		}))
		defer ts.Close()

		srv := NewServer(Config{
			BankClient: plaid.NewClient(plaid.Config{BaseURL: ts.URL}),
		})

		_, err := srv.SettleACHTransfer(context.Background(), &customerbllv1.SettleACHTransferRequest{
			TransferID:   "ach-001",
			BankAccount:  "acc-1",
			AmountUSD:    250,
			TransferType: "deposit",
		})
		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
		assertNoVendorNames(t, err.Error())
	})

	t.Run("bank 412 reject_transfers maps to FailedPrecondition without vendor name", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPreconditionFailed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "transfer rejected by injected fault"})
		}))
		defer ts.Close()

		srv := NewServer(Config{
			BankClient: plaid.NewClient(plaid.Config{BaseURL: ts.URL}),
		})

		_, err := srv.SettleACHTransfer(context.Background(), &customerbllv1.SettleACHTransferRequest{
			TransferID:   "ach-002",
			BankAccount:  "acc-1",
			AmountUSD:    250,
			TransferType: "deposit",
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, err.Error(), "transfer rejected")
		assertNoVendorNames(t, err.Error())
	})

	t.Run("bank latency exceeding bounded deadline maps to DeadlineExceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(500 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
			}
		}))
		defer ts.Close()

		srv := NewServer(Config{
			BankClient:      plaid.NewClient(plaid.Config{BaseURL: ts.URL}),
			UpstreamTimeout: 50 * time.Millisecond,
		})

		start := time.Now()
		_, err := srv.SettleACHTransfer(context.Background(), &customerbllv1.SettleACHTransferRequest{
			TransferID:   "ach-003",
			BankAccount:  "acc-1",
			AmountUSD:    250,
			TransferType: "deposit",
		})
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
		assert.Less(t, elapsed, 400*time.Millisecond)
		assertNoVendorNames(t, err.Error())
	})
}

func TestCaptureDailySnapshots_UpstreamTimeoutMapsToDeadlineExceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(500 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer ts.Close()

	srv := NewServer(Config{
		BrokerClient:    alpaca.NewClient(alpaca.Config{BaseURL: ts.URL}),
		UpstreamTimeout: 50 * time.Millisecond,
	})
	srv.SetCoreInvestClient(&mockCoreInvestForHistory{
		listPortfoliosFn: func(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
			return &coreinvestv1.ListPortfoliosResponse{
				Portfolios: []*coreinvestv1.Portfolio{
					{PortfolioID: "port-slow", Status: "active", TotalBalanceUSD: 1000, CashBalanceUSD: 500},
				},
			}, nil
		},
	})

	_, err := srv.CaptureDailySnapshots(context.Background(), &customerbllv1.CaptureDailySnapshotsRequest{
		SnapshotDate: "2026-11-18",
	})
	require.Error(t, err)
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
	assertNoVendorNames(t, err.Error())
}

func TestMapUpstreamError_SanitizesVendorNames(t *testing.T) {
	err := mapUpstreamError("broker", &alpaca.APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Message:    "Alpaca rejected order because Plaid funding is pending",
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assertNoVendorNames(t, err.Error())
	assert.Contains(t, err.Error(), "broker rejected request")
}

func TestNewServer_UpstreamTimeoutEnv(t *testing.T) {
	t.Setenv("UPSTREAM_TIMEOUT", "2500ms")
	srv := NewServer(Config{})
	assert.Equal(t, 2500*time.Millisecond, srv.upstreamDeadline())

	srv.SetUpstreamTimeout(75 * time.Millisecond)
	assert.Equal(t, 75*time.Millisecond, srv.upstreamDeadline())
}
