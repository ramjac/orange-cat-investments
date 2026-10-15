package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomToken(t *testing.T) {
	token, err := generateRandomToken(16)
	require.NoError(t, err)
	assert.Len(t, token, 32) // 16 bytes encoded in hex is 32 chars

	token2, err := generateRandomToken(16)
	require.NoError(t, err)
	assert.NotEqual(t, token, token2, "subsequent tokens should be cryptographically unique")
}

func TestNewSessionStore_NoHardcodedTokensByDefault(t *testing.T) {
	store := NewSessionStore()
	_, okArthur := store.Get("sess-arthur-token")
	assert.False(t, okArthur, "sess-arthur-token should not exist by default")
	_, okChloe := store.Get("sess-chloe-token")
	assert.False(t, okChloe, "sess-chloe-token should not exist by default")

	store.SetSession(&CustomerSession{
		SessionID:  "sess-test-token",
		CustomerID: "cust-test",
		ExpiresAt:  time.Now().Add(time.Hour),
	})
	sess, okAfter := store.Get("sess-test-token")
	assert.True(t, okAfter)
	assert.Equal(t, "cust-test", sess.CustomerID)
}

func TestSessionStore_Create_And_GenerateRandomToken(t *testing.T) {
	store := NewSessionStore()
	sess, err := store.Create("cust-123", "Test User", "test@oci.local")
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, "cust-123", sess.CustomerID)
	assert.True(t, len(sess.SessionID) > 5)
	assert.Equal(t, "sess-", sess.SessionID[:5])

	gotSess, ok := store.Get(sess.SessionID)
	assert.True(t, ok)
	assert.Equal(t, sess, gotSess)

	tok, err := generateRandomToken(16)
	require.NoError(t, err)
	assert.Len(t, tok, 32) // 16 bytes encoded as hex = 32 chars
}

func setupTestBFF() (*CustomerBFF, *http.ServeMux) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	ciRepo := core_invest_repo.NewMockRepository()
	ciSvc := core_invest_svc.NewService(ciRepo)
	wfRepo := workforce_repo.NewRepository(nil)
	wfSvc := workforce_svc.NewService(wfRepo)

	bff := NewCustomerBFF(ciSvc, wfSvc, logger)
	bff.sessions.SetSession(&CustomerSession{
		SessionID:  "sess-arthur-token",
		CustomerID: "cust-longterm-arthur",
		Email:      "arthur@oci.local",
		Name:       "Arthur Pendelton",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})
	bff.sessions.SetSession(&CustomerSession{
		SessionID:  "sess-chloe-token",
		CustomerID: "cust-active-chloe",
		Email:      "chloe@oci.local",
		Name:       "Chloe Spark",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})
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

func TestCustomerBFF_GetESGTransparency(t *testing.T) {
	_, mux := setupTestBFF()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/transparency/welfare", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp, "overall_welfare_score")
	assert.Contains(t, resp, "veterinary_adherence_pct")
	assert.Contains(t, resp, "feline_executives")

	execs, ok := resp["feline_executives"].([]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, execs)
}

