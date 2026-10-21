package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionStore(t *testing.T) {
	t.Run("Create and Get valid session", func(t *testing.T) {
		store := NewSessionStore()
		sess, err := store.Create("emp-001", "alice@oci.local", "Alice", " HR ", "Human Resources")
		if err != nil {
			t.Fatalf("unexpected error creating session: %v", err)
		}

		if sess.EmployeeID != "emp-001" {
			t.Errorf("expected EmployeeID 'emp-001', got '%s'", sess.EmployeeID)
		}
		if sess.Role != "hr" {
			t.Errorf("expected role 'hr', got '%s'", sess.Role)
		}
		if !strings.HasPrefix(sess.SessionID, "emp-sess-") {
			t.Errorf("expected SessionID prefix 'emp-sess-', got '%s'", sess.SessionID)
		}

		got, ok := store.Get(sess.SessionID)
		if !ok || got == nil {
			t.Fatalf("expected to retrieve session from store")
		}
		if got.Email != "alice@oci.local" {
			t.Errorf("expected email 'alice@oci.local', got '%s'", got.Email)
		}
	})

	t.Run("Get non-existent session", func(t *testing.T) {
		store := NewSessionStore()
		_, ok := store.Get("non-existent-token")
		if ok {
			t.Errorf("expected ok to be false for non-existent session")
		}
	})

	t.Run("Get expired session", func(t *testing.T) {
		store := NewSessionStore()
		sess, err := store.Create("emp-002", "bob@oci.local", "Bob", "ops", "Ops")
		if err != nil {
			t.Fatalf("unexpected error creating session: %v", err)
		}

		// Manually expire session
		store.mu.Lock()
		sess.ExpiresAt = time.Now().Add(-1 * time.Hour)
		store.mu.Unlock()

		_, ok := store.Get(sess.SessionID)
		if ok {
			t.Errorf("expected ok to be false for expired session")
		}
	})

	t.Run("Delete session", func(t *testing.T) {
		store := NewSessionStore()
		sess, err := store.Create("emp-003", "charlie@oci.local", "Charlie", "staff", "Staff")
		if err != nil {
			t.Fatalf("unexpected error creating session: %v", err)
		}

		store.Delete(sess.SessionID)

		_, ok := store.Get(sess.SessionID)
		if ok {
			t.Errorf("expected session to be deleted")
		}
	})
}

