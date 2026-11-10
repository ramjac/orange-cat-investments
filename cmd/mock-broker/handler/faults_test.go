package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-broker/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFaultTestServer(t *testing.T, enabled bool) *http.ServeMux {
	t.Helper()
	if enabled {
		t.Setenv("ENABLE_FAULT_INJECTION", "true")
	} else {
		t.Setenv("ENABLE_FAULT_INJECTION", "false")
	}
	store := ledger.NewStore()
	eng := engine.NewMarketEngine(store, 100*time.Millisecond, engine.DefaultAssets())
	h := NewHandler(store, eng)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func postFaults(t *testing.T, mux *http.ServeMux, payload string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/sim/faults", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Code
}

func deleteFaults(t *testing.T, mux *http.ServeMux) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/sim/faults", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Code
}

func TestBrokerFaults_NotRegisteredWhenDisabled(t *testing.T) {
	mux := setupFaultTestServer(t, false)

	req := httptest.NewRequest(http.MethodGet, "/sim/faults", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "fault endpoint must 404 when disabled")
}

func TestBrokerFaults_OutageAndReset(t *testing.T) {
	mux := setupFaultTestServer(t, true)

	assert.Equal(t, http.StatusOK, postFaults(t, mux, `{"outage":true}`))

	req := httptest.NewRequest(http.MethodGet, "/v1/accounts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "expected 503 during outage")

	stateReq := httptest.NewRequest(http.MethodGet, "/sim/faults", nil)
	stateRec := httptest.NewRecorder()
	mux.ServeHTTP(stateRec, stateReq)
	var state BrokerFaultsResponse
	require.NoError(t, json.Unmarshal(stateRec.Body.Bytes(), &state))
	assert.True(t, state.Outage)

	assert.Equal(t, http.StatusNoContent, deleteFaults(t, mux))

	healthyReq := httptest.NewRequest(http.MethodGet, "/v1/accounts", nil)
	healthyRec := httptest.NewRecorder()
	mux.ServeHTTP(healthyRec, healthyReq)
	assert.Equal(t, http.StatusOK, healthyRec.Code, "expected 200 after reset")
}

func createTestAccount(t *testing.T, mux *http.ServeMux) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"initial_cash": "100000"})
	req := httptest.NewRequest(http.MethodPost, "/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "account creation must succeed")
	var acc struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &acc))
	require.NotEmpty(t, acc.ID)
	return acc.ID
}

func TestBrokerFaults_RejectOrders(t *testing.T) {
	mux := setupFaultTestServer(t, true)

	accountID := createTestAccount(t, mux)

	assert.Equal(t, http.StatusOK, postFaults(t, mux, `{"reject_orders":true}`))

	orderReq := map[string]any{"symbol": "NVDA", "qty": 1.0, "side": "buy", "type": "market"}
	body, _ := json.Marshal(orderReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/trading/accounts/"+accountID+"/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusPreconditionFailed, rec.Code, "expected 422 for rejected order")

	// Non-order endpoints stay healthy.
	accountsReq := httptest.NewRequest(http.MethodGet, "/v1/accounts", nil)
	accountsRec := httptest.NewRecorder()
	mux.ServeHTTP(accountsRec, accountsReq)
	assert.Equal(t, http.StatusOK, accountsRec.Code)
}

func TestBrokerFaults_HaltedSymbols(t *testing.T) {
	mux := setupFaultTestServer(t, true)

	accountID := createTestAccount(t, mux)

	assert.Equal(t, http.StatusOK, postFaults(t, mux, `{"halted_symbols":["NVDA"]}`))

	mkOrderReq := func(symbol string) *http.Request {
		orderReq := map[string]any{"symbol": symbol, "qty": 1.0, "side": "buy", "type": "market"}
		body, _ := json.Marshal(orderReq)
		req := httptest.NewRequest(http.MethodPost, "/v1/trading/accounts/"+accountID+"/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	haltedReq := mkOrderReq("NVDA")
	haltedRec := httptest.NewRecorder()
	mux.ServeHTTP(haltedRec, haltedReq)
	assert.Equal(t, http.StatusUnprocessableEntity, haltedRec.Code, "expected 422 for halted symbol")

	otherReq := mkOrderReq("AAPL")
	otherRec := httptest.NewRecorder()
	mux.ServeHTTP(otherRec, otherReq)
	assert.Equal(t, http.StatusOK, otherRec.Code, "expected 200 for non-halted symbol")

	assert.Equal(t, http.StatusNoContent, deleteFaults(t, mux))

	recoveredReq := mkOrderReq("NVDA")
	recoveredRec := httptest.NewRecorder()
	mux.ServeHTTP(recoveredRec, recoveredReq)
	assert.Equal(t, http.StatusOK, recoveredRec.Code, "expected 200 after halt reset")
}

func TestBrokerFaults_Latency(t *testing.T) {
	mux := setupFaultTestServer(t, true)

	assert.Equal(t, http.StatusOK, postFaults(t, mux, `{"latency_ms":250}`))

	req := httptest.NewRequest(http.MethodGet, "/v1/accounts", nil)
	rec := httptest.NewRecorder()
	start := time.Now()
	mux.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.GreaterOrEqual(t, elapsed, 250*time.Millisecond, "expected injected latency to be applied")
}
