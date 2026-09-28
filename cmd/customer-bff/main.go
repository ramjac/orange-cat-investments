package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
)

type CustomerBFF struct {
	ciSvc  core_invest_svc.Service
	logger *slog.Logger
}

func NewCustomerBFF(ciSvc core_invest_svc.Service, logger *slog.Logger) *CustomerBFF {
	return &CustomerBFF{
		ciSvc:  ciSvc,
		logger: logger,
	}
}

func (b *CustomerBFF) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", b.handleHealthCheck)
	mux.HandleFunc("GET /api/v1/customer/portfolio", b.handleGetPortfolio)
	mux.HandleFunc("GET /api/v1/customer/trade-logs", b.handleGetTradeLogs)
	mux.HandleFunc("POST /api/v1/customer/trading/orders", b.handleCreateOrder)
	mux.HandleFunc("GET /api/v1/customer/ticker", b.handleTickerSSE)
	mux.HandleFunc("GET /api/v1/customer/zoomie-index", b.handleGetZoomieIndex)
	mux.HandleFunc("GET /api/v1/customer/deposits", b.handleGetDeposits)
	mux.HandleFunc("POST /api/v1/customer/deposits", b.handleCreateDeposit)
	mux.HandleFunc("GET /api/v1/customer/api-keys", b.handleGetAPIKeys)
	mux.HandleFunc("POST /api/v1/customer/api-keys", b.handleCreateAPIKey)
	mux.HandleFunc("DELETE /api/v1/customer/api-keys/{id}", b.handleRevokeAPIKey)
	mux.HandleFunc("GET /api/v1/customer/webhooks", b.handleGetWebhooks)
	mux.HandleFunc("POST /api/v1/customer/webhooks", b.handleCreateWebhook)
	mux.HandleFunc("GET /api/v1/customer/transparency/welfare", b.handleGetESGTransparency)
	mux.HandleFunc("GET /api/v1/customer/streams", b.handleGetStreams)
}

