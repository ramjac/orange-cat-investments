package main

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	core_invest_repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	core_invest_svc "github.com/orange-cat-investments/oci/internal/service/core_invest"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
)

type CustomerSession struct {
	SessionID  string    `json:"session_id"`
	CustomerID string    `json:"customer_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*CustomerSession
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*CustomerSession),
	}
	// Seed well-known sessions for local dev and testing
	store.sessions["sess-arthur-token"] = &CustomerSession{
		SessionID:  "sess-arthur-token",
		CustomerID: "cust-longterm-arthur",
		Email:      "arthur@oci.local",
		Name:       "Arthur Pendelton",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	store.sessions["sess-chloe-token"] = &CustomerSession{
		SessionID:  "sess-chloe-token",
		CustomerID: "cust-active-chloe",
		Email:      "chloe@oci.local",
		Name:       "Chloe Spark",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	return store
}

func (s *SessionStore) Get(token string) (*CustomerSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return sess, true
}

func (s *SessionStore) Create(customerID, name, email string) *CustomerSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := make([]byte, 16)
	_, _ = crand.Read(b)
	token := "sess-" + hex.EncodeToString(b)
	sess := &CustomerSession{
		SessionID:  token,
		CustomerID: customerID,
		Email:      email,
		Name:       name,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	s.sessions[token] = sess
	return sess
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

type contextKey string

const customerContextKey contextKey = "customer_session"

func CustomerFromContext(ctx context.Context) *CustomerSession {
	if s, ok := ctx.Value(customerContextKey).(*CustomerSession); ok {
		return s
	}
	return nil
}

func generateRandomToken(n int) string {
	b := make([]byte, n)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)
}

func getCSRFCookie(r *http.Request) string {
	if c, err := r.Cookie("csrf_token"); err == nil && c.Value != "" {
		return c.Value
	}
	if c, err := r.Cookie("__Host-csrf-token"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

func setCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by client JS for double-submit header
		SameSite: http.SameSiteLaxMode,
	})
}

type CustomerBFF struct {
	ciSvc    core_invest_svc.Service
	wfSvc    workforce_svc.Service
	sessions *SessionStore
	logger   *slog.Logger
}

func NewCustomerBFF(ciSvc core_invest_svc.Service, wfSvc workforce_svc.Service, logger *slog.Logger) *CustomerBFF {
	return &CustomerBFF{
		ciSvc:    ciSvc,
		wfSvc:    wfSvc,
		sessions: NewSessionStore(),
		logger:   logger,
	}
}

func (b *CustomerBFF) sessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		if c, err := r.Cookie("__Host-customer-session"); err == nil && c.Value != "" {
			token = c.Value
		} else if c, err := r.Cookie("customer_session"); err == nil && c.Value != "" {
			token = c.Value
		}

		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			token = r.URL.Query().Get("session_token")
		}

		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
			return
		}

		sess, ok := b.sessions.Get(token)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: invalid or expired customer session"})
			return
		}

		if getCSRFCookie(r) == "" {
			setCSRFCookie(w, generateRandomToken(16))
		}

		ctx := context.WithValue(r.Context(), customerContextKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (b *CustomerBFF) csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			headerToken := r.Header.Get("X-CSRF-Token")
			cookieToken := getCSRFCookie(r)

			if headerToken == "" || cookieToken == "" || headerToken != cookieToken {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden: invalid or missing CSRF token"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (b *CustomerBFF) RegisterRoutes(mux *http.ServeMux) {
	// Public routes
	mux.HandleFunc("GET /healthz", b.handleHealthCheck)
	mux.HandleFunc("GET /api/v1/customer/csrf", b.handleGetCSRFToken)
	mux.HandleFunc("POST /api/v1/customer/auth/login", b.handleLogin)
	mux.HandleFunc("POST /api/v1/customer/auth/logout", b.handleLogout)

	// Protected read endpoints (Session Auth)
	mux.Handle("GET /api/v1/customer/auth/session", b.sessionAuth(http.HandlerFunc(b.handleGetSession)))
	mux.Handle("GET /api/v1/customer/portfolio", b.sessionAuth(http.HandlerFunc(b.handleGetPortfolio)))
	mux.Handle("GET /api/v1/customer/trade-logs", b.sessionAuth(http.HandlerFunc(b.handleGetTradeLogs)))
	mux.Handle("GET /api/v1/customer/ticker", b.sessionAuth(http.HandlerFunc(b.handleTickerSSE)))
	mux.Handle("GET /api/v1/customer/zoomie-index", b.sessionAuth(http.HandlerFunc(b.handleGetZoomieIndex)))
	mux.Handle("GET /api/v1/customer/deposits", b.sessionAuth(http.HandlerFunc(b.handleGetDeposits)))
	mux.Handle("GET /api/v1/customer/api-keys", b.sessionAuth(http.HandlerFunc(b.handleGetAPIKeys)))
	mux.Handle("GET /api/v1/customer/webhooks", b.sessionAuth(http.HandlerFunc(b.handleGetWebhooks)))
	mux.Handle("GET /api/v1/customer/transparency/welfare", b.sessionAuth(http.HandlerFunc(b.handleGetESGTransparency)))
	mux.Handle("GET /api/v1/customer/streams", b.sessionAuth(http.HandlerFunc(b.handleGetStreams)))

	// Protected state-changing endpoints (Session Auth + Double-Submit CSRF)
	mux.Handle("POST /api/v1/customer/trading/orders", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateOrder))))
	mux.Handle("POST /api/v1/customer/deposits", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateDeposit))))
	mux.Handle("POST /api/v1/customer/api-keys", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateAPIKey))))
	mux.Handle("DELETE /api/v1/customer/api-keys/{id}", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleRevokeAPIKey))))
	mux.Handle("POST /api/v1/customer/webhooks", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateWebhook))))
}

func (b *CustomerBFF) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := getCSRFCookie(r)
	if token == "" {
		token = generateRandomToken(16)
		setCSRFCookie(w, token)
	}
	w.Header().Set("X-CSRF-Token", token)
	_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": token})
}

func (b *CustomerBFF) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Persona    string `json:"persona"`
		CustomerID string `json:"customer_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var custID, name, email string
	if req.Persona == "chloe" || req.CustomerID == "cust-active-chloe" {
		custID = "cust-active-chloe"
		name = "Chloe Spark"
		email = "chloe@oci.local"
	} else {
		custID = "cust-longterm-arthur"
		name = "Arthur Pendelton"
		email = "arthur@oci.local"
	}

	session := b.sessions.Create(custID, name, email)

	// Set session cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "customer_session",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-customer-session",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	csrfToken := generateRandomToken(16)
	setCSRFCookie(w, csrfToken)
	w.Header().Set("X-CSRF-Token", csrfToken)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "authenticated",
		"session":    session,
		"csrf_token": csrfToken,
	})
}

