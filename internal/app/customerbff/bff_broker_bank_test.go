package customerbff

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	badReq.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	badReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	badRec := httptest.NewRecorder()
	mux.ServeHTTP(badRec, badReq)
	assert.Equal(t, http.StatusBadRequest, badRec.Code)
}

func TestCustomerBFF_PortfolioSecurity_IDOR_And_Unauthenticated(t *testing.T) {
	_, mux := setupTestBFF()

	// 1. Unauthenticated request with customer_id or portfolio_id query params must return 401 Unauthorized
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio?customer_id=cust-active-chloe&portfolio_id=port-cust-active-chloe", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauthReq)
	assert.Equal(t, http.StatusUnauthorized, unauthRec.Code)

	// 2. Authenticated request as Arthur attempting to query Chloe's portfolio via query params must ignore params and return Arthur's portfolio
	authReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio?customer_id=cust-active-chloe&portfolio_id=port-cust-active-chloe", nil)
	authReq.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	authRec := httptest.NewRecorder()
	mux.ServeHTTP(authRec, authReq)
	assert.Equal(t, http.StatusOK, authRec.Code)

	var portResp map[string]interface{}
	err := json.Unmarshal(authRec.Body.Bytes(), &portResp)
	require.NoError(t, err)
	assert.Equal(t, "cust-longterm-arthur", portResp["customer_id"])
	assert.Equal(t, "port-cust-longterm-arthur", portResp["portfolio_id"])

	// 3. Verify unauthenticated requests to other sensitive customer endpoints are rejected with 401
	protectedEndpoints := []string{
		"/api/v1/customer/deposits?customer_id=cust-active-chloe",
		"/api/v1/customer/bank-accounts?customer_id=cust-active-chloe",
		"/api/v1/customer/statements?customer_id=cust-active-chloe",
		"/api/v1/customer/api-keys?customer_id=cust-active-chloe",
		"/api/v1/customer/webhooks?customer_id=cust-active-chloe",
		"/api/v1/customer/support/tickets?customer_id=cust-active-chloe",
	}

	for _, ep := range protectedEndpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Endpoint %s must return 401 Unauthorized without valid session", ep)
	}
}
