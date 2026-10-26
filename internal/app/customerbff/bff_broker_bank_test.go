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

	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	brokerMux.HandleFunc("GET /v2/stocks/quotes/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"quotes": map[string]alpaca.Quote{
				"AAPL": {BidPrice: 225.4, AskPrice: 225.6},
				"NVDA": {BidPrice: 139.9, AskPrice: 140.1},
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

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	bff := NewCustomerBFF(newMockCustomerBLLClient(), logger)
	bff.SetBankClient(bankClient)
	sessStore := sessionvalkey.NewMemoryCustomerSessionStore()
	sessStore.SetSession(&CustomerSession{
		SessionID:  "sess-arthur-token",
		CustomerID: "cust-longterm-arthur",
		Email:      "arthur@oci.local",
		Name:       "Arthur Pendelton",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})
	bff.SetSessionStore(sessStore)

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
