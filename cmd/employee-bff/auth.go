package main

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type EmployeeSession struct {
	SessionID   string    `json:"session_id"`
	EmployeeID  string    `json:"employee_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"` // "hr", "facilities", "veterinary", "investment", "ops", "engineering", "staff"
	Department  string    `json:"department"`
	Permissions []string  `json:"permissions,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*EmployeeSession
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*EmployeeSession),
	}
}

func (s *SessionStore) Create(employeeID, email, name, role, department string) (*EmployeeSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := make([]byte, 24)
	if _, err := crand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random session token: %w", err)
	}
	token := "emp-sess-" + hex.EncodeToString(b)
	sess := &EmployeeSession{
		SessionID:  token,
		EmployeeID: employeeID,
		Email:      email,
		Name:       name,
		Role:       strings.ToLower(strings.TrimSpace(role)),
		Department: department,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	s.sessions[token] = sess
	return sess, nil
}

func (s *SessionStore) Get(token string) (*EmployeeSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[token]
	if !ok {
		return nil, false
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return sess, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

type contextKey string

const employeeContextKey contextKey = "employee_session"

func EmployeeFromContext(ctx context.Context) *EmployeeSession {
	if s, ok := ctx.Value(employeeContextKey).(*EmployeeSession); ok {
		return s
	}
	return nil
}

func WithEmployeeSession(ctx context.Context, session *EmployeeSession) context.Context {
	return context.WithValue(ctx, employeeContextKey, session)
}

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func getCSRFCookie(r *http.Request) string {
	if c, err := r.Cookie("employee_csrf_token"); err == nil && c.Value != "" {
		return c.Value
	}
	if c, err := r.Cookie("__Host-employee-csrf-token"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

func setCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "employee_csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Accessible by frontend JS for double-submit header
		SameSite: http.SameSiteLaxMode,
	})
}

type AuthHandler struct {
	store *SessionStore
}

func NewAuthHandler(store *SessionStore) *AuthHandler {
	return &AuthHandler{store: store}
}

func (h *AuthHandler) RegisterAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/employee/csrf", h.HandleGetCSRFToken)
	mux.HandleFunc("POST /api/v1/employee/auth/login", h.HandleLogin)
	mux.HandleFunc("POST /api/v1/employee/auth/logout", h.HandleLogout)
	mux.Handle("GET /api/v1/employee/auth/session", SessionAuthMiddleware(h.store)(http.HandlerFunc(h.HandleGetSession)))
}

func (h *AuthHandler) HandleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
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

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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

	csrfToken, _ := generateRandomToken(16)
	setCSRFCookie(w, csrfToken)
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
	if c, err := r.Cookie("employee_session"); err == nil {
		h.store.Delete(c.Value)
	}
	if c, err := r.Cookie("__Host-employee-session"); err == nil {
		h.store.Delete(c.Value)
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
