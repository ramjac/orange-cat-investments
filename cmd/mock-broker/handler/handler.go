package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-broker/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-broker/ledger"
)

type Handler struct {
	store  *ledger.Store
	engine *engine.MarketEngine
}

func NewHandler(store *ledger.Store, engine *engine.MarketEngine) *Handler {
	return &Handler{
		store:  store,
		engine: engine,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("GET /health", h.handleHealthz)

	// Accounts API
	mux.HandleFunc("POST /v1/accounts", h.handleCreateAccount)
	mux.HandleFunc("GET /v1/accounts", h.handleListAccounts)
	mux.HandleFunc("GET /v1/accounts/{account_id}", h.handleGetAccount)

	// Trading - Orders API
	mux.HandleFunc("POST /v1/trading/accounts/{account_id}/orders", h.handleCreateOrder)
	mux.HandleFunc("GET /v1/trading/accounts/{account_id}/orders", h.handleListOrders)
	mux.HandleFunc("GET /v1/trading/accounts/{account_id}/orders/{order_id}", h.handleGetOrder)
	mux.HandleFunc("DELETE /v1/trading/accounts/{account_id}/orders/{order_id}", h.handleCancelOrder)

	// Also support non-nested Alpaca Trading API v2 order endpoints for flexibility
	mux.HandleFunc("POST /v2/orders", h.handleCreateOrderV2)
	mux.HandleFunc("GET /v2/orders", h.handleListOrdersV2)

	// Trading - Positions API
	mux.HandleFunc("GET /v1/trading/accounts/{account_id}/positions", h.handleListPositions)
	mux.HandleFunc("GET /v1/trading/accounts/{account_id}/positions/{symbol}", h.handleGetPosition)
	mux.HandleFunc("GET /v1/trading/accounts/{account_id}/portfolio/history", h.handleGetPortfolioHistory)

	// Market Data API (Alpaca Data v2)
	mux.HandleFunc("GET /v2/stocks/bars/latest", h.handleGetLatestBars)
	mux.HandleFunc("GET /v2/stocks/{symbol}/quotes/latest", h.handleGetLatestQuote)
	mux.HandleFunc("GET /v2/stocks/{symbol}/bars", h.handleGetBars)
	mux.HandleFunc("GET /v2/stocks/snapshots", h.handleGetSnapshots)

	// Trade Updates SSE Stream
	mux.HandleFunc("GET /v1/events/trade_updates", h.handleTradeUpdatesSSE)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "healthy",
		"service":   "faux-alpaca-broker",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// -------------------------------------------------------------
// Accounts API
// -------------------------------------------------------------

func (h *Handler) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Contact struct {
			EmailAddress string `json:"email_address"`
		} `json:"contact"`
		Identity struct {
			GivenName  string `json:"given_name"`
			FamilyName string `json:"family_name"`
		} `json:"identity"`
		InitialCash float64 `json:"initial_cash,string"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid account creation payload"})
		return
	}

	acc := &ledger.Account{
		Cash:         req.InitialCash,
		CustomerName: fmt.Sprintf("%s %s", req.Identity.GivenName, req.Identity.FamilyName),
	}
	if acc.Cash <= 0 {
		acc.Cash = 10000.00 // Default demo cash
	}

	if err := h.store.CreateAccount(acc); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(acc)
}

func (h *Handler) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accounts := h.store.ListAccounts()
	_ = json.NewEncoder(w).Encode(accounts)
}

func (h *Handler) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")

	acc, err := h.store.GetAccount(accountID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 40410000, "message": "account not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(acc)
}

// -------------------------------------------------------------
// Orders API
// -------------------------------------------------------------

func (h *Handler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")

	var req ledger.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "malformed order payload"})
		return
	}

	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	quote, ok := h.engine.GetQuote(req.Symbol)
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("asset %s not found in universe", req.Symbol)})
		return
	}

	order, err := h.store.CreateOrder(accountID, req, quote)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(order)
}

func (h *Handler) handleCreateOrderV2(w http.ResponseWriter, r *http.Request) {
	// Convenience fallback for trading API: default to Arthur's account if account not in path
	defaultAccID := "018f3a9a-2222-7000-8000-000000000002"
	accHeader := r.Header.Get("X-Account-ID")
	if accHeader != "" {
		defaultAccID = accHeader
	}

	r.SetPathValue("account_id", defaultAccID)
	h.handleCreateOrder(w, r)
}

func (h *Handler) handleListOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	orders := h.store.ListOrders(accountID, status, limit)
	_ = json.NewEncoder(w).Encode(orders)
}

func (h *Handler) handleListOrdersV2(w http.ResponseWriter, r *http.Request) {
	defaultAccID := "018f3a9a-2222-7000-8000-000000000002"
	if accHeader := r.Header.Get("X-Account-ID"); accHeader != "" {
		defaultAccID = accHeader
	}
	r.SetPathValue("account_id", defaultAccID)
	h.handleListOrders(w, r)
}

func (h *Handler) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")
	orderID := r.PathValue("order_id")

	order, err := h.store.GetOrder(accountID, orderID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "order not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(order)
}

func (h *Handler) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")
	orderID := r.PathValue("order_id")

	order, err := h.store.CancelOrder(accountID, orderID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(order)
}

// -------------------------------------------------------------
// Positions API
// -------------------------------------------------------------

func (h *Handler) handleListPositions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")

	positions := h.store.ListPositions(accountID)
	_ = json.NewEncoder(w).Encode(positions)
}

func (h *Handler) handleGetPosition(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")
	symbol := strings.ToUpper(r.PathValue("symbol"))

	pos, err := h.store.GetPosition(accountID, symbol)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "position not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(pos)
}

func (h *Handler) handleGetPortfolioHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accountID := r.PathValue("account_id")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1M"
	}

	history, err := h.store.GetPortfolioHistory(accountID, period)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "portfolio history unavailable"})
		return
	}

	_ = json.NewEncoder(w).Encode(history)
}

// -------------------------------------------------------------
// Market Data API (Alpaca Data v2)
// -------------------------------------------------------------

func (h *Handler) handleGetLatestBars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	symbolsParam := r.URL.Query().Get("symbols")
	symbols := strings.Split(symbolsParam, ",")

	bars := make(map[string]ledger.Bar)
	for _, s := range symbols {
		sym := strings.ToUpper(strings.TrimSpace(s))
		if sym == "" {
			continue
		}
		if bar, ok := h.engine.GetLatestBar(sym); ok {
			bars[sym] = bar
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"bars": bars})
}

func (h *Handler) handleGetLatestQuote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	symbol := strings.ToUpper(r.PathValue("symbol"))

	quote, ok := h.engine.GetQuote(symbol)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "symbol not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"symbol": symbol,
		"quote":  quote,
	})
}

func (h *Handler) handleGetBars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	symbol := strings.ToUpper(r.PathValue("symbol"))
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "1Min"
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	bars, ok := h.engine.GetBars(symbol, timeframe, limit, nil, nil)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "symbol not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"symbol": symbol,
		"bars":   bars,
	})
}

func (h *Handler) handleGetSnapshots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	symbolsParam := r.URL.Query().Get("symbols")
	symbols := strings.Split(symbolsParam, ",")

	snapshots := make(map[string]any)
	for _, s := range symbols {
		sym := strings.ToUpper(strings.TrimSpace(s))
		if sym == "" {
			continue
		}
		quote, hasQuote := h.engine.GetQuote(sym)
		bar, hasBar := h.engine.GetLatestBar(sym)
		price, changePct, hasPrice := h.engine.GetPriceSummary(sym)

		if hasQuote || hasBar || hasPrice {
			snapshots[sym] = map[string]any{
				"latestQuote": quote,
				"latestTrade": map[string]any{
					"p": price,
					"t": quote.Timestamp,
				},
				"minuteBar":   bar,
				"dailyBar":    bar,
				"prevDailyBar": map[string]any{
					"c": price / (1.0 + (changePct / 100.0)),
				},
			}
		}
	}

	_ = json.NewEncoder(w).Encode(snapshots)
}

// -------------------------------------------------------------
// Real-Time Events SSE
// -------------------------------------------------------------

func (h *Handler) handleTradeUpdatesSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := h.store.SubscribeTradeUpdates()
	defer unsubscribe()

	// Initial heartbeat
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case update, ok := <-ch:
			if !ok {
				return
			}
			bytes, err := json.Marshal(update)
			if err == nil {
				fmt.Fprintf(w, "event: trade_update\ndata: %s\n\n", string(bytes))
				flusher.Flush()
			}
		}
	}
}
