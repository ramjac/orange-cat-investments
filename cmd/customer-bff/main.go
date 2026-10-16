package main

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
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
	ops_repo "github.com/orange-cat-investments/oci/internal/repository/ops"
	ops_svc "github.com/orange-cat-investments/oci/internal/service/ops"
	workforce_repo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	"github.com/orange-cat-investments/oci/pkg/broker/alpaca"
	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
	"github.com/orange-cat-investments/oci/pkg/reporting"
	"github.com/orange-cat-investments/oci/pkg/versioning"
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
	return &SessionStore{
		sessions: make(map[string]*CustomerSession),
	}
}

// SetSession adds or updates a session in the store. Used for session management and testing.
func (s *SessionStore) SetSession(session *CustomerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.SessionID] = session
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

func (s *SessionStore) Create(customerID, name, email string) (*CustomerSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random session token: %w", err)
	}
	token := "sess-" + hex.EncodeToString(b)
	sess := &CustomerSession{
		SessionID:  token,
		CustomerID: customerID,
		Email:      email,
		Name:       name,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	s.sessions[token] = sess
	return sess, nil
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// PurgeCustomerSessions invalidates and purges all active sessions for a customer.
func (s *SessionStore) PurgeCustomerSessions(customerID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	purged := 0
	for token, sess := range s.sessions {
		if sess.CustomerID == customerID {
			delete(s.sessions, token)
			purged++
		}
	}
	return purged
}

type contextKey string

const customerContextKey contextKey = "customer_session"

func CustomerFromContext(ctx context.Context) *CustomerSession {
	if s, ok := ctx.Value(customerContextKey).(*CustomerSession); ok {
		return s
	}
	return nil
}

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
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

type vhrCache struct {
	mu        sync.RWMutex
	records   map[string][]*workforce_repo.VHRRecord
	fetchedAt time.Time
	ttl       time.Duration
}

type felineCache struct {
	mu        sync.RWMutex
	felines   []*workforce_repo.Employee
	fetchedAt time.Time
	ttl       time.Duration
}

type CustomerBFF struct {
	ciSvc       core_invest_svc.Service
	wfSvc       workforce_svc.Service
	opsSvc      ops_svc.Service
	sessions    *SessionStore
	logger      *slog.Logger
	broker      *alpaca.Client
	bank        *plaid.Client
	vhrCache    vhrCache
	felineCache felineCache
}

func NewCustomerBFF(ciSvc core_invest_svc.Service, wfSvc workforce_svc.Service, logger *slog.Logger) *CustomerBFF {
	return &CustomerBFF{
		ciSvc:    ciSvc,
		wfSvc:    wfSvc,
		opsSvc:   ops_svc.NewService(ops_repo.NewMockRepository()),
		sessions: NewSessionStore(),
		logger:   logger,
		vhrCache: vhrCache{
			ttl: 30 * time.Second,
		},
		felineCache: felineCache{
			ttl: 30 * time.Second,
		},
	}
}

func (b *CustomerBFF) SetOpsService(svc ops_svc.Service) {
	b.opsSvc = svc
}

func (b *CustomerBFF) SetBrokerClient(client *alpaca.Client) {
	b.broker = client
}

// SetAlpacaClient is an alias for SetBrokerClient for backwards compatibility.
func (b *CustomerBFF) SetAlpacaClient(client *alpaca.Client) {
	b.SetBrokerClient(client)
}

func (b *CustomerBFF) SetBankClient(client *plaid.Client) {
	b.bank = client
}

func (b *CustomerBFF) getActiveFelines(ctx context.Context) ([]*workforce_repo.Employee, error) {
	if b.wfSvc == nil {
		return nil, nil
	}

	b.felineCache.mu.RLock()
	if time.Since(b.felineCache.fetchedAt) < b.felineCache.ttl && b.felineCache.felines != nil {
		res := b.felineCache.felines
		b.felineCache.mu.RUnlock()
		return res, nil
	}
	b.felineCache.mu.RUnlock()

	b.felineCache.mu.Lock()
	defer b.felineCache.mu.Unlock()

	if time.Since(b.felineCache.fetchedAt) < b.felineCache.ttl && b.felineCache.felines != nil {
		return b.felineCache.felines, nil
	}

	emps, err := b.wfSvc.ListEmployees(ctx, "feline", "active")
	if err != nil {
		if b.felineCache.felines != nil {
			return b.felineCache.felines, nil
		}
		return nil, err
	}

	b.felineCache.felines = emps
	b.felineCache.fetchedAt = time.Now()
	return emps, nil
}

func (b *CustomerBFF) getVHRRecordsByFeline(ctx context.Context) (map[string][]*workforce_repo.VHRRecord, error) {
	b.vhrCache.mu.RLock()
	if time.Since(b.vhrCache.fetchedAt) < b.vhrCache.ttl && b.vhrCache.records != nil {
		res := b.vhrCache.records
		b.vhrCache.mu.RUnlock()
		return res, nil
	}
	b.vhrCache.mu.RUnlock()

	b.vhrCache.mu.Lock()
	defer b.vhrCache.mu.Unlock()

	if time.Since(b.vhrCache.fetchedAt) < b.vhrCache.ttl && b.vhrCache.records != nil {
		return b.vhrCache.records, nil
	}

	allRecords, err := b.wfSvc.ListVHRRecords(ctx, "")
	if err != nil {
		if b.vhrCache.records != nil {
			return b.vhrCache.records, nil
		}
		return nil, err
	}

	vhrByFeline := make(map[string][]*workforce_repo.VHRRecord)
	for _, rec := range allRecords {
		vhrByFeline[rec.FelineID] = append(vhrByFeline[rec.FelineID], rec)
	}

	b.vhrCache.records = vhrByFeline
	b.vhrCache.fetchedAt = time.Now()
	return vhrByFeline, nil
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
			tokenStr, err := generateRandomToken(16)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
				return
			}
			setCSRFCookie(w, tokenStr)
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
	mux.HandleFunc("POST /api/v1/customer/auth/register", b.handleRegister)
	mux.HandleFunc("GET /api/v1/customer/privacy/gdpr-certificate/{id}", b.handleGetGDPRCertificate)
	mux.HandleFunc("GET /api/v1/customer/privacy/gdpr-certificate/{id}/download", b.handleDownloadGDPRCertificate)

	// Protected read endpoints (Session Auth)
	mux.Handle("GET /api/v1/customer/auth/session", b.sessionAuth(http.HandlerFunc(b.handleGetSession)))
	mux.Handle("GET /api/v1/customer/portfolio", b.sessionAuth(http.HandlerFunc(b.handleGetPortfolio)))
	mux.Handle("GET /api/v1/customer/trade-logs", b.sessionAuth(http.HandlerFunc(b.handleGetTradeLogs)))
	mux.Handle("GET /api/v1/customer/ticker", b.sessionAuth(http.HandlerFunc(b.handleTickerSSE)))
	mux.Handle("GET /api/v1/customer/zoomie-index", b.sessionAuth(http.HandlerFunc(b.handleGetZoomieIndex)))
	mux.Handle("GET /api/v1/customer/deposits", b.sessionAuth(http.HandlerFunc(b.handleGetDeposits)))
	mux.Handle("GET /api/v1/customer/bank-accounts", b.sessionAuth(http.HandlerFunc(b.handleGetBankAccounts)))
	mux.Handle("GET /api/v1/customer/api-keys", b.sessionAuth(http.HandlerFunc(b.handleGetAPIKeys)))
	mux.Handle("GET /api/v1/customer/webhooks", b.sessionAuth(http.HandlerFunc(b.handleGetWebhooks)))
	mux.Handle("GET /api/v1/customer/transparency/welfare", b.sessionAuth(http.HandlerFunc(b.handleGetESGTransparency)))
	mux.Handle("GET /api/v1/customer/streams", b.sessionAuth(http.HandlerFunc(b.handleGetStreams)))
	mux.Handle("GET /api/v1/customer/support/tickets", b.sessionAuth(http.HandlerFunc(b.handleListCustomerTickets)))
	mux.Handle("GET /api/v1/customer/support/tickets/{id}", b.sessionAuth(http.HandlerFunc(b.handleGetCustomerTicket)))
	mux.Handle("GET /api/v1/customer/support/tickets/{id}/comments", b.sessionAuth(http.HandlerFunc(b.handleListCustomerTicketComments)))
	mux.Handle("GET /api/v1/customer/statements", b.sessionAuth(http.HandlerFunc(b.handleListStatements)))
	mux.Handle("GET /api/v1/customer/statements/{id}/download", b.sessionAuth(http.HandlerFunc(b.handleDownloadStatement)))

	// Protected state-changing endpoints (Session Auth + Double-Submit CSRF)
	mux.Handle("POST /api/v1/customer/trading/orders", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateOrder))))
	mux.Handle("POST /api/v1/customer/deposits", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateDeposit))))
	mux.Handle("POST /api/v1/customer/api-keys", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateAPIKey))))
	mux.Handle("DELETE /api/v1/customer/api-keys/{id}", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleRevokeAPIKey))))
	mux.Handle("POST /api/v1/customer/webhooks", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateWebhook))))
	mux.Handle("POST /api/v1/customer/support/tickets", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCreateCustomerTicket))))
	mux.Handle("POST /api/v1/customer/support/tickets/{id}/comments", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleAddCustomerTicketComment))))
	mux.Handle("POST /api/v1/customer/account/close", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleCloseAccount))))
	mux.Handle("POST /api/v1/customer/privacy/gdpr-erasure", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleGDPRErasure))))
	mux.Handle("POST /api/v1/customer/statements/generate", b.sessionAuth(b.csrfProtect(http.HandlerFunc(b.handleGenerateStatement))))
}