func TestWithEmployeeSessionAndContext(t *testing.T) {
	t.Run("WithEmployeeSession and EmployeeFromContext roundtrip", func(t *testing.T) {
		sess := &EmployeeSession{
			SessionID:  "sess-123",
			EmployeeID: "emp-001",
			Email:      "test@oci.local",
			Name:       "Test User",
			Role:       "engineering",
			Department: "Engineering",
		}

		ctx := WithEmployeeSession(context.Background(), sess)
		got := EmployeeFromContext(ctx)

		if got == nil {
			t.Fatalf("expected non-nil EmployeeSession, got nil")
		}
		if got.EmployeeID != sess.EmployeeID || got.Email != sess.Email || got.Role != sess.Role {
			t.Errorf("session mismatch: expected %+v, got %+v", sess, got)
		}
	})

	t.Run("EmployeeFromContext returns nil when no session in context", func(t *testing.T) {
		got := EmployeeFromContext(context.Background())
		if got != nil {
			t.Fatalf("expected nil EmployeeSession, got %+v", got)
		}
	})

	t.Run("EmployeeFromContext returns nil when unexpected type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), employeeContextKey, "not-a-session-struct")
		got := EmployeeFromContext(ctx)
		if got != nil {
			t.Fatalf("expected nil EmployeeSession for wrong type, got %+v", got)
		}
	})

	t.Run("SessionAuthMiddleware passes user context to mock handler", func(t *testing.T) {
		store := NewSessionStore()
		sess, err := store.Create("emp-test-01", "mock@oci.local", "Mock User", "ops", "Operations")
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		var capturedSession *EmployeeSession
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedSession = EmployeeFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		middleware := SessionAuthMiddleware(store)
		handler := middleware(mockHandler)

		// 1. Test using employee_session cookie
		req1 := httptest.NewRequest("GET", "/api/v1/protected", nil)
		req1.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
		rec1 := httptest.NewRecorder()

		handler.ServeHTTP(rec1, req1)
		if rec1.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec1.Code)
		}
		if capturedSession == nil || capturedSession.EmployeeID != "emp-test-01" {
			t.Fatalf("expected captured session emp-test-01, got %+v", capturedSession)
		}

		// 2. Test using __Host-employee-session cookie
		capturedSession = nil
		req2 := httptest.NewRequest("GET", "/api/v1/protected", nil)
		req2.AddCookie(&http.Cookie{Name: "__Host-employee-session", Value: sess.SessionID})
		rec2 := httptest.NewRecorder()

		handler.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec2.Code)
		}
		if capturedSession == nil || capturedSession.EmployeeID != "emp-test-01" {
			t.Fatalf("expected captured session emp-test-01, got %+v", capturedSession)
		}

		// 3. Test using Bearer token
		capturedSession = nil
		req3 := httptest.NewRequest("GET", "/api/v1/protected", nil)
		req3.Header.Set("Authorization", "Bearer "+sess.SessionID)
		rec3 := httptest.NewRecorder()

		handler.ServeHTTP(rec3, req3)
		if rec3.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec3.Code)
		}
		if capturedSession == nil || capturedSession.EmployeeID != "emp-test-01" {
			t.Fatalf("expected captured session emp-test-01, got %+v", capturedSession)
		}

		// 4. Test missing token -> 401
		req4 := httptest.NewRequest("GET", "/api/v1/protected", nil)
		rec4 := httptest.NewRecorder()

		handler.ServeHTTP(rec4, req4)
		if rec4.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec4.Code)
		}

		// 5. Test invalid token -> 401
		req5 := httptest.NewRequest("GET", "/api/v1/protected", nil)
		req5.AddCookie(&http.Cookie{Name: "employee_session", Value: "invalid-token"})
		rec5 := httptest.NewRecorder()

		handler.ServeHTTP(rec5, req5)
		if rec5.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec5.Code)
		}
	})
}

func TestCSRFCookieHelpers(t *testing.T) {
	t.Run("getCSRFCookie checks employee_csrf_token and __Host-employee-csrf-token", func(t *testing.T) {
		req1 := httptest.NewRequest("GET", "/", nil)
		if getCSRFCookie(req1) != "" {
			t.Errorf("expected empty string when no cookies set")
		}

		req2 := httptest.NewRequest("GET", "/", nil)
		req2.AddCookie(&http.Cookie{Name: "employee_csrf_token", Value: "token-1"})
		if getCSRFCookie(req2) != "token-1" {
			t.Errorf("expected 'token-1', got '%s'", getCSRFCookie(req2))
		}

		req3 := httptest.NewRequest("GET", "/", nil)
		req3.AddCookie(&http.Cookie{Name: "__Host-employee-csrf-token", Value: "token-2"})
		if getCSRFCookie(req3) != "token-2" {
			t.Errorf("expected 'token-2', got '%s'", getCSRFCookie(req3))
		}
	})

	t.Run("setCSRFCookie sets cookie header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		setCSRFCookie(rec, "new-csrf-token")

		res := rec.Result()
		cookies := res.Cookies()
		if len(cookies) == 0 {
			t.Fatalf("expected cookie to be set")
		}
		if cookies[0].Name != "employee_csrf_token" || cookies[0].Value != "new-csrf-token" {
			t.Errorf("unexpected cookie: %v", cookies[0])
		}
	})
}

