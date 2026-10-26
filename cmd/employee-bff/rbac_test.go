package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
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

func TestEmployeeAuthProductionGuards(t *testing.T) {
	store := NewSessionStore()

	t.Run("Production mode does not register self-asserted login route", func(t *testing.T) {
		mux := http.NewServeMux()
		handler := NewAuthHandler(store, "http://localhost:3000", true)
		handler.RegisterAuthRoutes(mux)

		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound && rec.Code != http.StatusForbidden {
			t.Fatalf("expected 404 Not Found or 403 Forbidden in prod, got %d", rec.Code)
		}
	})

	t.Run("Development mode registers self-asserted login route", func(t *testing.T) {
		mux := http.NewServeMux()
		handler := NewAuthHandler(store, "http://localhost:3000", false)
		handler.RegisterAuthRoutes(mux)

		payload := `{"employee_id":"emp-human-elise","role":"engineering","name":"Elise"}`
		req := httptest.NewRequest("POST", "/api/v1/employee/auth/login", strings.NewReader(payload))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK in dev mode, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

func TestProtectedOpsRoutes(t *testing.T) {
	store := NewSessionStore()
	sess, err := store.Create("emp-human-elise", "elise@oci.local", "Elise Dev", "engineering", "Engineering")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	bllClient := newMockInternalBLLClient()
	opsH := opshandler.NewHandler(bllClient)

	mux := http.NewServeMux()
	registerProtectedOpsRoutes(mux, opsH, store)

	t.Run("Unauthenticated request to /api/v1/ops/tickets returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ops/tickets", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Unauthenticated request to unversioned /ops/tickets returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ops/tickets", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Authenticated request to /api/v1/ops/tickets returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ops/tickets", nil)
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Authenticated POST /api/v1/ops/tickets without CSRF token returns 403", func(t *testing.T) {
		payload := `{"title":"Bug in pipeline","description":"Fix it","severity":"medium"}`
		req := httptest.NewRequest("POST", "/api/v1/ops/tickets", strings.NewReader(payload))
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden without CSRF, got %d", rec.Code)
		}
	})

	t.Run("Authenticated POST /api/v1/ops/tickets with CSRF token succeeds", func(t *testing.T) {
		csrfToken := "csrf-ops-token-12345"
		payload := `{"title":"Bug in pipeline","description":"Fix it","severity":"medium","author_username":"emp-human-elise"}`
		req := httptest.NewRequest("POST", "/api/v1/ops/tickets", strings.NewReader(payload))
		req.AddCookie(&http.Cookie{Name: "employee_session", Value: sess.SessionID})
		req.AddCookie(&http.Cookie{Name: "employee_csrf_token", Value: csrfToken})
		req.Header.Set("X-CSRF-Token", csrfToken)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("expected 201 Created or 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})
}