func BenchmarkGetESGTransparency(b *testing.B) {
	_, mux := setupTestBFF()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/transparency/welfare", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkGetZoomieIndex(b *testing.B) {
	_, mux := setupTestBFF()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/zoomie-index", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
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

func TestCustomerBFF_Webhooks(t *testing.T) {
	_, mux := setupTestBFF()

	csrfVal := "webhook-test-csrf"

	// 1. GET /api/v1/customer/webhooks must not expose hardcoded or static webhook secrets
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/webhooks", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var webhooks []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &webhooks)
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, "sub-018f-1111", webhooks[0]["subscription_id"])
	assert.Nil(t, webhooks[0]["secret"], "GET /webhooks must not include secret field")

	// 2. POST /api/v1/customer/webhooks creates subscription with a dynamically generated secret upon creation
	body := bytes.NewBufferString(`{"target_url":"https://example.com/callback","events":["events.trade.executed.v1"]}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/customer/webhooks", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfVal)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var created map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &created)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/callback", created["target_url"])
	secretStr, ok := created["secret"].(string)
	assert.True(t, ok)
	assert.True(t, len(secretStr) > 6)
	assert.True(t, secretStr[:6] == "whsec_")
}

func TestCustomerBFF_AlpacaIntegration(t *testing.T) {
	brokerMux := http.NewServeMux()
	arthurID := "018f3a9a-2222-7000-8000-000000000002"

	brokerMux.HandleFunc("GET /v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]alpaca.Account{
			{
				ID:             arthurID,
				CustomerName:   "Arthur Pendelton",
				Cash:           45000.0,
				PortfolioValue: 155000.0,
				Status:         "ACTIVE",
			},
		})
	})

	brokerMux.HandleFunc("GET /v1/accounts/"+arthurID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alpaca.Account{
			ID:             arthurID,
			CustomerName:   "Arthur Pendelton",
			Cash:           45000.0,
			PortfolioValue: 155000.0,
			Status:         "ACTIVE",
		})
	})

	brokerMux.HandleFunc("GET /v1/trading/accounts/"+arthurID+"/positions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]alpaca.Position{
			{
				Symbol:        "NVDA",
				Qty:           100.0,
				AvgEntryPrice: 120.0,
				CurrentPrice:  140.0,
				MarketValue:   14000.0,
			},
		})
	})

	brokerMux.HandleFunc("GET /v2/stocks/bars/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bars": map[string]alpaca.Bar{
				"AAPL": {Open: 220.0, Close: 225.5, Volume: 50000},
				"NVDA": {Open: 135.0, Close: 140.0, Volume: 80000},
			},
		})
	})

	brokerMux.HandleFunc("GET /v2/stocks/AAPL/quotes/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "AAPL",
			"quote":  alpaca.Quote{BidPrice: 225.4, AskPrice: 225.6},
		})
	})

	brokerServer := httptest.NewServer(brokerMux)
	defer brokerServer.Close()

	brokerClient := alpaca.NewClient(alpaca.Config{
		BaseURL: brokerServer.URL,
	})

	bff, mux := setupTestBFF()
	bff.SetBrokerClient(brokerClient)

	// 1. GET /api/v1/customer/portfolio with Alpaca client active
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var portResp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &portResp)
	require.NoError(t, err)
	assert.Equal(t, 155000.0, portResp["total_balance_usd"])
	assert.Equal(t, 45000.0, portResp["cash_balance_usd"])

	// 2. GET /api/v1/customer/ticker with Alpaca client active
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

func TestCustomerBFF_BankIntegration(t *testing.T) {
	bankMux := http.NewServeMux()
	bankMux.HandleFunc("GET /accounts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"accounts": []map[string]interface{}{
				{
					"account_id":        "acc_arthur_chase_chk",
					"institution_name":  "JPMorgan Chase",
					"account_number":    "****4821",
					"name":              "Chase Checking (****4821)",
					"available_balance": 48250.00,
					"current_balance":   48500.00,
				},
			},
		})
	})
	bankMux.HandleFunc("POST /transfer/authorization/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body struct {
			Amount float64 `json:"amount"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Amount > 10000 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"authorization": map[string]interface{}{
					"decision": "declined",
					"decision_rationale": map[string]string{
						"code":        "NSF",
						"description": "Insufficient funds in bank account",
					},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"authorization": map[string]interface{}{
				"decision": "approved",
			},
		})
	})
	bankMux.HandleFunc("POST /transfer/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"transfer": map[string]interface{}{
				"transfer_id": "tr-test-12345",
				"status":      "pending",
			},
		})
	})

	bankServer := httptest.NewServer(bankMux)
	defer bankServer.Close()

	bankClient := plaid.NewClient(plaid.Config{
		BaseURL: bankServer.URL,
	})

	mockRepo := core_invest_repo.NewMockRepository()
	ciSvc := core_invest_svc.NewServiceWithIntegrations(mockRepo, nil, bankClient)
	wfRepo := workforce_repo.NewRepository(nil)
	wfSvc := workforce_svc.NewService(wfRepo)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	bff := NewCustomerBFF(ciSvc, wfSvc, logger)
	bff.SetBankClient(bankClient)
	bff.sessions.SetSession(&CustomerSession{
		SessionID:  "sess-arthur-token",
		CustomerID: "cust-longterm-arthur",
		Email:      "arthur@oci.local",
		Name:       "Arthur Pendelton",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})

	mux := http.NewServeMux()
	bff.RegisterRoutes(mux)

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

	// 2. Successful POST /api/v1/customer/deposits with bank pre-authorization
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

	// 3. Declined POST /api/v1/customer/deposits on NSF
	nsfBody := bytes.NewBufferString(`{
		"amount_usd": 50000.0,
		"frequency": "monthly",
		"bank_account": "Chase Checking (****4821)",
		"account_id": "acc_arthur_chase_chk"
	}`)
	nsfReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", nsfBody)
	nsfReq.Header.Set("Content-Type", "application/json")
	nsfReq.Header.Set("X-CSRF-Token", csrfVal)
	nsfReq.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-arthur-token"})
	nsfReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	nsfRec := httptest.NewRecorder()
	mux.ServeHTTP(nsfRec, nsfReq)
	assert.Equal(t, http.StatusBadRequest, nsfRec.Code)
}

