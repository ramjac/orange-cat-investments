package customerbff

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type resilienceStubCustomerBLL struct {
	customerbllv1.CustomerBLLServiceClient
	getPortfolioFn       func(ctx context.Context, in *customerbllv1.GetPortfolioRequest) (*customerbllv1.GetPortfolioResponse, error)
	listDepositsFn       func(ctx context.Context, in *customerbllv1.ListDepositsRequest) (*customerbllv1.ListDepositsResponse, error)
	initiateACHDepositFn func(ctx context.Context, in *customerbllv1.InitiateACHDepositRequest) (*customerbllv1.InitiateACHDepositResponse, error)
}

func (s *resilienceStubCustomerBLL) GetPortfolio(ctx context.Context, in *customerbllv1.GetPortfolioRequest) (*customerbllv1.GetPortfolioResponse, error) {
	return s.getPortfolioFn(ctx, in)
}

func (s *resilienceStubCustomerBLL) ListDeposits(ctx context.Context, in *customerbllv1.ListDepositsRequest) (*customerbllv1.ListDepositsResponse, error) {
	return s.listDepositsFn(ctx, in)
}

func (s *resilienceStubCustomerBLL) InitiateACHDeposit(ctx context.Context, in *customerbllv1.InitiateACHDepositRequest) (*customerbllv1.InitiateACHDepositResponse, error) {
	return s.initiateACHDepositFn(ctx, in)
}

func TestCustomerBFF_ErrorLoggingSanitizationAndStatusMapping(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	var capturedDepositReq *customerbllv1.InitiateACHDepositRequest
	var portfolioCalls int

	mockBLL := &resilienceStubCustomerBLL{
		getPortfolioFn: func(ctx context.Context, in *customerbllv1.GetPortfolioRequest) (*customerbllv1.GetPortfolioResponse, error) {
			portfolioCalls++
			return nil, status.Error(codes.Unavailable, `connection error: desc = "transport: Error while dialing: dial tcp 10.152.183.181:8085: connect: connection refused"`)
		},
		listDepositsFn: func(ctx context.Context, in *customerbllv1.ListDepositsRequest) (*customerbllv1.ListDepositsResponse, error) {
			return nil, status.Error(codes.Unknown, "failed to connect to `user=oci_admin database=oci_dev`: dial tcp 10.152.183.10:5432")
		},
		initiateACHDepositFn: func(ctx context.Context, in *customerbllv1.InitiateACHDepositRequest) (*customerbllv1.InitiateACHDepositResponse, error) {
			capturedDepositReq = in
			return &customerbllv1.InitiateACHDepositResponse{
				TransferID:  "ach-123",
				CustomerID:  in.CustomerID,
				PortfolioID: in.PortfolioID,
				AmountUSD:   in.AmountUSD,
				Frequency:   in.Frequency,
				Status:      "pending_settlement",
			}, nil
		},
	}

	bff := NewCustomerBFF(mockBLL, logger)
	bff.SetCache(nil)
	store := sessionvalkey.NewMemoryCustomerSessionStore()
	sess, err := store.Create("cust-longterm-arthur", "Arthur Pendelton", "arthur@oci.local")
	require.NoError(t, err)
	bff.SetSessionStore(store)
	router := bff.Routes()

	t.Run("GetPortfolio maps BLL Unavailable to 503, logs slog.Error, and strips internal IP", func(t *testing.T) {
		logBuf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
		req.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		body := rec.Body.String()
		assert.NotContains(t, body, "10.152.183.181")
		assert.NotContains(t, body, "dial tcp")
		assert.Contains(t, body, "customer service temporarily unavailable")
		assert.Contains(t, logBuf.String(), "customer-bll gRPC call failed")
		assert.Contains(t, logBuf.String(), "GetPortfolio")
	})

	t.Run("ListDeposits sanitizes DB user= and database= strings from HTTP response", func(t *testing.T) {
		logBuf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/deposits", nil)
		req.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		body := rec.Body.String()
		assert.NotContains(t, body, "user=oci_admin")
		assert.NotContains(t, body, "database=oci_dev")
		assert.NotContains(t, body, "10.152.183.10")
		assert.Contains(t, logBuf.String(), "ListDeposits")
	})

	t.Run("handleLogin skips GetPortfolio when name is already provided", func(t *testing.T) {
		portfolioCalls = 0
		payload := `{"customer_id":"cust-active-chloe","name":"Chloe Spark","email":"chloe@oci.local"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		bff.handleLogin(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 0, portfolioCalls, "handleLogin must skip GetPortfolio when name is already set")
	})

	t.Run("handleCreateDeposit forwards BankAccount and AccountID to BLL", func(t *testing.T) {
		capturedDepositReq = nil
		payload, _ := json.Marshal(map[string]any{
			"portfolio_id": "port-1",
			"amount_usd":   250.0,
			"frequency":    "monthly",
			"bank_account": "Chase Checking (****4821)",
			"account_id":   "acc_arthur_chase_chk",
			"day_of_month": 15,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", bytes.NewReader(payload))
		req.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf"})
		req.Header.Set("X-CSRF-Token", "test-csrf")
		req.Header.Set("Idempotency-Key", "99999999-9999-4999-8999-999999999999")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		require.NotNil(t, capturedDepositReq)
		assert.Equal(t, "Chase Checking (****4821)", capturedDepositReq.BankAccount)
		assert.Equal(t, "acc_arthur_chase_chk", capturedDepositReq.AccountID)
		assert.Equal(t, int32(15), capturedDepositReq.DayOfMonth)
	})
}
