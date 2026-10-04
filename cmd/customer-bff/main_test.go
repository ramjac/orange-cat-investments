package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionStore_NoHardcodedTokensByDefault(t *testing.T) {
	store := NewSessionStore()
	_, okArthur := store.Get("sess-arthur-token")
	assert.False(t, okArthur, "sess-arthur-token should not exist by default")
	_, okChloe := store.Get("sess-chloe-token")
	assert.False(t, okChloe, "sess-chloe-token should not exist by default")

	store.SeedDevSessions()
	_, okArthurAfter := store.Get("sess-arthur-token")
	assert.True(t, okArthurAfter, "sess-arthur-token should exist after SeedDevSessions")
	_, okChloeAfter := store.Get("sess-chloe-token")
	assert.True(t, okChloeAfter, "sess-chloe-token should exist after SeedDevSessions")
}

func setupTestBFF() (*CustomerBFF, *http.ServeMux) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	ciRepo := core_invest_repo.NewMockRepository()
	ciSvc := core_invest_svc.NewService(ciRepo)
	wfRepo := workforce_repo.NewRepository(nil)
	wfSvc := workforce_svc.NewService(wfRepo)

	bff := NewCustomerBFF(ciSvc, wfSvc, logger)
	bff.sessions.SeedDevSessions()
	mux := http.NewServeMux()
	bff.RegisterRoutes(mux)
	return bff, mux
}

func TestCustomerBFF_UnauthenticatedProtectedRoutes(t *testing.T) {
	_, mux := setupTestBFF()

	// GET protected route without session
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// POST protected route without session
	body := bytes.NewBufferString(`{"symbol":"AAPL","side":"buy","quantity":10}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", body)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCustomerBFF_CSRFProtection(t *testing.T) {
	_, mux := setupTestBFF()

	// 1. Authenticated session but NO CSRF header or cookie
	body := bytes.NewBufferString(`{"symbol":"NVDA","side":"buy","quantity":5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "forbidden: invalid or missing CSRF token")

	// 2. Authenticated session with mismatched CSRF token
	body = bytes.NewBufferString(`{"symbol":"NVDA","side":"buy","quantity":5}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "token-mismatch-123")
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-csrf-cookie-value"})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// 3. Authenticated session with matching double-submit CSRF cookie & header
	csrfVal := "secret-double-submit-csrf-token"
	body = bytes.NewBufferString(`{"symbol":"NVDA","side":"buy","quantity":5}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfVal)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var orderResp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &orderResp)
	require.NoError(t, err)
	assert.Equal(t, "NVDA", orderResp["symbol"])
	assert.Equal(t, "buy", orderResp["side"])
	// Context propagation: Arthur gets Arthur's portfolio
	assert.Equal(t, "018f3a9a-2222-7000-8000-000000000002", orderResp["portfolio_id"])
}

func TestCustomerBFF_ContextPropagation_Chloe(t *testing.T) {
	_, mux := setupTestBFF()

	csrfVal := "chloe-csrf-token"
	body := bytes.NewBufferString(`{"symbol":"TSLA","side":"buy","quantity":10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfVal)
	req.AddCookie(&http.Cookie{Name: "__Host-customer-session", Value: "sess-chloe-token"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var orderResp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &orderResp)
	require.NoError(t, err)
	// Context propagation: Chloe gets Chloe's portfolio
	assert.Equal(t, "018f3a9a-2222-7000-8000-000000000003", orderResp["portfolio_id"])
}

func TestCustomerBFF_LoginAndSessionFlow(t *testing.T) {
	_, mux := setupTestBFF()

	// 1. Login as Chloe
	loginBody := bytes.NewBufferString(`{"persona":"chloe"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/login", loginBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var loginResp struct {
		Status    string           `json:"status"`
		Session   *CustomerSession `json:"session"`
		CSRFToken string           `json:"csrf_token"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	assert.Equal(t, "authenticated", loginResp.Status)
	assert.Equal(t, "cust-active-chloe", loginResp.Session.CustomerID)
	assert.NotEmpty(t, loginResp.CSRFToken)

	// Check cookies set
	cookies := rec.Result().Cookies()
	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "customer_session" {
			sessionCookie = c
		}
		if c.Name == "csrf_token" {
			csrfCookie = c
		}
	}
	require.NotNil(t, sessionCookie)
	require.NotNil(t, csrfCookie)
	assert.True(t, sessionCookie.HttpOnly)
	assert.False(t, csrfCookie.HttpOnly) // Accessible to JS

	// 2. Access portfolio using newly issued session cookie
	req = httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var portResp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &portResp)
	require.NoError(t, err)
	assert.Equal(t, "cust-active-chloe", portResp["customer_id"])
}

func TestCustomerBFF_StateMutations_RequireCSRF(t *testing.T) {
	_, mux := setupTestBFF()

	csrfVal := "test-mutation-csrf"

	// 1. Create Deposit without CSRF
	body := bytes.NewBufferString(`{"amount_usd":250,"frequency":"monthly","day_of_month":1,"bank_account":"Test Bank"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Create Deposit with CSRF
	body = bytes.NewBufferString(`{"amount_usd":250,"frequency":"monthly","day_of_month":1,"bank_account":"Test Bank"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfVal)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// 2. Create API Key with CSRF
	body = bytes.NewBufferString(`{"name":"Test Bot","permissions":["read_streams"]}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/customer/api-keys", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfVal)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// 3. Revoke API Key without CSRF
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/customer/api-keys/key-123", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Revoke API Key with CSRF
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/customer/api-keys/key-123", nil)
	req.Header.Set("X-CSRF-Token", csrfVal)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
