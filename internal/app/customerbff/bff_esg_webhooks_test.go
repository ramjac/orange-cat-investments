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
	_, mux := setupTestBFF()

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
