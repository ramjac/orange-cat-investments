package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmployeeAuthAndRBAC(t *testing.T) {
	store := NewSessionStore()
	hrSess, _ := store.Create("emp-human-alice", "alice@oci.local", "Alice Vance", "hr", "Human Resources")
	facSess, _ := store.Create("emp-human-bob", "bob@oci.local", "Bob Builder", "facilities", "Facilities & Habitat")
	vetSess, _ := store.Create("emp-human-elena", "elena@oci.local", "Dr. Elena Rostova", "veterinary", "Veterinary Medicine")

	authMiddleware := SessionAuthMiddleware(store)
	hrOnlyMiddleware := RequireRoleMiddleware("hr")
	vetOnlyMiddleware := RequireRoleMiddleware("veterinary")

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"authorized"}`))
	})

	hrEndpoint := authMiddleware(hrOnlyMiddleware(dummyHandler))
	vetEndpoint := authMiddleware(vetOnlyMiddleware(dummyHandler))

	t.Run("Unauthenticated request returns 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/review-cycles", nil)
		rec := httptest.NewRecorder()

		hrEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("HR accessing HR endpoint succeeds", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/review-cycles", nil)
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: hrSess.SessionID})
		rec := httptest.NewRecorder()

		hrEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("Facilities role accessing HR endpoint returns 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/review-cycles", nil)
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: facSess.SessionID})
		rec := httptest.NewRecorder()

		hrEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 Forbidden, got %d", rec.Code)
		}

		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["user_role"] != "facilities" {
			t.Errorf("expected user_role 'facilities', got '%v'", resp["user_role"])
		}
	})

	t.Run("Veterinary role accessing Veterinary endpoint succeeds", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/workforce/care-schedules/cat-01/medical-hold", nil)
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: vetSess.SessionID})
		rec := httptest.NewRecorder()

		vetEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("HR role accessing Veterinary endpoint returns 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/workforce/care-schedules/cat-01/medical-hold", nil)
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: hrSess.SessionID})
		rec := httptest.NewRecorder()

		vetEndpoint.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 Forbidden, got %d", rec.Code)
		}
	})
}

func TestEmployeeCSRFProtection(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	protected := CSRFProtectMiddleware(dummyHandler)

	t.Run("GET request succeeds without CSRF header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("POST request fails without CSRF header and cookie", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/workforce/leave-requests", nil)
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rec.Code)
		}
	})

	t.Run("POST request succeeds with matching CSRF header and cookie", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/workforce/leave-requests", nil)
		token := "valid-csrf-token-12345"
		req.Header.Set("X-CSRF-Token", token)
		req.AddCookie(&http.Cookie{Name: "employee_csrf_token", Value: token})
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}
