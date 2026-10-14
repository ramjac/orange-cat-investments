package alpaca

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlpacaClient(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/accounts/acc-1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "KEY123", r.Header.Get("APCA-API-KEY-ID"))
		assert.Equal(t, "SECRET456", r.Header.Get("APCA-API-SECRET-KEY"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Account{
			ID:     "acc-1",
			Cash:   25000.50,
			Status: "ACTIVE",
		})
	})

	mux.HandleFunc("POST /v1/trading/accounts/acc-1/orders", func(w http.ResponseWriter, r *http.Request) {
		var req CreateOrderRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "AAPL", req.Symbol)
		assert.Equal(t, 10.0, req.Qty)

		avgPrice := 220.50
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Order{
			ID:             "ord-123",
			Symbol:         req.Symbol,
			Qty:            req.Qty,
			FilledQty:      req.Qty,
			FilledAvgPrice: &avgPrice,
			Status:         "filled",
		})
	})

	mux.HandleFunc("GET /v2/stocks/bars/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bars": map[string]Bar{
				"AAPL": {Close: 224.50, Volume: 1000},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		APIKey:    "KEY123",
		APISecret: "SECRET456",
	})

	ctx := context.Background()

	// 1. GetAccount
	acc, err := client.GetAccount(ctx, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, "acc-1", acc.ID)
	assert.Equal(t, 25000.50, acc.Cash)

	// 2. CreateOrder
	ord, err := client.CreateOrder(ctx, "acc-1", CreateOrderRequest{
		Symbol: "AAPL",
		Qty:    10.0,
		Side:   "buy",
		Type:   "market",
	})
	require.NoError(t, err)
	assert.Equal(t, "ord-123", ord.ID)
	assert.Equal(t, "filled", ord.Status)

	// 3. GetLatestBars
	bars, err := client.GetLatestBars(ctx, []string{"AAPL"})
	require.NoError(t, err)
	assert.Contains(t, bars, "AAPL")
	assert.Equal(t, 224.50, bars["AAPL"].Close)
}