func (b *CustomerBFF) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if c, err := r.Cookie("customer_session"); err == nil {
		b.sessions.Delete(c.Value)
	}
	if c, err := r.Cookie("__Host-customer-session"); err == nil {
		b.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "customer_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-customer-session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

func (b *CustomerBFF) handleGetSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no active session"})
		return
	}
	_ = json.NewEncoder(w).Encode(session)
}

func (b *CustomerBFF) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"customer-bff"}`))
}

func (b *CustomerBFF) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	custID := ""
	if session := CustomerFromContext(r.Context()); session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}

	var resp map[string]interface{}
	if custID == "chloe" || custID == "cust-active-chloe" {
		resp = map[string]interface{}{
			"portfolio_id":       "018f3a9a-2222-7000-8000-000000000003",
			"customer_id":        "cust-active-chloe",
			"customer_name":      "Chloe Spark",
			"account_name":       "Chloe Spark High-Velocity Alpha",
			"total_balance_usd": 85200.00,
			"cash_balance_usd":  15400.00,
			"status":             "active",
			"holdings": []map[string]interface{}{
				{"symbol": "NVDA", "quantity": 150.0, "avg_price": 112.40, "current_price": 138.80, "market_value": 20820.00},
				{"symbol": "TSLA", "quantity": 120.0, "avg_price": 185.50, "current_price": 247.12, "market_value": 29654.40},
				{"symbol": "AAPL", "quantity": 85.0, "avg_price": 180.10, "current_price": 224.30, "market_value": 19065.50},
			},
			"historical_performance": []map[string]interface{}{
				{"date": "2026-01-01", "valuation": 50000.00},
				{"date": "2026-02-01", "valuation": 62400.00},
				{"date": "2026-03-01", "valuation": 74100.00},
				{"date": "2026-04-01", "valuation": 85200.00},
			},
		}
	} else {
		resp = map[string]interface{}{
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
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (b *CustomerBFF) handleGetTradeLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	orders, err := b.ciSvc.ListBrokerageOrders(r.Context(), 50, 0)

	execName := "Feline Executive"
	if b.wfSvc != nil {
		emps, err := b.wfSvc.ListEmployees(r.Context(), "feline", "active")
		if err == nil && len(emps) > 0 {
			execName = emps[0].FirstName
		}
	}

	if err != nil || len(orders) == 0 {
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
				"feline_executive": execName,
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
				"feline_executive": execName,
			},
		}
		_ = json.NewEncoder(w).Encode(mockOrders)
		return
	}
	_ = json.NewEncoder(w).Encode(orders)
}

func (b *CustomerBFF) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

	var req struct {
		PortfolioID string  `json:"portfolio_id"`
		Symbol      string  `json:"symbol"`
		Side        string  `json:"side"`
		Quantity    float64 `json:"quantity"`
		Price       float64 `json:"price"`
		OrderType   string  `json:"order_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid order payload"})
		return
	}

	if req.PortfolioID == "" {
		if session.CustomerID == "cust-active-chloe" || session.CustomerID == "chloe" {
			req.PortfolioID = "018f3a9a-2222-7000-8000-000000000003"
		} else {
			req.PortfolioID = "018f3a9a-2222-7000-8000-000000000002"
		}
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

	leaderName := "Feline Leader"
	if b.wfSvc != nil {
		emps, err := b.wfSvc.ListEmployees(r.Context(), "feline", "active")
		if err == nil && len(emps) > 0 {
			leaderName = emps[0].FirstName
		}
	}

	resp := map[string]interface{}{
		"zoomie_index_score":     88.5,
		"momentum_level":         "HIGH_VELOCITY_3AM_ZOOMIES",
		"feline_leader":          leaderName,
		"active_pounces_cnt":     14,
		"current_activity":       "zooming",
		"confidence_score":       0.965,
		"trading_signal":         "TACTICAL_BUY_BULLISH",
		"last_updated":           time.Now().Format(time.RFC3339),
		"recommended_multiplier": 1.75,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (b *CustomerBFF) handleGetDeposits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	customerID := "cust-longterm-arthur"
	if session := CustomerFromContext(r.Context()); session != nil {
		customerID = session.CustomerID
	}
	deposits := []map[string]interface{}{
		{
			"deposit_id":     "dep-018f-01",
			"customer_id":    customerID,
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
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

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
		"customer_id":    session.CustomerID,
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
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

	var req struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid api key payload"})
		return
	}

	rawSecret := fmt.Sprintf("oci_live_%s", generateRandomToken(16))
	created := map[string]interface{}{
		"key_id":      fmt.Sprintf("key-%d", time.Now().UnixNano()),
		"customer_id": session.CustomerID,
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
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}
	id := r.PathValue("id")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "revoked", "key_id": id, "customer_id": session.CustomerID})
}

