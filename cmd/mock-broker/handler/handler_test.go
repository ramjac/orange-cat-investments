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

func setupTestServer(t *testing.T) (*ledger.Store, *engine.MarketEngine, http.Handler) {
	store := ledger.NewStore()
	mktEngine := engine.NewMarketEngine(store, 100*time.Millisecond, engine.DefaultAssets())
	ledger.SeedDefaultAccounts(store, mktEngine.InitialPrices())

	mux := http.NewServeMux()
	h := NewHandler(store, mktEngine)
	h.RegisterRoutes(mux)

	return store, mktEngine, mux
}

func TestHealthCheck(t *testing.T) {
	_, _, handler := setupTestServer(t)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
}

func TestGetAccountAndPositions(t *testing.T) {
	_, _, handler := setupTestServer(t)
	arthurID := "018f3a9a-2222-7000-8000-000000000002"

	// 1. Get Account
	req := httptest.NewRequest("GET", "/v1/accounts/"+arthurID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var acc ledger.Account
	err := json.Unmarshal(rec.Body.Bytes(), &acc)
	require.NoError(t, err)
	assert.Equal(t, arthurID, acc.ID)
	assert.Equal(t, 32100.25, acc.Cash)

	// 2. Get Positions
	posReq := httptest.NewRequest("GET", "/v1/trading/accounts/"+arthurID+"/positions", nil)
	posRec := httptest.NewRecorder()
	handler.ServeHTTP(posRec, posReq)

	assert.Equal(t, http.StatusOK, posRec.Code)
	var positions []ledger.Position
	err = json.Unmarshal(posRec.Body.Bytes(), &positions)
	require.NoError(t, err)
	assert.Len(t, positions, 4)
}

func TestOrderExecution(t *testing.T) {
	_, _, handler := setupTestServer(t)
	chloeID := "018f3a9a-2222-7000-8000-000000000003"

	// Submit market buy order for 5 NVDA shares
	orderReq := map[string]any{
		"symbol": "NVDA",
		"qty":    5.0,
		"side":   "buy",
		"type":   "market",
	}
	body, _ := json.Marshal(orderReq)

	req := httptest.NewRequest("POST", "/v1/trading/accounts/"+chloeID+"/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var ord ledger.Order
	err := json.Unmarshal(rec.Body.Bytes(), &ord)
	require.NoError(t, err)
	assert.Equal(t, "filled", ord.Status)
	assert.Equal(t, 5.0, ord.FilledQty)
	assert.NotNil(t, ord.FilledAvgPrice)
}

func TestMarketDataEndpoints(t *testing.T) {
	_, _, handler := setupTestServer(t)

	// 1. Latest Quote
	req := httptest.NewRequest("GET", "/v2/stocks/AAPL/quotes/latest", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var quoteResp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &quoteResp)
	require.NoError(t, err)
	assert.Equal(t, "AAPL", quoteResp["symbol"])

	// 2. Multi-symbol Latest Bars
	barsReq := httptest.NewRequest("GET", "/v2/stocks/bars/latest?symbols=AAPL,NVDA", nil)
	barsRec := httptest.NewRecorder()
	handler.ServeHTTP(barsRec, barsReq)

	assert.Equal(t, http.StatusOK, barsRec.Code)
	var barsResp map[string]any
	err = json.Unmarshal(barsRec.Body.Bytes(), &barsResp)
	require.NoError(t, err)
	bars := barsResp["bars"].(map[string]any)
	assert.Contains(t, bars, "AAPL")
	assert.Contains(t, bars, "NVDA")

	// 3. Multi-symbol Latest Quotes
	quotesReq := httptest.NewRequest("GET", "/v2/stocks/quotes/latest?symbols=AAPL,NVDA", nil)
	quotesRec := httptest.NewRecorder()
	handler.ServeHTTP(quotesRec, quotesReq)

	assert.Equal(t, http.StatusOK, quotesRec.Code)
	var quotesResp map[string]any
	err = json.Unmarshal(quotesRec.Body.Bytes(), &quotesResp)
	require.NoError(t, err)
	quotes := quotesResp["quotes"].(map[string]any)
	assert.Contains(t, quotes, "AAPL")
	assert.Contains(t, quotes, "NVDA")
}
