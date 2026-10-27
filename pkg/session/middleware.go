package session

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)

type CustomerSession = sessionvalkey.CustomerSession
type CustomerSessionStore = sessionvalkey.CustomerSessionStore
type EmployeeSession = sessionvalkey.EmployeeSession
type EmployeeSessionStore = sessionvalkey.EmployeeSessionStore

type contextKey string

const (
	customerContextKey contextKey = "customer_session"
	employeeContextKey contextKey = "employee_session"
)

// CustomerFromContext extracts the authenticated CustomerSession from context.
func CustomerFromContext(ctx context.Context) *CustomerSession {
	if s, ok := ctx.Value(customerContextKey).(*CustomerSession); ok {
		return s
	}
	return nil
}

// WithCustomerSession injects a CustomerSession into context.
func WithCustomerSession(ctx context.Context, session *CustomerSession) context.Context {
	return context.WithValue(ctx, customerContextKey, session)
}

// EmployeeFromContext extracts the authenticated EmployeeSession from context.
func EmployeeFromContext(ctx context.Context) *EmployeeSession {
	if s, ok := ctx.Value(employeeContextKey).(*EmployeeSession); ok {
		return s
	}
	return nil
}

// WithEmployeeSession injects an EmployeeSession into context.
func WithEmployeeSession(ctx context.Context, session *EmployeeSession) context.Context {
	return context.WithValue(ctx, employeeContextKey, session)
}

// GenerateRandomToken generates a hex-encoded cryptographic random token of n bytes.
func GenerateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GetCSRFCookie reads the CSRF cookie by checking primaryName and __Host- prefixed fallback.
func GetCSRFCookie(r *http.Request, primaryName string) string {
	if primaryName == "" {
		primaryName = "csrf_token"
	}
	if c, err := r.Cookie(primaryName); err == nil && c.Value != "" {
		return c.Value
	}
	hostName := "__Host-" + strings.ReplaceAll(primaryName, "_", "-")
	if c, err := r.Cookie(hostName); err == nil && c.Value != "" {
		return c.Value
	}
	// Check alternate standard cookie names for compatibility
	for _, alt := range []string{"csrf_token", "__Host-csrf-token", "employee_csrf_token", "__Host-employee-csrf-token"} {
		if c, err := r.Cookie(alt); err == nil && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

// SetCSRFCookie sets the non-HttpOnly double-submit CSRF cookie.
func SetCSRFCookie(w http.ResponseWriter, cookieName, token string) {
	if cookieName == "" {
		cookieName = "csrf_token"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Readable by frontend JS for X-CSRF-Token double-submit header
		SameSite: http.SameSiteLaxMode,
	})
}

// CSRFTokenHandler returns an HTTP handler that issues or echoes the current CSRF token.
func CSRFTokenHandler(cookieName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := GetCSRFCookie(r, cookieName)
		if token == "" {
			var err error
			token, err = GenerateRandomToken(16)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate csrf token"})
				return
			}
			SetCSRFCookie(w, cookieName, token)
		}
		w.Header().Set("X-CSRF-Token", token)
		_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": token})
	}
}

// CSRFProtectMiddleware validates the double-submit X-CSRF-Token header against the CSRF cookie.
func CSRFProtectMiddleware(cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
				headerToken := r.Header.Get("X-CSRF-Token")
				cookieToken := GetCSRFCookie(r, cookieName)
				if headerToken == "" || cookieToken == "" || headerToken != cookieToken {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "forbidden: invalid or missing CSRF token",
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CustomerSessionAuthMiddleware validates customer session cookies or Bearer header.
func CustomerSessionAuthMiddleware(store CustomerSessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
			sess, ok := store.Get(token)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: invalid or expired customer session"})
				return
			}
			if GetCSRFCookie(r, "csrf_token") == "" {
				tokenStr, err := GenerateRandomToken(16)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
					return
				}
				SetCSRFCookie(w, "csrf_token", tokenStr)
			}
			ctx := WithCustomerSession(r.Context(), sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// EmployeeSessionAuthMiddleware validates employee session cookies or Bearer header.
func EmployeeSessionAuthMiddleware(store EmployeeSessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/webhooks/") || r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			var token string
			if c, err := r.Cookie("__Host-employee-session"); err == nil && c.Value != "" {
				token = c.Value
			} else if c, err := r.Cookie("employee_session"); err == nil && c.Value != "" {
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
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized: valid employee session required",
				})
				return
			}
			sess, ok := store.Get(token)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized: invalid or expired employee session",
				})
				return
			}
			if GetCSRFCookie(r, "employee_csrf_token") == "" {
				if csrfToken, err := GenerateRandomToken(16); err == nil {
					SetCSRFCookie(w, "employee_csrf_token", csrfToken)
				}
			}
			ctx := WithEmployeeSession(r.Context(), sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoleMiddleware enforces employee role-based access control (RBAC).
func RequireRoleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	allowedMap := make(map[string]bool)
	for _, r := range allowedRoles {
		allowedMap[strings.ToLower(strings.TrimSpace(r))] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := EmployeeFromContext(r.Context())
			if sess == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized: employee session context required for role verification",
				})
				return
			}

			userRole := strings.ToLower(strings.TrimSpace(sess.Role))
			if userRole == "admin" || allowedMap["*"] || allowedMap[userRole] {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":          "forbidden: insufficient permissions for employee role",
				"user_role":      sess.Role,
				"required_roles": allowedRoles,
			})
		})
	}
}

// InternalServiceAuthMiddleware authenticates internal service callers via X-Internal-Service-Key or Bearer token.
func InternalServiceAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Internal-Service-Key")
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			expectedKey := os.Getenv("INTERNAL_SERVICE_KEY")
			if expectedKey == "" {
				expectedKey = "oci-internal-service-secret"
			}

			if token != expectedKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized: valid internal service key required",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
