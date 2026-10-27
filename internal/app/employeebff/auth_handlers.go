package employeebff

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type AuthHandler struct {
	store      SessionStore
	forgejoURL string
	httpClient *http.Client
	isProd     bool
}

func NewAuthHandler(store SessionStore, forgejoURL string, isProd bool) *AuthHandler {
	return &AuthHandler{
		store:      store,
		forgejoURL: strings.TrimRight(forgejoURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		isProd:     isProd,
	}
}

func (h *AuthHandler) RegisterAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/employee/csrf", h.HandleGetCSRFToken)
	mux.HandleFunc("POST /api/v1/employee/auth/logout", h.HandleLogout)
	mux.Handle("GET /api/v1/employee/auth/session", SessionAuthMiddleware(h.store)(http.HandlerFunc(h.HandleGetSession)))

	// Forgejo-authenticated login endpoint (production & dev)
	mux.HandleFunc("POST /api/v1/employee/auth/forgejo/login", h.HandleForgejoLogin)

	// Self-asserted login is strictly forbidden in production.
	// Only register self-asserted login in non-production development/test modes.
	if !h.isProd {
		mux.HandleFunc("POST /api/v1/employee/auth/login", h.HandleLogin)
	}
}

func (h *AuthHandler) HandleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := GetCSRFCookie(r)
	if token == "" {
		var err error
		token, err = GenerateRandomToken(16)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate csrf token"})
			return
		}
		SetCSRFCookie(w, token)
	}
	w.Header().Set("X-CSRF-Token", token)
	_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": token})
}

func (h *AuthHandler) HandleForgejoLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request payload"})
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "username and password are required"})
		return
	}

	if h.forgejoURL == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Forgejo identity provider is not configured"})
		return
	}

	// Verify credentials against Forgejo REST API (GET /api/v1/user with HTTP Basic Auth)
	reqURL := fmt.Sprintf("%s/api/v1/user", h.forgejoURL)
	forgejoReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, reqURL, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create identity provider request"})
		return
	}
	forgejoReq.SetBasicAuth(req.Username, req.Password)

	resp, err := h.httpClient.Do(forgejoReq)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to connect to Forgejo identity provider"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid Forgejo credentials"})
		return
	}

	var forgejoUser struct {
		ID       int64  `json:"id"`
		UserName string `json:"username"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&forgejoUser); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to parse Forgejo identity response"})
		return
	}

	role := "engineering"
	dept := "Engineering & Operations"
	if forgejoUser.IsAdmin {
		role = "admin"
		dept = "Executive Leadership"
	}
	name := forgejoUser.FullName
	if name == "" {
		name = forgejoUser.UserName
	}

	session, err := h.store.Create(fmt.Sprintf("emp-%s", forgejoUser.UserName), forgejoUser.Email, name, role, dept)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create employee session"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "employee_session",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-employee-session",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	csrfToken, _ := GenerateRandomToken(16)
	SetCSRFCookie(w, csrfToken)
	w.Header().Set("X-CSRF-Token", csrfToken)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "authenticated",
		"session":    session,
		"csrf_token": csrfToken,
	})
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.isProd {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "forbidden: self-asserted login is disabled in production; authenticate through Forgejo (/api/v1/employee/auth/forgejo/login)",
		})
		return
	}

	var req struct {
		EmployeeID string `json:"employee_id"`
		Email      string `json:"email"`
		Name       string `json:"name"`
		Role       string `json:"role"`
		Department string `json:"department"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request payload"})
		return
	}

	if req.EmployeeID == "" || req.Role == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "employee_id and role are required"})
		return
	}

	session, err := h.store.Create(req.EmployeeID, req.Email, req.Name, req.Role, req.Department)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create employee session"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "employee_session",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-employee-session",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	csrfToken, _ := GenerateRandomToken(16)
	SetCSRFCookie(w, csrfToken)
	w.Header().Set("X-CSRF-Token", csrfToken)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "authenticated",
		"session":    session,
		"csrf_token": csrfToken,
	})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var token string
	if c, err := r.Cookie("__Host-employee-session"); err == nil && c.Value != "" {
		token = c.Value
	} else if c, err := r.Cookie("employee_session"); err == nil && c.Value != "" {
		token = c.Value
	}

	var employeeID string
	if sess := EmployeeFromContext(r.Context()); sess != nil && sess.EmployeeID != "" {
		employeeID = sess.EmployeeID
	} else if token != "" {
		if sess, ok := h.store.Get(token); ok && sess != nil {
			employeeID = sess.EmployeeID
		}
	}

	if token != "" {
		h.store.Delete(token)
	}

	if r.URL.Query().Get("all") == "true" && employeeID != "" {
		if _, err := h.store.PurgeEmployeeSessions(employeeID); err != nil {
			slog.Warn("failed to purge all employee sessions", "employee_id", employeeID, "error", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "employee_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-employee-session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

func (h *AuthHandler) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := EmployeeFromContext(r.Context())
	if session == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	_ = json.NewEncoder(w).Encode(session)
}