func TestHandleGetSessionAndLogout(t *testing.T) {
	store := NewSessionStore()
	handler := NewAuthHandler(store, "http://localhost:3000", false)

	t.Run("HandleGetSession returns session when authenticated", func(t *testing.T) {
		sess := &EmployeeSession{
			SessionID:  "sess-abc",
			EmployeeID: "emp-alice",
			Email:      "alice@oci.local",
			Name:       "Alice",
			Role:       "hr",
			Department: "HR",
		}

		ctx := WithEmployeeSession(context.Background(), sess)
		req := httptest.NewRequest("GET", "/api/v1/employee/auth/session", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.HandleGetSession(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp EmployeeSession
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.EmployeeID != sess.EmployeeID || resp.Role != sess.Role {
			t.Errorf("expected session %+v, got %+v", sess, resp)
		}
	})

	t.Run("HandleGetSession returns 401 when unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/employee/auth/session", nil)
		rec := httptest.NewRecorder()

		handler.HandleGetSession(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("HandleLogout invalidates session and clears cookies", func(t *testing.T) {
		sess, _ := store.Create("emp-bob", "bob@oci.local", "Bob", "facilities", "Facilities")

		req := httptest.NewRequest("POST", "/api/v1/employee/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
		req.AddCookie(&http.Cookie{Name: "__Host-employee-session", Value: sess.SessionID})
		rec := httptest.NewRecorder()

		handler.HandleLogout(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		// Verify session is removed from store
		if _, ok := store.Get(sess.SessionID); ok {
			t.Fatalf("expected session to be deleted from store")
		}

		// Verify cookies were cleared (MaxAge -1)
		cookies := rec.Result().Cookies()
		if len(cookies) < 2 {
			t.Fatalf("expected at least 2 clear-cookie headers, got %d", len(cookies))
		}
		for _, c := range cookies {
			if c.Name == "employee_session" || c.Name == "__Host-employee-session" {
				if c.MaxAge != -1 {
					t.Errorf("expected cookie %s MaxAge to be -1, got %d", c.Name, c.MaxAge)
				}
			}
		}
	})
}

func TestHandleGetCSRFToken(t *testing.T) {
	store := NewSessionStore()
	handler := NewAuthHandler(store, "http://localhost:3000", false)

	t.Run("Generates new CSRF token when none provided", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/employee/csrf", nil)
		rec := httptest.NewRecorder()

		handler.HandleGetCSRFToken(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		token := resp["csrf_token"]
		if token == "" {
			t.Fatalf("expected non-empty csrf_token in response body")
		}

		headerToken := rec.Header().Get("X-CSRF-Token")
		if headerToken != token {
			t.Errorf("expected X-CSRF-Token header %s, got %s", token, headerToken)
		}
	})

	t.Run("Reuses existing CSRF token from cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/employee/csrf", nil)
		existingToken := "existing-csrf-token-12345"
		req.AddCookie(&http.Cookie{Name: "employee_csrf_token", Value: existingToken})
		rec := httptest.NewRecorder()

		handler.HandleGetCSRFToken(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp map[string]string
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		if resp["csrf_token"] != existingToken {
			t.Errorf("expected csrf_token %s, got %s", existingToken, resp["csrf_token"])
		}
	})
}

func TestInternalServiceAuthMiddleware(t *testing.T) {
	middleware := InternalServiceAuthMiddleware()
	mockHandler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	t.Run("Fails without internal service key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/internal/endpoint", nil)
		rec := httptest.NewRecorder()

		mockHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Succeeds with default key in X-Internal-Service-Key header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/internal/endpoint", nil)
		req.Header.Set("X-Internal-Service-Key", "oci-internal-service-secret")
		rec := httptest.NewRecorder()

		mockHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Succeeds with default key in Authorization Bearer header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/internal/endpoint", nil)
		req.Header.Set("Authorization", "Bearer oci-internal-service-secret")
		rec := httptest.NewRecorder()

		mockHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Succeeds with custom environment variable key", func(t *testing.T) {
		t.Setenv("INTERNAL_SERVICE_KEY", "custom-internal-key-999")

		req := httptest.NewRequest("GET", "/api/v1/internal/endpoint", nil)
		req.Header.Set("X-Internal-Service-Key", "custom-internal-key-999")
		rec := httptest.NewRecorder()

		mockHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})
}

func TestHandleForgejoLogin(t *testing.T) {
	// 1. Setup mock Forgejo server
	mockForgejo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" {
			http.NotFound(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if user == "devcat" && pass == "correct-pass" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        101,
				"username":  "devcat",
				"email":     "devcat@oci.local",
				"full_name": "Developer Cat",
				"is_admin":  false,
			})
			return
		}

		if user == "admincat" && pass == "correct-admin-pass" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        1,
				"username":  "admincat",
				"email":     "admincat@oci.local",
				"full_name": "Admin Cat",
				"is_admin":  true,
			})
			return
		}

		if user == "malformed" && pass == "pass" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{invalid-json}`))
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "User does not exist or invalid password"})
	}))
	defer mockForgejo.Close()

	store := NewSessionStore()
	authHandler := NewAuthHandler(store, mockForgejo.URL, false)

	t.Run("Invalid JSON request body returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`invalid-json`))
		rec := httptest.NewRecorder()

		authHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Missing username or password returns 400", func(t *testing.T) {
		payload := `{"username":"devcat","password":""}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		authHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Unconfigured Forgejo URL returns 503", func(t *testing.T) {
		unconfiguredHandler := NewAuthHandler(store, "", false)
		payload := `{"username":"devcat","password":"somepassword"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		unconfiguredHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 Service Unavailable, got %d", rec.Code)
		}
	})

	t.Run("Valid developer user credentials returns 200 and session", func(t *testing.T) {
		payload := `{"username":"devcat","password":"correct-pass"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		authHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode login response: %v", err)
		}

		if resp["status"] != "authenticated" {
			t.Errorf("expected status 'authenticated', got '%v'", resp["status"])
		}

		// Verify cookies were set
		cookies := rec.Result().Cookies()
		hasSessionCookie := false
		for _, c := range cookies {
			if c.Name == "employee_session" || c.Name == "__Host-employee-session" {
				hasSessionCookie = true
			}
		}
		if !hasSessionCookie {
			t.Errorf("expected session cookie to be set in HTTP response")
		}
	})

	t.Run("Valid admin user credentials returns 200 with admin role", func(t *testing.T) {
		payload := `{"username":"admincat","password":"correct-admin-pass"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		authHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp struct {
			Status  string          `json:"status"`
			Session EmployeeSession `json:"session"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Session.Role != "admin" {
			t.Errorf("expected admin role, got '%s'", resp.Session.Role)
		}
		if resp.Session.Department != "Executive Leadership" {
			t.Errorf("expected 'Executive Leadership' department, got '%s'", resp.Session.Department)
		}
	})

	t.Run("Invalid credentials return 401 Unauthorized", func(t *testing.T) {
		payload := `{"username":"devcat","password":"wrong-password"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		authHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Malformed JSON from identity provider returns 500 InternalServerError", func(t *testing.T) {
		payload := `{"username":"malformed","password":"pass"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		authHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 InternalServerError, got %d", rec.Code)
		}
	})

	t.Run("Unreachable identity provider returns 502 Bad Gateway", func(t *testing.T) {
		unreachableHandler := NewAuthHandler(store, "http://127.0.0.1:59999", false)
		payload := `{"username":"devcat","password":"correct-pass"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		unreachableHandler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 Bad Gateway, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_HandleLogin(t *testing.T) {
	store := NewSessionStore()

	t.Run("Production mode blocks self-asserted login with 403 Forbidden", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://localhost:3000", true)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader(`{"employee_id":"emp-1","role":"staff"}`))
		rec := httptest.NewRecorder()

		handler.HandleLogin(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden in prod, got %d", rec.Code)
		}
	})

	t.Run("Dev mode returns 400 Bad Request on invalid JSON", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://localhost:3000", false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader(`invalid json`))
		rec := httptest.NewRecorder()

		handler.HandleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Dev mode returns 400 Bad Request when employee_id or role is missing", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://localhost:3000", false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader(`{"employee_id":"emp-1"}`))
		rec := httptest.NewRecorder()

		handler.HandleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Dev mode successful login creates session and returns 200 OK", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://localhost:3000", false)
		payload := `{"employee_id":"emp-dev-1","email":"dev@oci.local","name":"Dev Name","role":"engineering","department":"Engineering"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		handler.HandleLogin(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if resp["status"] != "authenticated" {
			t.Errorf("expected status 'authenticated', got '%v'", resp["status"])
		}
	})
}
