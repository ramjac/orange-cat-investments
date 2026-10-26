package customerbff

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	"github.com/orange-cat-investments/oci/pkg/session"
	"github.com/orange-cat-investments/oci/pkg/versioning"
)

// Routes builds and returns the chi.Router for CustomerBFF.
func (b *CustomerBFF) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(versioning.Middleware(versioning.DefaultConfig()))

	// Sliding-window rate limiters
	loginLimiter := func(next http.Handler) http.Handler { return next }
	orderLimiter := func(next http.Handler) http.Handler { return next }
	depositLimiter := func(next http.Handler) http.Handler { return next }

	if b.rateLimiter != nil {
		loginLimiter = cachevalkey.RateLimitMiddleware(b.rateLimiter, func(req *http.Request) string {
			return "customer_login:" + cachevalkey.ClientIP(req)
		}, 5, 15*time.Minute)

		orderLimiter = cachevalkey.RateLimitMiddleware(b.rateLimiter, func(req *http.Request) string {
			if sess := CustomerFromContext(req.Context()); sess != nil && sess.CustomerID != "" {
				return "customer_orders:" + sess.CustomerID
			}
			return "customer_orders:" + cachevalkey.ClientIP(req)
		}, 10, time.Second)

		depositLimiter = cachevalkey.RateLimitMiddleware(b.rateLimiter, func(req *http.Request) string {
			return "customer_deposits:" + cachevalkey.ClientIP(req)
		}, 5, 15*time.Minute)
	}

	authMiddleware := session.CustomerSessionAuthMiddleware(b.sessions)
	csrfMiddleware := session.CSRFProtectMiddleware("csrf_token")

	// Public routes
	r.Get("/healthz", b.handleHealthCheck)
	r.Get("/api/v1/customer/csrf", b.handleGetCSRFToken)
	r.With(loginLimiter).Post("/api/v1/customer/auth/login", b.handleLogin)
	r.Post("/api/v1/customer/auth/logout", b.handleLogout)
	r.Post("/api/v1/customer/auth/register", b.handleRegister)
	r.Get("/api/v1/customer/privacy/gdpr-certificate/{id}", b.handleGetGDPRCertificate)
	r.Get("/api/v1/customer/privacy/gdpr-certificate/{id}/download", b.handleDownloadGDPRCertificate)

	// Protected read endpoints (Session Auth)
	r.Group(func(pr chi.Router) {
		pr.Use(authMiddleware)

		pr.Get("/api/v1/customer/auth/session", b.handleGetSession)
		pr.Get("/api/v1/customer/portfolio", b.handleGetPortfolio)
		pr.Get("/api/v1/customer/trade-logs", b.handleGetTradeLogs)
		pr.Get("/api/v1/customer/ticker", b.handleTickerSSE)
		pr.Get("/api/v1/customer/zoomie-index", b.handleGetZoomieIndex)
		pr.Get("/api/v1/customer/deposits", b.handleGetDeposits)
		pr.Get("/api/v1/customer/bank-accounts", b.handleGetBankAccounts)
		pr.Get("/api/v1/customer/api-keys", b.handleGetAPIKeys)
		pr.Get("/api/v1/customer/webhooks", b.handleGetWebhooks)
		pr.Get("/api/v1/customer/transparency/welfare", b.handleGetESGTransparency)
		pr.Get("/api/v1/customer/streams", b.handleGetStreams)
		pr.Get("/api/v1/customer/support/tickets", b.handleListCustomerTickets)
		pr.Get("/api/v1/customer/support/tickets/{id}", b.handleGetCustomerTicket)
		pr.Get("/api/v1/customer/support/tickets/{id}/comments", b.handleListCustomerTicketComments)
		pr.Get("/api/v1/customer/statements", b.handleListStatements)
		pr.Get("/api/v1/customer/statements/{id}/download", b.handleDownloadStatement)
		pr.Get("/api/v1/customer/search", b.handleSearch)
	})

	// Protected state-changing endpoints (Session Auth + Double-Submit CSRF)
	r.Group(func(pw chi.Router) {
		pw.Use(authMiddleware)
		pw.Use(csrfMiddleware)

		pw.With(orderLimiter).Post("/api/v1/customer/trading/orders", b.handleCreateOrder)
		pw.With(depositLimiter).Post("/api/v1/customer/deposits", b.handleCreateDeposit)
		pw.Post("/api/v1/customer/api-keys", b.handleCreateAPIKey)
		pw.Delete("/api/v1/customer/api-keys/{id}", b.handleRevokeAPIKey)
		pw.Post("/api/v1/customer/webhooks", b.handleCreateWebhook)
		pw.Post("/api/v1/customer/support/tickets", b.handleCreateCustomerTicket)
		pw.Post("/api/v1/customer/support/tickets/{id}/comments", b.handleAddCustomerTicketComment)
		pw.Post("/api/v1/customer/account/close", b.handleCloseAccount)
		pw.Post("/api/v1/customer/privacy/gdpr-erasure", b.handleGDPRErasure)
		pw.Post("/api/v1/customer/statements/generate", b.handleGenerateStatement)
	})

	return r
}

// RegisterRoutes registers the CustomerBFF routes onto an http.ServeMux for backwards compatibility.
func (b *CustomerBFF) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/", b.Routes())
}
