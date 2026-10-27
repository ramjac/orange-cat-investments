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
		sess.ExpiresAt = time.Now().Add(-1 * time.Hour)
		store.SetSession(sess)

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

func TestEmployeeContextHelpers(t *testing.T) {
	ctx := context.Background()

	if sess := EmployeeFromContext(ctx); sess != nil {
		t.Errorf("expected nil session from empty context, got %v", sess)
	}

	session := &EmployeeSession{
		SessionID:  "emp-sess-12345",
		EmployeeID: "emp-100",
		Email:      "diana@oci.local",
		Role:       "veterinary",
	}

	ctxWithSess := WithEmployeeSession(ctx, session)
	retrieved := EmployeeFromContext(ctxWithSess)

	if retrieved == nil || retrieved.SessionID != session.SessionID {
		t.Errorf("expected retrieved session to match inserted session")
	}
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

func TestAuthHandler_HandleGetCSRFToken(t *testing.T) {
	store := NewSessionStore()
	handler := NewAuthHandler(store, "http://localhost:3000", false)

	t.Run("Generates new CSRF token when none exists", func(t *testing.T) {
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
			t.Errorf("expected non-empty csrf_token in response")
		}
		if rec.Header().Get("X-CSRF-Token") != token {
			t.Errorf("expected X-CSRF-Token header to match response token")
		}
	})

	t.Run("Returns existing CSRF token from cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/employee/csrf", nil)
		req.AddCookie(&http.Cookie{Name: "employee_csrf_token", Value: "existing-csrf-token"})
		rec := httptest.NewRecorder()

		handler.HandleGetCSRFToken(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp map[string]string
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		if resp["csrf_token"] != "existing-csrf-token" {
			t.Errorf("expected 'existing-csrf-token', got '%s'", resp["csrf_token"])
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

func TestAuthHandler_HandleLogout(t *testing.T) {
	store := NewSessionStore()
	sess, err := store.Create("emp-logout-1", "logout@oci.local", "Logout Test", "staff", "General")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	handler := NewAuthHandler(store, "http://localhost:3000", false)
	req := httptest.NewRequest("POST", "/api/v1/employee/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
	req.AddCookie(&http.Cookie{Name: "__Host-employee-session", Value: sess.SessionID})
	rec := httptest.NewRecorder()

	handler.HandleLogout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// Verify session was removed from store
	if _, ok := store.Get(sess.SessionID); ok {
		t.Errorf("expected session to be deleted from store on logout")
	}

	// Verify cookies were cleared (MaxAge < 0 or Value "")
	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected at least 2 cookies to be set for deletion")
	}
	for _, c := range cookies {
		if c.MaxAge != -1 {
			t.Errorf("expected cookie %s to have MaxAge -1, got %d", c.Name, c.MaxAge)
		}
	}
}

func TestAuthHandler_HandleGetSession(t *testing.T) {
	store := NewSessionStore()
	handler := NewAuthHandler(store, "http://localhost:3000", false)

	t.Run("Returns 401 Unauthorized when unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/employee/auth/session", nil)
		rec := httptest.NewRecorder()

		handler.HandleGetSession(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Returns session data when authenticated in context", func(t *testing.T) {
		sess := &EmployeeSession{
			SessionID:  "emp-sess-test",
			EmployeeID: "emp-999",
			Email:      "session@oci.local",
			Role:       "hr",
		}
		req := httptest.NewRequest("GET", "/api/v1/employee/auth/session", nil)
		ctx := WithEmployeeSession(req.Context(), sess)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.HandleGetSession(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var retrieved EmployeeSession
		if err := json.NewDecoder(rec.Body).Decode(&retrieved); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if retrieved.EmployeeID != "emp-999" || retrieved.Role != "hr" {
			t.Errorf("unexpected session retrieved: %v", retrieved)
		}
	})
}
