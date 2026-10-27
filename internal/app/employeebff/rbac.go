package employeebff

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/orange-cat-investments/oci/pkg/session"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)

type EmployeeSession = sessionvalkey.EmployeeSession
type SessionStore = sessionvalkey.EmployeeSessionStore

func NewSessionStore() SessionStore {
	return sessionvalkey.NewMemoryEmployeeSessionStore()
}

func EmployeeFromContext(ctx context.Context) *EmployeeSession {
	return session.EmployeeFromContext(ctx)
}

func WithEmployeeSession(ctx context.Context, sess *EmployeeSession) context.Context {
	return session.WithEmployeeSession(ctx, sess)
}

func GenerateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func GetCSRFCookie(r *http.Request) string {
	if c, err := r.Cookie("employee_csrf_token"); err == nil && c.Value != "" {
		return c.Value
	}
	if c, err := r.Cookie("__Host-employee-csrf-token"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

func SetCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "employee_csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Accessible by frontend JS for double-submit header
		SameSite: http.SameSiteLaxMode,
	})
}

// SessionAuthMiddleware validates employee session cookies or Authorization Bearer header.
func SessionAuthMiddleware(store SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/webhooks/") || r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			token := ""

			// 1. Check cookies: __Host-employee-session first, then employee_session
			if c, err := r.Cookie("__Host-employee-session"); err == nil && c.Value != "" {
				token = c.Value
			} else if c, err := r.Cookie("employee_session"); err == nil && c.Value != "" {
				token = c.Value
			}

			// 2. Check Authorization Bearer header
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

			// Ensure CSRF token is present on session cookie refresh
			if GetCSRFCookie(r) == "" {
				if csrfToken, err := GenerateRandomToken(16); err == nil {
					SetCSRFCookie(w, csrfToken)
				}
			}

			ctx := session.WithEmployeeSession(r.Context(), sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoleMiddleware enforces role-based access control (RBAC).
// Allowed roles: "hr", "facilities", "veterinary", "investment", "ops", "engineering", "admin", "staff"
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

			// Admin role always has universal access
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

// CSRFProtectMiddleware validates double-submit CSRF tokens for mutating requests.
func CSRFProtectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
			headerToken := r.Header.Get("X-CSRF-Token")
			cookieToken := GetCSRFCookie(r)

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

// InternalServiceAuthMiddleware authenticates internal service callers (such as worker or internal BLL)
// via an internal service authorization token or shared secret.
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
