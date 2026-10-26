package employeebff

import (
	"net/http"
	"strings"
	"time"

	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
)

// RegisterProtectedOpsRoutes registers all IT helpdesk, Pebble alert, and ticket operations
// endpoints behind session authentication, role-based access control (RBAC), and CSRF protection.
func RegisterProtectedOpsRoutes(mux *http.ServeMux, opsH *opshandler.Handler, sessionStore SessionStore, limiters ...*cachevalkey.RateLimiter) {
	sessionAuth := SessionAuthMiddleware(sessionStore)
	csrfProtect := CSRFProtectMiddleware

	var rateLimiter *cachevalkey.RateLimiter
	if len(limiters) > 0 && limiters[0] != nil {
		rateLimiter = limiters[0]
	}

	pebbleAckLimiter := func(h http.Handler) http.Handler { return h }
	if rateLimiter != nil {
		pebbleAckLimiter = cachevalkey.RateLimitMiddleware(rateLimiter, func(r *http.Request) string {
			if sess := EmployeeFromContext(r.Context()); sess != nil && sess.EmployeeID != "" {
				return "pebble_ack:" + sess.EmployeeID
			}
			return "pebble_ack:" + cachevalkey.ClientIP(r)
		}, 10, time.Minute)
	}

	// Role access definitions
	opsRole := RequireRoleMiddleware("ops", "admin", "engineering", "facilities", "staff")
	adminOpsRole := RequireRoleMiddleware("ops", "admin", "engineering")
	anyEmployeeRole := RequireRoleMiddleware("ops", "admin", "engineering", "hr", "facilities", "veterinary", "investment", "staff")

	// Helper to register both versioned (/api/v1/...) and unversioned aliases
	handleProtected := func(method, path string, handler http.Handler) {
		mux.Handle(method+" "+path, handler)
		if strings.HasPrefix(path, "/api/v1/") {
			alias := strings.TrimPrefix(path, "/api/v1")
			mux.Handle(method+" "+alias, handler)
		}
	}

	// 1. Pebble Watch Endpoints
	handleProtected("GET", "/api/v1/ops/pebble/alerts", sessionAuth(opsRole(http.HandlerFunc(opsH.GetPebbleAlerts))))
	handleProtected("POST", "/api/v1/ops/pebble/ack", sessionAuth(csrfProtect(opsRole(pebbleAckLimiter(http.HandlerFunc(opsH.SubmitPebbleAck))))))

	// 2. IT Helpdesk Tickets
	handleProtected("GET", "/api/v1/ops/tickets", sessionAuth(anyEmployeeRole(http.HandlerFunc(opsH.ListTickets))))
	handleProtected("POST", "/api/v1/ops/tickets", sessionAuth(csrfProtect(anyEmployeeRole(http.HandlerFunc(opsH.CreateTicket)))))
	handleProtected("GET", "/api/v1/ops/tickets/{id}", sessionAuth(anyEmployeeRole(http.HandlerFunc(opsH.GetTicket))))
	handleProtected("PUT", "/api/v1/ops/tickets/{id}", sessionAuth(csrfProtect(anyEmployeeRole(http.HandlerFunc(opsH.UpdateTicket)))))
	handleProtected("GET", "/api/v1/ops/tickets/{id}/comments", sessionAuth(anyEmployeeRole(http.HandlerFunc(opsH.ListTicketComments))))
	handleProtected("POST", "/api/v1/ops/tickets/{id}/comments", sessionAuth(csrfProtect(anyEmployeeRole(http.HandlerFunc(opsH.AddTicketComment)))))

	// 3. Push Notifications
	handleProtected("POST", "/api/v1/ops/notifications/push", sessionAuth(csrfProtect(adminOpsRole(http.HandlerFunc(opsH.SendPushNotification)))))
	handleProtected("POST", "/api/v1/ops/notifications", sessionAuth(csrfProtect(adminOpsRole(http.HandlerFunc(opsH.SendPushNotification)))))

	// 4. Customer Support Tickets (Internal Helpdesk Staff Intake)
	handleProtected("GET", "/api/v1/customer/support/tickets", sessionAuth(anyEmployeeRole(http.HandlerFunc(opsH.ListCustomerTickets))))
	handleProtected("POST", "/api/v1/customer/support/tickets", sessionAuth(csrfProtect(anyEmployeeRole(http.HandlerFunc(opsH.CreateCustomerTicket)))))
	handleProtected("GET", "/api/v1/customer/support/tickets/{id}", sessionAuth(anyEmployeeRole(http.HandlerFunc(opsH.GetCustomerTicket))))
	handleProtected("GET", "/api/v1/customer/support/tickets/{id}/comments", sessionAuth(anyEmployeeRole(http.HandlerFunc(opsH.ListCustomerTicketComments))))
	handleProtected("POST", "/api/v1/customer/support/tickets/{id}/comments", sessionAuth(csrfProtect(anyEmployeeRole(http.HandlerFunc(opsH.AddCustomerTicketComment)))))
}
