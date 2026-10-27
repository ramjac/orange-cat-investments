package customerbff

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestBFF() (*CustomerBFF, *http.ServeMux) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	bff := NewCustomerBFF(newMockCustomerBLLClient(), logger)
	sessStore := sessionvalkey.NewMemoryCustomerSessionStore()
	sessStore.SetSession(&CustomerSession{
		SessionID:  "sess-arthur-token",
		CustomerID: "cust-longterm-arthur",
		Email:      "arthur@oci.local",
		Name:       "Arthur Pendelton",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})
	sessStore.SetSession(&CustomerSession{
		SessionID:  "sess-chloe-token",
		CustomerID: "cust-active-chloe",
		Email:      "chloe@oci.local",
		Name:       "Chloe Spark",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})
	bff.SetSessionStore(sessStore)
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

func TestCustomerBFF_SessionTokenViaURLParamsRejected(t *testing.T) {
	_, mux := setupTestBFF()

	// GET protected route with session token in query parameter - MUST BE REJECTED
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio?session_token=sess-arthur-token", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized: valid customer session required")

	// GET protected route with session token via Authorization Bearer header - MUST BE ACCEPTED
	req = httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	req.Header.Set("Authorization", "Bearer sess-arthur-token")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// GET protected route with session token via Cookie - MUST BE ACCEPTED
	req = httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
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
	assert.Equal(t, "port-cust-longterm-arthur", orderResp["portfolio_id"])
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
	assert.Equal(t, "port-cust-active-chloe", orderResp["portfolio_id"])
}

func TestCustomerBFF_LoginAndSessionFlow(t *testing.T) {
	_, mux := setupTestBFF()

	// 1. Login as Chloe
	loginBody := bytes.NewBufferString(`{"customer_id":"cust-active-chloe","name":"Chloe Spark","email":"chloe@oci.local"}`)
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
