package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthHandler_HandleForgejoLogin(t *testing.T) {
	store := NewSessionStore()

	t.Run("Invalid JSON payload returns 400", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://localhost:3000", false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Missing username or password returns 400", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://localhost:3000", false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`{"username":"user"}`))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Unconfigured Forgejo URL returns 503", func(t *testing.T) {
		handler := NewAuthHandler(store, "", false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`{"username":"user","password":"pass"}`))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 Service Unavailable, got %d", rec.Code)
		}
	})

	t.Run("Failed connection to Forgejo provider returns 502", func(t *testing.T) {
		handler := NewAuthHandler(store, "http://127.0.0.1:59999", false) // unused port
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`{"username":"user","password":"pass"}`))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 Bad Gateway, got %d", rec.Code)
		}
	})

	t.Run("Forgejo provider returning non-200 returns 401 Unauthorized", func(t *testing.T) {
		mockForgejo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer mockForgejo.Close()

		handler := NewAuthHandler(store, mockForgejo.URL, false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`{"username":"user","password":"wrongpass"}`))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Forgejo provider returning invalid user JSON returns 500", func(t *testing.T) {
		mockForgejo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`invalid json`))
		}))
		defer mockForgejo.Close()

		handler := NewAuthHandler(store, mockForgejo.URL, false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`{"username":"user","password":"pass"}`))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 Internal Server Error, got %d", rec.Code)
		}
	})

	t.Run("Successful Forgejo login for standard user", func(t *testing.T) {
		mockForgejo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "devuser" || pass != "secret123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        101,
				"username":  "devuser",
				"email":     "devuser@oci.local",
				"full_name": "Dev User",
				"is_admin":  false,
			})
		}))
		defer mockForgejo.Close()

		handler := NewAuthHandler(store, mockForgejo.URL, false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`{"username":"devuser","password":"secret123"}`))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to parse response body: %v", err)
		}

		if resp["status"] != "authenticated" {
			t.Errorf("expected status 'authenticated', got '%v'", resp["status"])
		}

		cookies := rec.Result().Cookies()
		foundSessionCookie := false
		for _, c := range cookies {
			if c.Name == "employee_session" && strings.HasPrefix(c.Value, "emp-sess-") {
				foundSessionCookie = true
			}
		}
		if !foundSessionCookie {
			t.Errorf("expected employee_session cookie to be set")
		}
	})

	t.Run("Successful Forgejo login for admin user", func(t *testing.T) {
		mockForgejo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        1,
				"username":  "adminuser",
				"email":     "admin@oci.local",
				"full_name": "",
				"is_admin":  true,
			})
		}))
		defer mockForgejo.Close()

		handler := NewAuthHandler(store, mockForgejo.URL, false)
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/forgejo/login", strings.NewReader(`{"username":"adminuser","password":"secret123"}`))
		rec := httptest.NewRecorder()

		handler.HandleForgejoLogin(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})
}