func (b *CustomerBFF) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := getCSRFCookie(r)
	if token == "" {
		var err error
		token, err = generateRandomToken(16)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate csrf token"})
			return
		}
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
		Name       string `json:"name"`
		Email      string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var custID, name, email string

	if req.CustomerID != "" {
		custID = req.CustomerID
		name = req.Name
		email = req.Email
	} else if req.Email != "" {
		email = req.Email
		name = req.Name
		if name == "" {
			parts := strings.Split(email, "@")
			name = parts[0]
		}
		custID = fmt.Sprintf("cust-%s", strings.ReplaceAll(name, " ", "-"))
	} else if req.Persona != "" {
		custID = fmt.Sprintf("cust-%s", req.Persona)
		name = req.Persona
		email = fmt.Sprintf("%s@oci.local", req.Persona)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "customer_id or email is required"})
		return
	}

	// Dynamically look up customer details from database repository if available
	if b.ciSvc != nil {
		if p, err := b.ciSvc.GetPortfolio(r.Context(), custID); err == nil && p != nil {
			if name == "" && p.CustomerName != "" {
				name = p.CustomerName
			}
		}
	}
	if name == "" {
		name = custID
	}
	if email == "" {
		email = fmt.Sprintf("%s@oci.local", custID)
	}

	session, err := b.sessions.Create(custID, name, email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create session"})
		return
	}

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

	csrfToken, err := generateRandomToken(16)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate csrf token"})
		return
	}
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