func TestCustomerBFF_DynamicCustomerRemediation(t *testing.T) {
	_, mux := setupTestBFF()
	csrfVal := "remediation-csrf-token"

	// 1. Dynamic Login with arbitrary new customer ID (not Chloe or Arthur)
	loginBody := bytes.NewBufferString(`{"customer_id":"cust-dynamo-999","name":"Diana Prince","email":"diana@oci.local"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/auth/login", loginBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var loginResp struct {
		Session *CustomerSession `json:"session"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	require.NotNil(t, loginResp.Session)
	assert.Equal(t, "cust-dynamo-999", loginResp.Session.CustomerID)
	assert.Equal(t, "Diana Prince", loginResp.Session.Name)

	sessToken := loginResp.Session.SessionID

	// 2. Dynamic portfolio fallback for Diana: Must NOT return Chloe or Arthur's holdings
	portReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/portfolio", nil)
	portReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	portRec := httptest.NewRecorder()
	mux.ServeHTTP(portRec, portReq)
	assert.Equal(t, http.StatusOK, portRec.Code)

	var portResp map[string]interface{}
	err = json.Unmarshal(portRec.Body.Bytes(), &portResp)
	require.NoError(t, err)
	assert.Equal(t, "cust-dynamo-999", portResp["customer_id"])
	assert.Equal(t, "Diana Prince", portResp["customer_name"])
	assert.Equal(t, "port-cust-dynamo-999", portResp["portfolio_id"])
	assert.Equal(t, 0.0, portResp["total_balance_usd"])

	// 3. API Keys: Diana starts with 0 keys (not Chloe Momentum Bot)
	keysReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/api-keys", nil)
	keysReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	keysRec := httptest.NewRecorder()
	mux.ServeHTTP(keysRec, keysReq)
	assert.Equal(t, http.StatusOK, keysRec.Code)

	var keys []map[string]interface{}
	err = json.Unmarshal(keysRec.Body.Bytes(), &keys)
	require.NoError(t, err)
	assert.Empty(t, keys, "new customer should start with empty API keys list, not hardcoded bot keys")

	// Create an API key for Diana
	createKeyBody := bytes.NewBufferString(`{"name":"Diana Alpha Bot","permissions":["execute_orders"]}`)
	createKeyReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/api-keys", createKeyBody)
	createKeyReq.Header.Set("Content-Type", "application/json")
	createKeyReq.Header.Set("X-CSRF-Token", csrfVal)
	createKeyReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	createKeyReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	createKeyRec := httptest.NewRecorder()
	mux.ServeHTTP(createKeyRec, createKeyReq)
	assert.Equal(t, http.StatusCreated, createKeyRec.Code)

	var createdKey map[string]interface{}
	err = json.Unmarshal(createKeyRec.Body.Bytes(), &createdKey)
	require.NoError(t, err)
	assert.Equal(t, "Diana Alpha Bot", createdKey["name"])
	assert.Equal(t, "cust-dynamo-999", createdKey["customer_id"])

	// Diana now has 1 key
	keysReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/customer/api-keys", nil)
	keysReq2.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	keysRec2 := httptest.NewRecorder()
	mux.ServeHTTP(keysRec2, keysReq2)
	assert.Equal(t, http.StatusOK, keysRec2.Code)

	var keys2 []map[string]interface{}
	err = json.Unmarshal(keysRec2.Body.Bytes(), &keys2)
	require.NoError(t, err)
	require.Len(t, keys2, 1)
	assert.Equal(t, "Diana Alpha Bot", keys2[0]["name"])

	// 4. Webhooks: Diana starts with 0 webhooks (not Arthur's or Chloe's webhook)
	whReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/webhooks", nil)
	whReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	whRec := httptest.NewRecorder()
	mux.ServeHTTP(whRec, whReq)
	assert.Equal(t, http.StatusOK, whRec.Code)

	var webhooks []map[string]interface{}
	err = json.Unmarshal(whRec.Body.Bytes(), &webhooks)
	require.NoError(t, err)
	assert.Empty(t, webhooks, "new customer should start with empty webhooks list")

	// Create a webhook for Diana
	createWhBody := bytes.NewBufferString(`{"target_url":"https://diana-bot.example.com/webhook","events":["events.trade.executed.v1"]}`)
	createWhReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/webhooks", createWhBody)
	createWhReq.Header.Set("Content-Type", "application/json")
	createWhReq.Header.Set("X-CSRF-Token", csrfVal)
	createWhReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	createWhReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	createWhRec := httptest.NewRecorder()
	mux.ServeHTTP(createWhRec, createWhReq)
	assert.Equal(t, http.StatusCreated, createWhRec.Code)

	// Diana now has 1 webhook
	whReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/customer/webhooks", nil)
	whReq2.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	whRec2 := httptest.NewRecorder()
	mux.ServeHTTP(whRec2, whReq2)
	assert.Equal(t, http.StatusOK, whRec2.Code)

	var webhooks2 []map[string]interface{}
	err = json.Unmarshal(whRec2.Body.Bytes(), &webhooks2)
	require.NoError(t, err)
	require.Len(t, webhooks2, 1)
	assert.Equal(t, "https://diana-bot.example.com/webhook", webhooks2[0]["target_url"])

	// 5. Order placement dynamically resolves Diana's portfolio
	orderBody := bytes.NewBufferString(`{"symbol":"AAPL","side":"buy","quantity":2}`)
	orderReq := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", orderBody)
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("X-CSRF-Token", csrfVal)
	orderReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sessToken})
	orderReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
	orderRec := httptest.NewRecorder()
	mux.ServeHTTP(orderRec, orderReq)
	assert.Equal(t, http.StatusOK, orderRec.Code)

	var orderResp map[string]interface{}
	err = json.Unmarshal(orderRec.Body.Bytes(), &orderResp)
	require.NoError(t, err)
	assert.Equal(t, "port-cust-dynamo-999", orderResp["portfolio_id"])
}