func (b *CustomerBFF) handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	webhooks := []map[string]interface{}{
		{
			"subscription_id": "sub-018f-1111",
			"target_url":      "https://quant-bot.chloespark.io/api/v1/oci-callback",
			"events":          []string{"events.observation.cat_spotted.v1", "events.trade.executed.v1"},
			"status":          "active",
		},
	}
	_ = json.NewEncoder(w).Encode(webhooks)
}

func (b *CustomerBFF) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

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
		"customer_id":     session.CustomerID,
		"target_url":      req.TargetURL,
		"events":          req.Events,
		"status":          "active",
		"secret":          "whsec_" + generateRandomToken(16),
		"created_at":      time.Now().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (b *CustomerBFF) handleGetESGTransparency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	felinesList := []map[string]interface{}{}
	vhrRecordsCount := 0

	if b.wfSvc != nil {
		emps, err := b.wfSvc.ListEmployees(r.Context(), "feline", "active")
		if err == nil {
			for _, emp := range emps {
				healthStatus := "OPTIMAL_ALPHA"
				whiskerSymmetry := "100%"
				purrFreq := 28.5

				records, vhrErr := b.wfSvc.ListVHRRecords(r.Context(), emp.EmployeeID)
				if vhrErr == nil && len(records) > 0 {
					vhrRecordsCount += len(records)
					latest := records[0]
					if latest.DentalScore >= 4 {
						healthStatus = "PEAK_PERFORMER"
						whiskerSymmetry = "99.8%"
						purrFreq = 26.2
					} else {
						healthStatus = "UNDER_OBSERVATION"
						whiskerSymmetry = "95.0%"
					}
				}

				felinesList = append(felinesList, map[string]interface{}{
					"employee_id":       emp.EmployeeID,
					"name":              emp.FirstName,
					"role":              emp.RoleTitle,
					"health_status":     healthStatus,
					"whisker_symmetry": whiskerSymmetry,
					"purr_frequency_hz": purrFreq,
					"preferred_sunbeam": "Alpha Sunbeam Lounge",
					"vhr_records_cnt":   len(records),
				})
			}
		}
	}

	incidentsCnt := 0
	if b.wfSvc != nil {
		incidents, incErr := b.wfSvc.ListWorkplaceIncidents(r.Context(), "")
		if incErr == nil {
			incidentsCnt = len(incidents)
		}
	}

	welfareScore := 5.0
	if incidentsCnt > 0 {
		welfareScore -= float64(incidentsCnt) * 0.02
	}
	if welfareScore < 1.0 {
		welfareScore = 1.0
	}

	adherencePct := 100.0
	if vhrRecordsCount == 0 && len(felinesList) > 0 {
		adherencePct = 98.5
	}

	resp := map[string]interface{}{
		"overall_welfare_score":    welfareScore,
		"veterinary_adherence_pct": adherencePct,
		"nutritional_compliance":   "100% Grain-Free Prescription Salmon & Turkey",
		"avg_daily_nap_hours":      14.6,
		"perch_comfort_rating":     "5.0 / 5.0 (Ergonomic Thermal Cushioning)",
		"workplace_incidents_cnt":  incidentsCnt,
		"medical_holds_issued_ytd": 0,
		"feline_executives":        felinesList,
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
	wfRepo := workforce_repo.NewRepository(pool)
	wfSvc := workforce_svc.NewService(wfRepo)
	custBFF := NewCustomerBFF(ciSvc, wfSvc, logger)

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