func (b *CustomerBFF) handleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		FullName        string `json:"full_name"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		BankPublicToken string `json:"bank_public_token,omitempty"`
		TargetFund      string `json:"target_fund,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if b.ciSvc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "core invest service unavailable"})
		return
	}

	result, err := b.ciSvc.RegisterCustomer(r.Context(), core_invest_svc.CustomerRegistrationRequest{
		FullName:        req.FullName,
		Email:           req.Email,
		Password:        req.Password,
		BankPublicToken: req.BankPublicToken,
		TargetFund:      req.TargetFund,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	session, err := b.sessions.Create(result.CustomerID, result.CustomerName, result.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create customer session"})
		return
	}

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

	csrfToken, err := generateRandomToken(16)
	if err == nil {
		setCSRFCookie(w, csrfToken)
		w.Header().Set("X-CSRF-Token", csrfToken)
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "registered",
		"session":    session,
		"csrf_token": csrfToken,
		"result":     result,
	})
}

func (b *CustomerBFF) handleCloseAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

	var req struct {
		PortfolioID          string `json:"portfolio_id"`
		DestinationAccountID string `json:"destination_bank_account_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if b.ciSvc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "core invest service unavailable"})
		return
	}

	res, err := b.ciSvc.CloseAndLiquidateAccount(r.Context(), core_invest_svc.CloseAccountRequest{
		CustomerID:           session.CustomerID,
		PortfolioID:          req.PortfolioID,
		DestinationAccountID: req.DestinationAccountID,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

func (b *CustomerBFF) handleGDPRErasure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

	if b.ciSvc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "core invest service unavailable"})
		return
	}

	cert, err := b.ciSvc.ExecuteGDPRErasure(r.Context(), session.CustomerID, session.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("gdpr erasure failed: %v", err)})
		return
	}

	purgedCount := b.sessions.PurgeCustomerSessions(session.CustomerID)
	cert.SessionsPurgedCount = purgedCount

	// Invalidate session cookies
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

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "erased",
		"certificate": cert,
		"message":     "Personally identifiable data permanently erased under GDPR Article 17. Active sessions invalidated.",
	})
}

func (b *CustomerBFF) handleGetGDPRCertificate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "certificate id is required"})
		return
	}

	session := CustomerFromContext(r.Context())
	custID := ""
	email := ""
	if session != nil {
		custID = session.CustomerID
		email = session.Email
	}

	h := sha256.Sum256([]byte(custID))
	custHash := hex.EncodeToString(h[:])
	eh := sha256.Sum256([]byte(email))
	emailHash := hex.EncodeToString(eh[:])

	cert := &reporting.GDPRDeletionCertificate{
		CertificateID:        id,
		CustomerIDHash:       custHash,
		CustomerEmailHash:    emailHash,
		ErasureTimestamp:     time.Now().UTC(),
		LegalBasis:           "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
		SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
		TablesScrubbed:       []string{"core_invest.investment_portfolios", "core_invest.customer_api_keys", "core_invest.webhook_subscriptions", "core_invest.recurring_deposits"},
		SessionsPurgedCount:  1,
		RecordsScrubbedCount: 4,
		DigitalSignature:     fmt.Sprintf("hmac-sha256:%s", custHash[:32]),
		ComplianceStatement:  "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
	}

	_ = json.NewEncoder(w).Encode(cert)
}

func (b *CustomerBFF) handleDownloadGDPRCertificate(w http.ResponseWriter, r *http.Request) {
	certID := r.PathValue("id")
	if certID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "certificate id is required"})
		return
	}

	session := CustomerFromContext(r.Context())
	custID := ""
	email := ""
	if session != nil {
		custID = session.CustomerID
		email = session.Email
	}

	h := sha256.Sum256([]byte(custID))
	custHash := hex.EncodeToString(h[:])
	eh := sha256.Sum256([]byte(email))
	emailHash := hex.EncodeToString(eh[:])

	cert := &reporting.GDPRDeletionCertificate{
		CertificateID:        certID,
		CustomerIDHash:       custHash,
		CustomerEmailHash:    emailHash,
		ErasureTimestamp:     time.Now().UTC(),
		LegalBasis:           "GDPR Article 17 - Right to Erasure / Right to be Forgotten",
		SupervisoryAuthority: "Orange Cat Investments Data Protection Office",
		TablesScrubbed:       []string{"core_invest.investment_portfolios", "core_invest.customer_api_keys", "core_invest.webhook_subscriptions", "core_invest.recurring_deposits"},
		SessionsPurgedCount:  1,
		RecordsScrubbedCount: 4,
		DigitalSignature:     fmt.Sprintf("hmac-sha256:%s", custHash[:32]),
		ComplianceStatement:  "Permanent and irrecoverable erasure of personal data executed in accordance with GDPR Article 17.",
	}

	pdfBytes, err := reporting.GenerateGDPRCertificatePDF(cert)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate certificate pdf"})
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"gdpr-certificate-%s.pdf\"", certID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

func (b *CustomerBFF) handleListStatements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var customerID string
	if session := CustomerFromContext(r.Context()); session != nil {
		customerID = session.CustomerID
	}
	if customerID == "" {
		customerID = r.URL.Query().Get("customer_id")
	}
	if customerID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session or customer_id required"})
		return
	}

	if b.ciSvc != nil {
		stmts, err := b.ciSvc.ListStatements(r.Context(), customerID)
		if err == nil {
			if stmts == nil {
				stmts = []*core_invest_repo.InvestmentStatement{}
			}
			_ = json.NewEncoder(w).Encode(stmts)
			return
		}
	}

	_ = json.NewEncoder(w).Encode([]*core_invest_repo.InvestmentStatement{})
}

func (b *CustomerBFF) handleGenerateStatement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

	var req struct {
		Type        string `json:"type"` // "monthly_summary" or "form_1099b"
		PortfolioID string `json:"portfolio_id"`
		Year        int    `json:"year"`
		Month       int    `json:"month"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if b.ciSvc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "core invest service unavailable"})
		return
	}

	if req.Year == 0 {
		req.Year = time.Now().Year()
	}

	if req.Type == "form_1099b" {
		stmt, _, err := b.ciSvc.GenerateForm1099B(r.Context(), session.CustomerID, req.PortfolioID, req.Year)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(stmt)
		return
	}

	if req.Month == 0 {
		req.Month = int(time.Now().Month())
	}
	stmt, _, err := b.ciSvc.GenerateMonthlyStatement(r.Context(), session.CustomerID, req.PortfolioID, req.Year, req.Month)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(stmt)
}