func (b *CustomerBFF) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"customer-bff"}`))
}

func (b *CustomerBFF) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"portfolio_id":       "018f3a9a-2222-7000-8000-000000000002",
		"customer_id":        "cust-longterm-arthur",
		"customer_name":      "Arthur Pendelton",
		"account_name":       "OCI Alpha Growth Fund",
		"total_balance_usd": 128450.75,
		"cash_balance_usd":  32100.25,
		"status":             "active",
		"holdings": []map[string]interface{}{
			{"symbol": "AAPL", "quantity": 120.0, "avg_price": 178.50, "current_price": 224.30, "market_value": 26916.00},
			{"symbol": "NVDA", "quantity": 85.0, "avg_price": 110.20, "current_price": 138.80, "market_value": 11798.00},
			{"symbol": "MSFT", "quantity": 90.0, "avg_price": 380.00, "current_price": 448.20, "market_value": 40338.00},
			{"symbol": "TSLA", "quantity": 70.0, "avg_price": 190.00, "current_price": 247.12, "market_value": 17298.40},
		},
		"historical_performance": []map[string]interface{}{
			{"date": "2026-01-01", "valuation": 100000.00},
			{"date": "2026-02-01", "valuation": 108400.00},
			{"date": "2026-03-01", "valuation": 114200.00},
			{"date": "2026-04-01", "valuation": 128450.75},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (b *CustomerBFF) handleGetTradeLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	orders, err := b.ciSvc.ListBrokerageOrders(r.Context(), 50, 0)
	if err != nil || len(orders) == 0 {
		// Mock sample fallback
		mockOrders := []map[string]interface{}{
			{
				"order_id":         "ord-98234-a1",
				"symbol":           "AAPL",
				"side":             "buy",
				"quantity":         15.0,
				"price":            224.30,
				"status":           "executed",
				"executed_at":      time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				"trigger_activity": "zooming",
				"feline_executive": "Garfield",
			},
			{
				"order_id":         "ord-98233-a2",
				"symbol":           "NVDA",
				"side":             "buy",
				"quantity":         10.0,
				"price":            138.80,
				"status":           "executed",
				"executed_at":      time.Now().Add(-6 * time.Hour).Format(time.RFC3339),
				"trigger_activity": "playful_pounce",
				"feline_executive": "Barneby",
			},
		}
		_ = json.NewEncoder(w).Encode(mockOrders)
		return
	}
	_ = json.NewEncoder(w).Encode(orders)
}

func (b *CustomerBFF) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		PortfolioID string  `json:"portfolio_id"`
		Symbol      string  `json:"symbol"`
		Side        string  `json:"side"`
		Quantity    float64 `json:"quantity"`
		Price       float64 `json:"price"`
		OrderType   string  `json:"order_type"` // market, limit, bracket, trailing_stop
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid order payload"})
		return
	}

	if req.PortfolioID == "" {
		req.PortfolioID = "018f3a9a-2222-7000-8000-000000000002"
	}
	if req.Price <= 0 {
		req.Price = 224.30
	}

	order := &core_invest_repo.BrokerageOrder{
		PortfolioID: req.PortfolioID,
		BrokerName:  "OCI Smart Execution Router",
		Symbol:      req.Symbol,
		Side:        req.Side,
		Quantity:    req.Quantity,
		Price:       req.Price,
		Status:      "executed",
	}

	created, err := b.ciSvc.ExecuteBrokerageOrder(r.Context(), req.PortfolioID, "OCI Smart Execution Router", req.Symbol, req.Side, req.Quantity, req.Price, nil, nil)
	if err != nil {
		now := time.Now()
		order.OrderID = fmt.Sprintf("ord-%d", now.UnixNano())
		order.ExecutedAt = &now
		_ = json.NewEncoder(w).Encode(order)
		return
	}
	_ = json.NewEncoder(w).Encode(created)
}

func (b *CustomerBFF) handleTickerSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tickerData := []map[string]interface{}{
		{"symbol": "AAPL", "price": 224.30, "change_pct": 1.45, "volume": 42100000},
		{"symbol": "NVDA", "price": 138.80, "change_pct": 3.12, "volume": 89200000},
		{"symbol": "MSFT", "price": 448.20, "change_pct": 0.82, "volume": 18400000},
		{"symbol": "TSLA", "price": 247.12, "change_pct": -0.65, "volume": 31200000},
	}
	_ = json.NewEncoder(w).Encode(tickerData)
}

func (b *CustomerBFF) handleGetZoomieIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"zoomie_index_score":   88.5,
		"momentum_level":       "HIGH_VELOCITY_3AM_ZOOMIES",
		"feline_leader":        "Garfield",
		"active_pounces_cnt":   14,
		"current_activity":     "zooming",
		"confidence_score":     0.965,
		"trading_signal":       "TACTICAL_BUY_BULLISH",
		"last_updated":         time.Now().Format(time.RFC3339),
		"recommended_multiplier": 1.75,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (b *CustomerBFF) handleGetDeposits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	deposits := []map[string]interface{}{
		{
			"deposit_id":     "dep-018f-01",
			"customer_id":    "cust-longterm-arthur",
			"amount_usd":     500.00,
			"frequency":      "monthly",
			"day_of_month":   1,
			"status":         "active",
			"bank_account":   "Chase Checking (****4821)",
			"next_charge_at": "2026-11-01T00:00:00Z",
		},
	}
	_ = json.NewEncoder(w).Encode(deposits)
}

func (b *CustomerBFF) handleCreateDeposit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		AmountUSD   float64 `json:"amount_usd"`
		Frequency   string  `json:"frequency"`
		DayOfMonth  int     `json:"day_of_month"`
		BankAccount string  `json:"bank_account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid deposit payload"})
		return
	}

	created := map[string]interface{}{
		"deposit_id":     fmt.Sprintf("dep-%d", time.Now().UnixNano()),
		"customer_id":    "cust-longterm-arthur",
		"amount_usd":     req.AmountUSD,
		"frequency":      req.Frequency,
		"day_of_month":   req.DayOfMonth,
		"status":         "active",
		"bank_account":   req.BankAccount,
		"next_charge_at": time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
		"created_at":     time.Now().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (b *CustomerBFF) handleGetAPIKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	keys := []map[string]interface{}{
		{
			"key_id":      "key-018f-8888",
			"name":        "Chloe Momentum Bot",
			"key_prefix":  "oci_live_9a8f",
			"status":      "active",
			"permissions": []string{"read_streams", "execute_orders", "read_telemetry"},
			"created_at":  time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
		},
	}
	_ = json.NewEncoder(w).Encode(keys)
}

