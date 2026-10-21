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

	// 5. Client ID passed as Bearer token should be rejected (401)
	reqClientIDBearer := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	reqClientIDBearer.Header.Set("Authorization", "Bearer "+expectedClientID)
	rrClientIDBearer := httptest.NewRecorder()
	handler.ServeHTTP(rrClientIDBearer, reqClientIDBearer)
	if rrClientIDBearer.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when client ID used as Bearer token, got %d", rrClientIDBearer.Code)
	}

	// 6. X-Bank-API-Key with secret should authenticate
	reqAPIKey := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	reqAPIKey.Header.Set("X-Bank-API-Key", expectedSecret)
	rrAPIKey := httptest.NewRecorder()
	handler.ServeHTTP(rrAPIKey, reqAPIKey)
	if rrAPIKey.Code != http.StatusOK {
		t.Errorf("expected 200 with X-Bank-API-Key, got %d", rrAPIKey.Code)
	}

	// 7. Unconfigured credentials (empty client ID or secret) should deny requests
	unconfiguredHandler := RequireBankAuth("", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	reqUnconfig := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	rrUnconfig := httptest.NewRecorder()
	unconfiguredHandler.ServeHTTP(rrUnconfig, reqUnconfig)
	if rrUnconfig.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unconfigured auth middleware, got %d", rrUnconfig.Code)
	}

	// Healthz should still succeed when unconfigured
	reqUnconfigHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rrUnconfigHealth := httptest.NewRecorder()
	unconfiguredHandler.ServeHTTP(rrUnconfigHealth, reqUnconfigHealth)
	if rrUnconfigHealth.Code != http.StatusOK {
		t.Errorf("expected 200 for healthz on unconfigured middleware, got %d", rrUnconfigHealth.Code)
	}
}