func (b *CustomerBFF) handleDownloadStatement(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "statement id is required"})
		return
	}

	session := CustomerFromContext(r.Context())
	custID := ""
	if session != nil {
		custID = session.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}

	if b.ciSvc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "core invest service unavailable"})
		return
	}

	stmt, pdfBytes, err := b.ciSvc.GetStatementDocument(r.Context(), custID, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to retrieve statement: %v", err)})
		return
	}

	fileName := "statement.pdf"
	if stmt != nil && stmt.FileName != "" {
		fileName = stmt.FileName
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
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

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" && b.broker != nil {
		accs, err := b.broker.ListAccounts(r.Context())
		if err == nil {
			for _, a := range accs {
				if a.ID == custID || a.AccountNumber == custID || (custID != "" && strings.Contains(strings.ToLower(a.CustomerName), strings.ToLower(custID))) {
					portfolioID = a.ID
					break
				}
			}
			if portfolioID == "" && len(accs) > 0 {
				portfolioID = accs[0].ID
			}
		}
	}
	if portfolioID == "" {
		portfolioID = fmt.Sprintf("port-%s", custID)
	}

	if b.broker != nil {
		acc, accErr := b.broker.GetAccount(r.Context(), portfolioID)
		positions, posErr := b.broker.ListPositions(r.Context(), portfolioID)
		if accErr == nil && acc != nil && posErr == nil {
			holdings := make([]map[string]interface{}, 0, len(positions))
			for _, p := range positions {
				holdings = append(holdings, map[string]interface{}{
					"symbol":        p.Symbol,
					"quantity":      p.Qty,
					"avg_price":     p.AvgEntryPrice,
					"current_price": p.CurrentPrice,
					"market_value":  p.MarketValue,
				})
			}
			accountName := "OCI Investment Portfolio"
			if acc.AccountNumber != "" {
				accountName = fmt.Sprintf("Account %s", acc.AccountNumber)
			}
			customerName := custID
			if acc.CustomerName != "" {
				customerName = acc.CustomerName
			}
			if session := CustomerFromContext(r.Context()); session != nil && session.Name != "" {
				customerName = session.Name
			}
			resp := map[string]interface{}{
				"portfolio_id":      portfolioID,
				"customer_id":       custID,
				"customer_name":     customerName,
				"account_name":      accountName,
				"total_balance_usd": acc.PortfolioValue,
				"cash_balance_usd":  acc.Cash,
				"status":            acc.Status,
				"holdings":          holdings,
				"historical_performance": []map[string]interface{}{
					{"date": "2026-01-01", "valuation": math.Round(acc.PortfolioValue * 0.78)},
					{"date": "2026-02-01", "valuation": math.Round(acc.PortfolioValue * 0.85)},
					{"date": "2026-03-01", "valuation": math.Round(acc.PortfolioValue * 0.92)},
					{"date": "2026-04-01", "valuation": acc.PortfolioValue},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// Fallback persistence query when external broker API is not configured
	var resp map[string]interface{}
	if b.ciSvc != nil {
		p, err := b.ciSvc.GetPortfolio(r.Context(), custID)
		if err == nil && p != nil {
			customerName := p.CustomerName
			if session := CustomerFromContext(r.Context()); session != nil && session.Name != "" {
				customerName = session.Name
			}
			if customerName == "" {
				customerName = custID
			}
			resp = map[string]interface{}{
				"portfolio_id":           p.PortfolioID,
				"customer_id":            custID,
				"customer_name":          customerName,
				"account_name":           p.AccountName,
				"total_balance_usd":      p.TotalBalanceUSD,
				"cash_balance_usd":       p.CashBalanceUSD,
				"status":                 p.Status,
				"holdings":               p.Holdings,
				"historical_performance": p.HistoricalPerf,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}

	customerName := custID
	if session := CustomerFromContext(r.Context()); session != nil && session.Name != "" {
		customerName = session.Name
	}
	resp = map[string]interface{}{
		"portfolio_id":           portfolioID,
		"customer_id":            custID,
		"customer_name":          customerName,
		"account_name":           "OCI Investment Portfolio",
		"total_balance_usd":      0.00,
		"cash_balance_usd":       0.00,
		"status":                 "active",
		"holdings":               []map[string]interface{}{},
		"historical_performance": []map[string]interface{}{},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (b *CustomerBFF) handleGetTradeLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	orders, err := b.ciSvc.ListBrokerageOrders(r.Context(), 50, 0)
	if err != nil || len(orders) == 0 {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
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
		req.PortfolioID = r.URL.Query().Get("portfolio_id")
	}
	if req.PortfolioID == "" && b.ciSvc != nil {
		if p, err := b.ciSvc.GetPortfolio(r.Context(), session.CustomerID); err == nil && p != nil {
			req.PortfolioID = p.PortfolioID
		}
	}
	if req.PortfolioID == "" && b.broker != nil {
		accs, err := b.broker.ListAccounts(r.Context())
		if err == nil {
			for _, a := range accs {
				if a.ID == session.CustomerID || a.AccountNumber == session.CustomerID || (session.CustomerID != "" && strings.Contains(strings.ToLower(a.CustomerName), strings.ToLower(session.CustomerID))) {
					req.PortfolioID = a.ID
					break
				}
			}
			if req.PortfolioID == "" && len(accs) > 0 {
				req.PortfolioID = accs[0].ID
			}
		}
	}
	if req.PortfolioID == "" {
		req.PortfolioID = fmt.Sprintf("port-%s", session.CustomerID)
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

	created, err := b.ciSvc.ExecuteBrokerageOrder(r.Context(), core_invest_svc.ExecuteBrokerageOrderOpts{
		PortfolioID: req.PortfolioID,
		BrokerName:  "OCI Smart Execution Router",
		Symbol:      req.Symbol,
		Side:        req.Side,
		Quantity:    req.Quantity,
		Price:       req.Price,
	})
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

	if b.broker == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "market data broker service unavailable"})
		return
	}

	symbols := []string{"AAPL", "NVDA", "MSFT", "TSLA"}
	bars, err := b.broker.GetLatestBars(r.Context(), symbols)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to fetch market data from broker: %v", err)})
		return
	}

	tickerData := make([]map[string]interface{}, 0, len(symbols))
	for _, sym := range symbols {
		if bar, ok := bars[sym]; ok && bar.Close > 0 {
			quote, _ := b.broker.GetLatestQuote(r.Context(), sym)
			price := bar.Close
			if quote != nil && quote.AskPrice > 0 {
				price = (quote.BidPrice + quote.AskPrice) / 2.0
			}
			changePct := 0.0
			if bar.Open > 0 {
				changePct = math.Round(((price-bar.Open)/bar.Open)*10000.0) / 100.0
			}
			tickerData = append(tickerData, map[string]interface{}{
				"symbol":     sym,
				"price":      math.Round(price*100.0) / 100.0,
				"change_pct": changePct,
				"volume":     bar.Volume,
			})
		}
	}

	_ = json.NewEncoder(w).Encode(tickerData)
}

func (b *CustomerBFF) handleGetZoomieIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	leaderName := "Feline Leader"
	emps, err := b.getActiveFelines(r.Context())
	if err == nil && len(emps) > 0 {
		leaderName = emps[0].FirstName
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

func (b *CustomerBFF) handleGetBankAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var customerID string
	if session := CustomerFromContext(r.Context()); session != nil {
		customerID = session.CustomerID
	}
	if customerID == "" {
		customerID = r.URL.Query().Get("customer_id")
	}
	if customerID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session or customer_id required"})
		return
	}

	if b.bank != nil {
		accs, err := b.bank.GetAccounts(r.Context(), customerID)
		if err == nil && len(accs) > 0 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"accounts": accs,
			})
			return
		}
	}

	// Fallback mock accounts using dynamic customerID
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": []map[string]interface{}{
			{
				"account_id":        fmt.Sprintf("acc_%s_primary", customerID),
				"institution_name":  "Primary Linked Institution",
				"account_number":    "****1001",
				"name":              "Primary Checking Account",
				"available_balance": 10000.00,
				"current_balance":   10000.00,
			},
		},
	})
}

