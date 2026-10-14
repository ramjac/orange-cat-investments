package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireBankAuth(t *testing.T) {
	expectedClientID := "test_client_id"
	expectedSecret := "test_secret"

	handler := RequireBankAuth(expectedClientID, expectedSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("authorized"))
	}))

	// 1. Healthz should bypass auth
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rrHealth := httptest.NewRecorder()
	handler.ServeHTTP(rrHealth, reqHealth)
	if rrHealth.Code != http.StatusOK {
		t.Errorf("expected 200 for healthz, got %d", rrHealth.Code)
	}

	// 2. Missing credentials should return 401
	reqUnauth := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	rrUnauth := httptest.NewRecorder()
	handler.ServeHTTP(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauth, got %d", rrUnauth.Code)
	}

	// 3. Plaid headers should authenticate
	reqAuth := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	reqAuth.Header.Set("PLAID-CLIENT-ID", expectedClientID)
	reqAuth.Header.Set("PLAID-SECRET", expectedSecret)
	rrAuth := httptest.NewRecorder()
	handler.ServeHTTP(rrAuth, reqAuth)
	if rrAuth.Code != http.StatusOK {
		t.Errorf("expected 200 with Plaid headers, got %d", rrAuth.Code)
	}

	// 4. Bearer token should authenticate
	reqBearer := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+expectedSecret)
	rrBearer := httptest.NewRecorder()
	handler.ServeHTTP(rrBearer, reqBearer)
	if rrBearer.Code != http.StatusOK {
		t.Errorf("expected 200 with Bearer token, got %d", rrBearer.Code)
	}
}