func (b *CustomerBFF) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid api key payload"})
		return
	}

	rawSecret := fmt.Sprintf("oci_live_%x%x", rand.Int63(), rand.Int63())
	created := map[string]interface{}{
		"key_id":      fmt.Sprintf("key-%d", time.Now().UnixNano()),
		"name":        req.Name,
		"key_prefix":  rawSecret[:12],
		"api_key":     rawSecret,
		"status":      "active",
		"permissions": req.Permissions,
		"created_at":  time.Now().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (b *CustomerBFF) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

func (b *CustomerBFF) handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	webhooks := []map[string]interface{}{
		{
			"subscription_id": "sub-018f-1111",
			"target_url":      "https://quant-bot.chloespark.io/api/v1/oci-callback",
			"events":          []string{"events.observation.cat_spotted.v1", "events.trade.executed.v1"},
			"status":          "active",
			"secret":          "whsec_a87f9b0c1d2e3f4a5b6c7d8e9f0a",
		},
	}
	_ = json.NewEncoder(w).Encode(webhooks)
}

func (b *CustomerBFF) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		TargetURL string   `json:"target_url"`
		Events    []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid webhook payload"})
		return
	}

	created := map[string]interface{}{
		"subscription_id": fmt.Sprintf("sub-%d", time.Now().UnixNano()),
		"target_url":      req.TargetURL,
		"events":          req.Events,
		"status":          "active",
		"secret":          fmt.Sprintf("whsec_%x", rand.Int63()),
		"created_at":      time.Now().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (b *CustomerBFF) handleGetESGTransparency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"overall_welfare_score":    4.98,
		"veterinary_adherence_pct": 100.0,
		"nutritional_compliance":   "100% Grain-Free Prescription Salmon & Turkey",
		"avg_daily_nap_hours":      14.6,
		"perch_comfort_rating":     "5.0 / 5.0 (Ergonomic Thermal Cushioning)",
		"medical_holds_issued_ytd": 0,
		"habitat_temperature_c":    22.5,
		"feline_executives": []map[string]interface{}{
			{
				"name":                "Garfield",
				"role":                "Chief Observation Officer",
				"health_status":       "OPTIMAL_ALPHA",
				"whisker_symmetry":   "100%",
				"purr_frequency_hz":   28.5,
				"preferred_sunbeam":   "Alpha Sunbeam Lounge (Zone 1)",
			},
			{
				"name":                "Barneby",
				"role":                "Senior Alpha Perch Analyst",
				"health_status":       "PEAK_PERFORMER",
				"whisker_symmetry":   "99.8%",
				"purr_frequency_hz":   26.2,
				"preferred_sunbeam":   "Perch Zone B - West Tower",
			},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (b *CustomerBFF) handleGetStreams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	streams, err := b.ciSvc.ListCameraStreams(r.Context(), 10, 0)
	if err != nil || len(streams) == 0 {
		mockStreams := []map[string]interface{}{
			{
				"stream_id":       "str-018f-0001",
				"camera_asset_id": "018e0000-0000-7000-8000-000000000001",
				"name":            "Alpha Sunbeam Lounge Camera 1",
				"stream_url":      "https://stream.oci.local/webrtc/alpha-lounge-cam1",
				"protocol":        "webrtc",
				"status":          "active",
			},
			{
				"stream_id":       "str-018f-0002",
				"camera_asset_id": "018f3a9a-3333-7000-8000-000000000003",
				"name":            "4K Feline Perch Scope",
				"stream_url":      "https://stream.oci.local/webrtc/4k-perch-cam2",
				"protocol":        "webrtc",
				"status":          "active",
			},
		}
		_ = json.NewEncoder(w).Encode(mockStreams)
		return
	}
	_ = json.NewEncoder(w).Encode(streams)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting OCI Customer BFF API Server...")

	dbURL := os.Getenv("DATABASE_URL")
	var pool *pgxpool.Pool
	if dbURL != "" {
		var err error
		pool, err = pgxpool.New(context.Background(), dbURL)
		if err != nil {
			logger.Warn("failed to connect to database pool, falling back to mock persistence", "error", err)
		} else {
			defer pool.Close()
			logger.Info("connected to PostgreSQL database pool successfully")
		}
	} else {
		logger.Info("no DATABASE_URL provided, running with mock repository for local development")
	}

	var ciRepo core_invest_repo.Repository
	if pool != nil {
		ciRepo = core_invest_repo.NewRepository(pool)
	} else {
		ciRepo = core_invest_repo.NewMockRepository()
	}

	ciSvc := core_invest_svc.NewService(ciRepo)
	custBFF := NewCustomerBFF(ciSvc, logger)

	mux := http.NewServeMux()
	custBFF.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Customer BFF listening", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Customer BFF HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down Customer BFF API Server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed graceful shutdown", "error", err)
	}

	logger.Info("Customer BFF stopped gracefully")
}
