package employeebff

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type quantStore struct {
	mu        sync.RWMutex
	streams   []map[string]any
	orders    []map[string]any
	backtests []map[string]any
	deposits  []map[string]any
}

var globalQuantStore = &quantStore{
	streams: []map[string]any{
		{
			"stream_id":       "stream-018f3a9a-3333-7000-8000-000000000003",
			"camera_asset_id": "018f3a9a-3333-7000-8000-000000000003",
			"stream_url":      "rtsp://edge-cam-01.local/live",
			"protocol":        "rtsp",
			"status":          "online",
			"created_at":      time.Now().UTC().Format(time.RFC3339),
		},
	},
	orders: []map[string]any{
		{
			"order_id":     "ord-018f3a9a-2222-7000-8000-000000000001",
			"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
			"broker_name":  "Interactive Brokers",
			"symbol":       "AAPL",
			"side":         "buy",
			"quantity":     10.0,
			"price":        225.50,
			"status":       "filled",
			"created_at":   time.Now().UTC().Format(time.RFC3339),
		},
	},
	backtests: []map[string]any{
		{
			"backtest_id": "backtest-018f3a9a-4444-7000-8000-000000000001",
			"strategy_id": "018f3a9a-4444-7000-8000-000000000004",
			"status":      "completed",
			"alpha_score": 1.42,
			"created_at":  time.Now().UTC().Format(time.RFC3339),
		},
	},
}

func randomHexID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// RegisterQuantRoutes registers quant, research, and core-invest employee API routes.
func (b *EmployeeBFF) RegisterQuantRoutes(mux *http.ServeMux) {
	// Streams
	listStreams := func(w http.ResponseWriter, r *http.Request) {
		globalQuantStore.mu.RLock()
		defer globalQuantStore.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(globalQuantStore.streams)
	}
	createStream := func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req["stream_id"] == nil || req["stream_id"] == "" {
			req["stream_id"] = randomHexID("stream")
		}
		req["status"] = "online"
		req["created_at"] = time.Now().UTC().Format(time.RFC3339)

		globalQuantStore.mu.Lock()
		globalQuantStore.streams = append(globalQuantStore.streams, req)
		globalQuantStore.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(req)
	}

	mux.HandleFunc("GET /core-invest/streams", listStreams)
	mux.HandleFunc("GET /api/v1/core-invest/streams", listStreams)
	mux.HandleFunc("POST /core-invest/streams", createStream)
	mux.HandleFunc("POST /api/v1/core-invest/streams", createStream)

	// Brokerage Orders
	listOrders := func(w http.ResponseWriter, r *http.Request) {
		globalQuantStore.mu.RLock()
		defer globalQuantStore.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(globalQuantStore.orders)
	}
	createOrder := func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req["order_id"] == nil || req["order_id"] == "" {
			req["order_id"] = randomHexID("ord")
		}
		req["status"] = "filled"
		req["created_at"] = time.Now().UTC().Format(time.RFC3339)

		globalQuantStore.mu.Lock()
		globalQuantStore.orders = append(globalQuantStore.orders, req)
		globalQuantStore.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(req)
	}

	mux.HandleFunc("GET /core-invest/brokerage/orders", listOrders)
	mux.HandleFunc("GET /api/v1/core-invest/brokerage/orders", listOrders)
	mux.HandleFunc("POST /core-invest/brokerage/orders", createOrder)
	mux.HandleFunc("POST /api/v1/core-invest/brokerage/orders", createOrder)

	// Backtest Runs
	listBacktests := func(w http.ResponseWriter, r *http.Request) {
		globalQuantStore.mu.RLock()
		defer globalQuantStore.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(globalQuantStore.backtests)
	}
	createBacktest := func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req["backtest_id"] == nil || req["backtest_id"] == "" {
			req["backtest_id"] = randomHexID("backtest")
		}
		req["status"] = "completed"
		req["alpha_score"] = 1.45
		req["created_at"] = time.Now().UTC().Format(time.RFC3339)

		globalQuantStore.mu.Lock()
		globalQuantStore.backtests = append(globalQuantStore.backtests, req)
		globalQuantStore.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(req)
	}

	mux.HandleFunc("GET /core-invest/backtesting/runs", listBacktests)
	mux.HandleFunc("GET /api/v1/core-invest/backtesting/runs", listBacktests)
	mux.HandleFunc("POST /core-invest/backtesting/runs", createBacktest)
	mux.HandleFunc("POST /api/v1/core-invest/backtesting/runs", createBacktest)

	// Recurring Deposits
	listDeposits := func(w http.ResponseWriter, r *http.Request) {
		globalQuantStore.mu.RLock()
		defer globalQuantStore.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(globalQuantStore.deposits)
	}
	createDeposit := func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req["deposit_id"] == nil || req["deposit_id"] == "" {
			req["deposit_id"] = randomHexID("dep")
		}
		req["status"] = "active"
		req["created_at"] = time.Now().UTC().Format(time.RFC3339)

		globalQuantStore.mu.Lock()
		globalQuantStore.deposits = append(globalQuantStore.deposits, req)
		globalQuantStore.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(req)
	}

	mux.HandleFunc("GET /core-invest/recurring-deposits", listDeposits)
	mux.HandleFunc("GET /api/v1/core-invest/recurring-deposits", listDeposits)
	mux.HandleFunc("POST /core-invest/recurring-deposits", createDeposit)
	mux.HandleFunc("POST /api/v1/core-invest/recurring-deposits", createDeposit)
}
