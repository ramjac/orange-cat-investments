package customerbff

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCustomerBFF_PortfolioAndTickerRoutes(t *testing.T) {
	_, mux := setupTestBFF()

	// 1. GET /api/v1/customer/portfolio
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var portResp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &portResp)
	require.NoError(t, err)
	assert.Greater(t, portResp["total_balance_usd"].(float64), 0.0)
	assert.Greater(t, portResp["cash_balance_usd"].(float64), 0.0)

	// 1.5 GET /api/v1/customer/portfolio/history
	for _, period := range []string{"1M", "3M", "6M", "1Y", "YTD"} {
		reqHist := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio/history?period="+period, nil)
		reqHist.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
		recHist := httptest.NewRecorder()
		mux.ServeHTTP(recHist, reqHist)
		assert.Equal(t, http.StatusOK, recHist.Code)
	}

	// default period (no query parameter) must be accepted
	reqHistDefault := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio/history", nil)
	reqHistDefault.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	recHistDefault := httptest.NewRecorder()
	mux.ServeHTTP(recHistDefault, reqHistDefault)
	assert.Equal(t, http.StatusOK, recHistDefault.Code)

	reqHistInvalid := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio/history?period=invalid", nil)
	reqHistInvalid.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	recHistInvalid := httptest.NewRecorder()
	mux.ServeHTTP(recHistInvalid, reqHistInvalid)
	assert.Equal(t, http.StatusBadRequest, recHistInvalid.Code)

	// 2. GET /api/v1/customer/ticker
	tReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/ticker", nil)
	tReq.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	tRec := httptest.NewRecorder()
	mux.ServeHTTP(tRec, tReq)
	assert.Equal(t, http.StatusOK, tRec.Code)

	var tickers []map[string]interface{}
	err = json.Unmarshal(tRec.Body.Bytes(), &tickers)
	require.NoError(t, err)
	assert.NotEmpty(t, tickers)
}

func TestCustomerBFF_BankAccountsAndDeposits(t *testing.T) {
	_, mux := setupTestBFF()

	// 1. GET /api/v1/customer/bank-accounts
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/bank-accounts", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var bankResp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &bankResp)
	require.NoError(t, err)
	assert.NotEmpty(t, bankResp["accounts"])

	// 2. Successful POST /api/v1/customer/deposits
	csrfVal := "test-bank-csrf"
	depBody := bytes.NewBufferString(`{
		"amount_usd": 500.0,
		"frequency": "monthly",
		"bank_account": "Chase Checking (****4821)",
		"account_id": "acc_arthur_chase_chk"
	}`)
	depReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", depBody)
	depReq.Header.Set("Content-Type", "application/json")
	depReq.Header.Set("X-CSRF-Token", csrfVal)
	depReq.Header.Set("Idempotency-Key", "33333333-3333-4333-8333-333333333333")
	depReq.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	depReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	depRec := httptest.NewRecorder()
	mux.ServeHTTP(depRec, depReq)
	assert.Equal(t, http.StatusCreated, depRec.Code)

	// 3. Invalid amount_usd <= 0 returns 400 Bad Request
	badBody := bytes.NewBufferString(`{
		"amount_usd": -100.0,
		"frequency": "monthly"
	}`)
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", badBody)
	badReq.Header.Set("Content-Type", "application/json")
	badReq.Header.Set("X-CSRF-Token", csrfVal)
	badReq.Header.Set("Idempotency-Key", "44444444-4444-4444-8444-444444444444")
	badReq.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	badReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	badRec := httptest.NewRecorder()
	mux.ServeHTTP(badRec, badReq)
	assert.Equal(t, http.StatusBadRequest, badRec.Code)
}

type faultInjectingBLLClient struct {
	*mockCustomerBLLClient
	orderErr  error
	tickerErr error
}

func (f *faultInjectingBLLClient) ValidateAndExecuteOrder(ctx context.Context, in *customerbllv1.ValidateAndExecuteOrderRequest) (*customerbllv1.ValidateAndExecuteOrderResponse, error) {
	if f.orderErr != nil {
		return nil, f.orderErr
	}
	return f.mockCustomerBLLClient.ValidateAndExecuteOrder(ctx, in)
}

func (f *faultInjectingBLLClient) GetTickerSnapshot(ctx context.Context, in *customerbllv1.GetTickerSnapshotRequest) (*customerbllv1.GetTickerSnapshotResponse, error) {
	if f.tickerErr != nil {
		return nil, f.tickerErr
	}
	return f.mockCustomerBLLClient.GetTickerSnapshot(ctx, in)
}

func TestCustomerBFF_OrderAndTickerFaultStatusMapping(t *testing.T) {
	cases := []struct {
		name           string
		grpcErr        error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "Unavailable maps to 503",
			grpcErr:        status.Error(codes.Unavailable, "broker service unavailable"),
			expectedStatus: http.StatusServiceUnavailable,
			expectedMsg:    "broker service unavailable",
		},
		{
			name:           "DeadlineExceeded maps to 504",
			grpcErr:        status.Error(codes.DeadlineExceeded, "broker request timed out"),
			expectedStatus: http.StatusGatewayTimeout,
			expectedMsg:    "broker request timed out",
		},
		{
			name:           "FailedPrecondition maps to 422",
			grpcErr:        status.Error(codes.FailedPrecondition, "broker rejected request: trading halted"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedMsg:    "broker rejected request: trading halted",
		},
		{
			name:           "InvalidArgument maps to 400",
			grpcErr:        status.Error(codes.InvalidArgument, "invalid order quantity"),
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid order quantity",
		},
	}

	for _, tc := range cases {
		t.Run("order "+tc.name, func(t *testing.T) {
			bff, mux := setupTestBFF()
			bff.SetBLLClient(&faultInjectingBLLClient{
				mockCustomerBLLClient: newMockCustomerBLLClient(),
				orderErr:              tc.grpcErr,
			})

			csrfVal := "fault-csrf-token"
			body := bytes.NewBufferString(`{"symbol":"NVDA","side":"buy","quantity":1,"order_type":"market"}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", csrfVal)
			req.Header.Set("Idempotency-Key", "55555555-5555-4555-8555-555555555555")
			req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			var errResp map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
			assert.Equal(t, tc.expectedMsg, errResp["error"])
		})

		t.Run("ticker "+tc.name, func(t *testing.T) {
			bff, mux := setupTestBFF()
			bff.SetCache(nil)
			bff.SetBLLClient(&faultInjectingBLLClient{
				mockCustomerBLLClient: newMockCustomerBLLClient(),
				tickerErr:             tc.grpcErr,
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/ticker", nil)
			req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			var errResp map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
			assert.Equal(t, tc.expectedMsg, errResp["error"])
		})
	}
}
