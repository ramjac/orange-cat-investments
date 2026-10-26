package customerbff

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func BenchmarkHandleTickerSSE(b *testing.B) {
	brokerMux := http.NewServeMux()
	brokerMux.HandleFunc("GET /v2/stocks/bars/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bars": map[string]alpaca.Bar{
				"AAPL": {Open: 220.0, Close: 225.5, Volume: 50000},
				"NVDA": {Open: 135.0, Close: 140.0, Volume: 80000},
				"MSFT": {Open: 400.0, Close: 405.0, Volume: 60000},
				"TSLA": {Open: 250.0, Close: 255.0, Volume: 70000},
			},
		})
	})
	brokerMux.HandleFunc("GET /v2/stocks/quotes/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"quotes": map[string]alpaca.Quote{
				"AAPL": {BidPrice: 225.4, AskPrice: 225.6},
				"NVDA": {BidPrice: 139.9, AskPrice: 140.1},
				"MSFT": {BidPrice: 404.9, AskPrice: 405.1},
				"TSLA": {BidPrice: 254.9, AskPrice: 255.1},
			},
		})
	})
	brokerMux.HandleFunc("GET /v2/stocks/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "STOCK",
			"quote":  alpaca.Quote{BidPrice: 100.0, AskPrice: 102.0},
		})
	})

	brokerServer := httptest.NewServer(brokerMux)
	defer brokerServer.Close()

	brokerClient := alpaca.NewClient(alpaca.Config{
		BaseURL: brokerServer.URL,
	})

	bff, mux := setupTestBFF()
	bff.SetBrokerClient(brokerClient)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/ticker", nil)
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
	assert.Equal(t, 70, len(secretStr), "webhook secret should be prefix 'whsec_' (6 chars) + 64 hex chars (32 bytes = 256 bits entropy)")
	assert.True(t, secretStr[:6] == "whsec_")
}