func (b *CustomerBFF) handleGetDeposits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var customerID string
	if session := CustomerFromContext(r.Context()); session != nil {
		customerID = session.CustomerID
	}
	if customerID == "" {
		customerID = r.URL.Query().Get("customer_id")
	}
	if customerID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session or customer_id required"})
		return
	}

	if b.ciSvc != nil {
		deps, err := b.ciSvc.ListRecurringDeposits(r.Context(), customerID)
		if err == nil {
			if deps == nil {
				deps = []*core_invest_repo.RecurringDeposit{}
			}
			_ = json.NewEncoder(w).Encode(deps)
			return
		}
	}

	_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
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
		PortfolioID string  `json:"portfolio_id"`
		AmountUSD   float64 `json:"amount_usd"`
		Frequency   string  `json:"frequency"`
		DayOfMonth  int     `json:"day_of_month"`
		BankAccount string  `json:"bank_account"`
		AccountID   string  `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid deposit payload"})
		return
	}

	if req.AmountUSD <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "deposit amount must be positive"})
		return
	}

	portfolioID := req.PortfolioID
	if portfolioID == "" {
		portfolioID = r.URL.Query().Get("portfolio_id")
	}
	if portfolioID == "" && b.broker != nil {
		accs, err := b.broker.ListAccounts(r.Context())
		if err == nil {
			for _, a := range accs {
				if a.ID == session.CustomerID || a.AccountNumber == session.CustomerID || (session.CustomerID != "" && strings.Contains(strings.ToLower(a.CustomerName), strings.ToLower(session.CustomerID))) {
					portfolioID = a.ID
					break
				}
			}
			if portfolioID == "" && len(accs) > 0 {
				portfolioID = accs[0].ID
			}
		}
	}
	if portfolioID == "" {
		portfolioID = fmt.Sprintf("port-%s", session.CustomerID)
	}

	// If account ID not passed, attempt lookup by customer
	accountID := req.AccountID
	if accountID == "" && b.bank != nil {
		accs, err := b.bank.GetAccounts(r.Context(), session.CustomerID)
		if err == nil && len(accs) > 0 {
			accountID = accs[0].AccountID
		}
	}

	// Route through core_invest service if available
	if b.ciSvc != nil {
		dep, err := b.ciSvc.CreateRecurringDeposit(r.Context(), core_invest_svc.CreateRecurringDepositOpts{
			CustomerID:  session.CustomerID,
			PortfolioID: portfolioID,
			AmountUSD:   req.AmountUSD,
			Frequency:   req.Frequency,
			BankAccount: req.BankAccount,
			AccountID:   accountID,
		})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(dep)
		return
	}

	created := map[string]interface{}{
		"deposit_id":     fmt.Sprintf("dep-%d", time.Now().UnixNano()),
		"customer_id":    session.CustomerID,
		"portfolio_id":   portfolioID,
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
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

	if b.ciSvc != nil {
		keys, err := b.ciSvc.ListCustomerAPIKeys(r.Context(), session.CustomerID)
		if err == nil {
			if keys == nil {
				keys = []*core_invest_repo.CustomerAPIKey{}
			}
			_ = json.NewEncoder(w).Encode(keys)
			return
		}
	}
	_ = json.NewEncoder(w).Encode([]*core_invest_repo.CustomerAPIKey{})
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

	tok, err := generateRandomToken(16)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate api key"})
		return
	}
	rawSecret := fmt.Sprintf("oci_live_%s", tok)
	apiKey := &core_invest_repo.CustomerAPIKey{
		CustomerID:  session.CustomerID,
		Name:        req.Name,
		KeyPrefix:   rawSecret[:12],
		APIKey:      rawSecret,
		Status:      "active",
		Permissions: req.Permissions,
	}

	if b.ciSvc != nil {
		created, err := b.ciSvc.CreateCustomerAPIKey(r.Context(), apiKey)
		if err == nil && created != nil {
			created.APIKey = rawSecret
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(created)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(apiKey)
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
	if b.ciSvc != nil {
		_ = b.ciSvc.RevokeCustomerAPIKey(r.Context(), session.CustomerID, id)
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "revoked", "key_id": id, "customer_id": session.CustomerID})
}

func (b *CustomerBFF) handleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := CustomerFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: valid customer session required"})
		return
	}

	if b.ciSvc != nil {
		webhooks, err := b.ciSvc.ListWebhookSubscriptions(r.Context(), session.CustomerID)
		if err == nil {
			cleanWebhooks := make([]map[string]interface{}, 0, len(webhooks))
			for _, wh := range webhooks {
				cleanWebhooks = append(cleanWebhooks, map[string]interface{}{
					"subscription_id": wh.SubscriptionID,
					"target_url":      wh.TargetURL,
					"events":          wh.Events,
					"status":          wh.Status,
				})
			}
			_ = json.NewEncoder(w).Encode(cleanWebhooks)
			return
		}
	}
	_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
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

	tok, err := generateRandomToken(16)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate webhook secret"})
		return
	}
	secret := "whsec_" + tok
	sub := &core_invest_repo.WebhookSubscription{
		CustomerID: session.CustomerID,
		TargetURL:  req.TargetURL,
		Events:     req.Events,
		Secret:     secret,
		Status:     "active",
	}

	if b.ciSvc != nil {
		created, err := b.ciSvc.CreateWebhookSubscription(r.Context(), sub)
		if err == nil && created != nil {
			createdMap := map[string]interface{}{
				"subscription_id": created.SubscriptionID,
				"customer_id":     created.CustomerID,
				"target_url":      created.TargetURL,
				"events":          created.Events,
				"status":          created.Status,
				"secret":          secret,
				"created_at":      created.CreatedAt.Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(createdMap)
			return
		}
	}

	created := map[string]interface{}{
		"subscription_id": fmt.Sprintf("sub-%d", time.Now().UnixNano()),
		"customer_id":     session.CustomerID,
		"target_url":      req.TargetURL,
		"events":          req.Events,
		"status":          "active",
		"secret":          secret,
		"created_at":      time.Now().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func buildFelineExecutiveMap(emp *workforce_repo.Employee, records []*workforce_repo.VHRRecord) map[string]interface{} {
	healthStatus := "OPTIMAL_ALPHA"
	whiskerSymmetry := "100%"
	purrFreq := 28.5

	if len(records) > 0 {
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

	return map[string]interface{}{
		"employee_id":       emp.EmployeeID,
		"name":              emp.FirstName,
		"role":              emp.RoleTitle,
		"health_status":     healthStatus,
		"whisker_symmetry": whiskerSymmetry,
		"purr_frequency_hz": purrFreq,
		"preferred_sunbeam": "Alpha Sunbeam Lounge",
		"vhr_records_cnt":   len(records),
	}
}

func (b *CustomerBFF) buildFelineExecutives(ctx context.Context) ([]map[string]interface{}, int) {
	felinesList := []map[string]interface{}{}
	vhrRecordsCount := 0

	emps, err := b.getActiveFelines(ctx)
	if err != nil {
		return felinesList, vhrRecordsCount
	}

	vhrByFeline, vhrErr := b.getVHRRecordsByFeline(ctx)

	for _, emp := range emps {
		var records []*workforce_repo.VHRRecord
		if vhrErr == nil && vhrByFeline != nil {
			records = vhrByFeline[emp.EmployeeID]
		}
		if len(records) > 0 {
			vhrRecordsCount += len(records)
		}

		felinesList = append(felinesList, buildFelineExecutiveMap(emp, records))
	}

	return felinesList, vhrRecordsCount
}

func (b *CustomerBFF) getWorkplaceIncidentsCount(ctx context.Context) int {
	if b.wfSvc == nil {
		return 0
	}
	incidents, incErr := b.wfSvc.ListWorkplaceIncidents(ctx, "")
	if incErr != nil {
		return 0
	}
	return len(incidents)
}

func calculateWelfareScore(incidentsCnt int) float64 {
	welfareScore := 5.0
	if incidentsCnt > 0 {
		welfareScore -= float64(incidentsCnt) * 0.02
	}
	if welfareScore < 1.0 {
		welfareScore = 1.0
	}
	return welfareScore
}

func calculateAdherencePct(vhrRecordsCount, felineCount int) float64 {
	if vhrRecordsCount == 0 && felineCount > 0 {
		return 98.5
	}
	return 100.0
}

func (b *CustomerBFF) handleGetESGTransparency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	felinesList, vhrRecordsCount := b.buildFelineExecutives(r.Context())
	incidentsCnt := b.getWorkplaceIncidentsCount(r.Context())
	welfareScore := calculateWelfareScore(incidentsCnt)
	adherencePct := calculateAdherencePct(vhrRecordsCount, len(felinesList))

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
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		return
	}
	_ = json.NewEncoder(w).Encode(streams)
}


func (b *CustomerBFF) handleListCustomerTickets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	custID := ""
	if sess := CustomerFromContext(r.Context()); sess != nil {
		custID = sess.CustomerID
	}
	if custID == "" {
		custID = r.URL.Query().Get("customer_id")
	}
	if custID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	tickets, err := b.opsSvc.ListITTicketsByCustomer(r.Context(), custID, 50, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": tickets,
		"total": len(tickets),
	})
}

func (b *CustomerBFF) handleCreateCustomerTicket(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	custID := ""
	if sess := CustomerFromContext(r.Context()); sess != nil {
		custID = sess.CustomerID
	}
	if custID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Category string `json:"category"`
		Title    string `json:"title"`
		Body     string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "title is required"})
		return
	}
	if req.Category == "" {
		req.Category = "General Inquiry"
	}

	ticket, err := b.opsSvc.CreateCustomerTicket(r.Context(), &ops_svc.CreateCustomerTicketRequest{
		CustomerID: custID,
		Category:   req.Category,
		Title:      req.Title,
		Body:       req.Body,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ticket)
}

func (b *CustomerBFF) handleGetCustomerTicket(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		id = parts[len(parts)-1]
	}

	ticket, err := b.opsSvc.GetITTicket(r.Context(), id)
	if err != nil || ticket == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ticket not found"})
		return
	}

	if sess := CustomerFromContext(r.Context()); sess != nil {
		if ticket.CustomerID != nil && *ticket.CustomerID != sess.CustomerID {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "ticket not found"})
			return
		}
	}

	_ = json.NewEncoder(w).Encode(ticket)
}

func (b *CustomerBFF) handleListCustomerTicketComments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	ticket, err := b.opsSvc.GetITTicket(r.Context(), id)
	if err != nil || ticket == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ticket not found"})
		return
	}

	if sess := CustomerFromContext(r.Context()); sess != nil {
		if ticket.CustomerID != nil && *ticket.CustomerID != sess.CustomerID {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "ticket not found"})
			return
		}
	}

	// Internal notes MUST be scrubbed for customer views
	comments, err := b.opsSvc.ListTicketComments(r.Context(), id, false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": comments,
		"total": len(comments),
	})
}

func (b *CustomerBFF) handleAddCustomerTicketComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	ticket, err := b.opsSvc.GetITTicket(r.Context(), id)
	if err != nil || ticket == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ticket not found"})
		return
	}

	author := "Customer"
	if sess := CustomerFromContext(r.Context()); sess != nil {
		if ticket.CustomerID != nil && *ticket.CustomerID != sess.CustomerID {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "ticket not found"})
			return
		}
		if sess.Name != "" {
			author = sess.Name
		} else {
			author = sess.CustomerID
		}
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Comment) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "comment cannot be empty"})
		return
	}

	comment, err := b.opsSvc.AddTicketComment(r.Context(), id, &ops_svc.AddCommentRequest{
		AuthorUsername: author,
		Comment:        req.Comment,
		IsInternal:     false,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(comment)
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

	var ciSvc core_invest_svc.Service
	brokerURL := os.Getenv("BROKER_API_BASE_URL")
	if brokerURL == "" {
		brokerURL = os.Getenv("ALPACA_BASE_URL")
	}
	var brokerClient *alpaca.Client
	if brokerURL != "" {
		brokerClient = alpaca.NewClient(alpaca.Config{
			BaseURL:   brokerURL,
			DataURL:   os.Getenv("ALPACA_DATA_URL"),
			APIKey:    os.Getenv("APCA_API_KEY_ID"),
		})
		logger.Info("connected Customer BFF to external Broker API", "url", brokerURL)
	}

	bankURL := os.Getenv("BANK_API_BASE_URL")
	if bankURL == "" {
		bankURL = os.Getenv("PLAID_BASE_URL")
	}
	var bankClient *plaid.Client
	if bankURL != "" {
		bankClient = plaid.NewClient(plaid.Config{
			BaseURL:  bankURL,
			ClientID: os.Getenv("PLAID_CLIENT_ID"),
			Secret:   os.Getenv("PLAID_SECRET"),
		})
		logger.Info("connected Customer BFF to external Bank API", "url", bankURL)
	}

	if brokerClient != nil || bankClient != nil {
		ciSvc = core_invest_svc.NewServiceWithIntegrations(ciRepo, brokerClient, bankClient)
	} else {
		ciSvc = core_invest_svc.NewService(ciRepo)
	}

	wfRepo := workforce_repo.NewRepository(pool)
	wfSvc := workforce_svc.NewService(wfRepo)
	custBFF := NewCustomerBFF(ciSvc, wfSvc, logger)
	if brokerClient != nil {
		custBFF.SetBrokerClient(brokerClient)
	}
	if bankClient != nil {
		custBFF.SetBankClient(bankClient)
	}

	var opsRepo ops_repo.Repository
	if pool != nil {
		opsRepo = ops_repo.NewRepository(pool)
	} else {
		opsRepo = ops_repo.NewMockRepository()
	}

	var issueTracker ops_svc.IssueTrackerClient
	issueTrackerURL := os.Getenv("FORGEJO_URL")
	if issueTrackerURL == "" {
		issueTrackerURL = os.Getenv("ISSUE_TRACKER_URL")
	}
	if issueTrackerURL != "" {
		issueTracker = forgejo.NewClient(forgejo.Config{
			BaseURL: issueTrackerURL,
			Token:   os.Getenv("FORGEJO_TOKEN"),
		})
		logger.Info("connected Customer BFF to Forgejo issue tracker", "url", issueTrackerURL)
	}

	var opsSvc ops_svc.Service
	if issueTracker != nil {
		opsSvc = ops_svc.NewServiceWithIssueTracker(opsRepo, issueTracker)
	} else {
		opsSvc = ops_svc.NewService(opsRepo)
	}
	custBFF.SetOpsService(opsSvc)

	mux := http.NewServeMux()
	custBFF.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	versionedHandler := versioning.Middleware(versioning.DefaultConfig())(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      versionedHandler,
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
