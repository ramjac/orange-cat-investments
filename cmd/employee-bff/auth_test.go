package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBasicAuthMiddleware(t *testing.T) {
	expectedUser := "admin"
	expectedPass := "secret123"

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	middleware := BasicAuthMiddleware(expectedUser, expectedPass)
	protectedEndpoint := middleware(dummyHandler)

	t.Run("Missing Authorization header returns 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/protected", nil)
		rec := httptest.NewRecorder()

		protectedEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
		}

		body := strings.TrimSpace(rec.Body.String())
		if body != "Unauthorized" {
			t.Errorf("expected body 'Unauthorized', got '%s'", body)
		}
	})

	t.Run("Invalid BasicAuth credentials return 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/protected", nil)
		req.SetBasicAuth("admin", "wrongpassword")
		rec := httptest.NewRecorder()

		protectedEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Wrong username returns 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/protected", nil)
		req.SetBasicAuth("wronguser", "secret123")
		rec := httptest.NewRecorder()

		protectedEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Valid BasicAuth credentials pass through to handler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/protected", nil)
		req.SetBasicAuth("admin", "secret123")
		rec := httptest.NewRecorder()

		protectedEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "success") {
			t.Errorf("expected success response body, got '%s'", rec.Body.String())
		}
	})
}

func TestAuthHandler_LoginAndSession(t *testing.T) {
	store := NewSessionStore()
	handler := NewAuthHandler(store, "", false)

	t.Run("HandleLogin returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader("invalid-json"))
		rec := httptest.NewRecorder()

		handler.HandleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("HandleLogin returns 400 when missing employee_id or role", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader(`{"email":"test@oci.local"}`))
		rec := httptest.NewRecorder()

		handler.HandleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("HandleLogin succeeds in dev mode with required fields", func(t *testing.T) {
		payload := `{"employee_id":"emp-test-01","email":"test@oci.local","name":"Test User","role":"engineering","department":"Engineering"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		handler.HandleLogin(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["status"] != "authenticated" {
			t.Errorf("expected status 'authenticated', got %v", resp["status"])
		}
	})

	t.Run("HandleGetSession returns session from context", func(t *testing.T) {
		sess, err := store.Create("emp-test-02", "alice@oci.local", "Alice", "hr", "HR")
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		ctx := WithEmployeeSession(httptest.NewRequest("GET", "/api/v1/employee/auth/session", nil).Context(), sess)
		req := httptest.NewRequest("GET", "/api/v1/employee/auth/session", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.HandleGetSession(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var retrieved EmployeeSession
		if err := json.Unmarshal(rec.Body.Bytes(), &retrieved); err != nil {
			t.Fatalf("failed to decode session: %v", err)
		}
		if retrieved.EmployeeID != "emp-test-02" {
			t.Errorf("expected employee_id 'emp-test-02', got '%s'", retrieved.EmployeeID)
		}
	})

	t.Run("HandleLogout invalidates session cookies", func(t *testing.T) {
		sess, err := store.Create("emp-test-03", "bob@oci.local", "Bob", "ops", "Ops")
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/employee/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
		rec := httptest.NewRecorder()

		handler.HandleLogout(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		if _, ok := store.Get(sess.SessionID); ok {
			t.Errorf("expected session to be deleted from store after logout")
		}
	})
}

func TestAuthHandler_ForgejoLogin(t *testing.T) {
	store := NewSessionStore()

	t.Run("Unconfigured Forgejo returns 503 Service Unavailable", func(t *testing.T) {
		handler := NewAuthHandler(store, "", false)
		payload := `{"username":"alice","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", rec.Code)
		}
	})

	t.Run("Missing credentials returns 400 Bad Request", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://localhost:3000", false)
		payload := `{"username":"","password":""}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}
