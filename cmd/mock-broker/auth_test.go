package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	key := "CK_TEST_KEY"
	secret := "SK_TEST_SECRET"

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("authenticated"))
	})

	handler := AuthMiddleware(key, secret, okHandler)

	// 1. Health bypass
	reqHealth := httptest.NewRequest("GET", "/healthz", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	assert.Equal(t, http.StatusOK, recHealth.Code)

	// 2. Missing credentials
	reqNoAuth := httptest.NewRequest("GET", "/v1/accounts", nil)
	recNoAuth := httptest.NewRecorder()
	handler.ServeHTTP(recNoAuth, reqNoAuth)
	assert.Equal(t, http.StatusUnauthorized, recNoAuth.Code)

	// 3. Invalid credentials
	reqBadAuth := httptest.NewRequest("GET", "/v1/accounts", nil)
	reqBadAuth.Header.Set("APCA-API-KEY-ID", "wrong_key")
	reqBadAuth.Header.Set("APCA-API-SECRET-KEY", "wrong_secret")
	recBadAuth := httptest.NewRecorder()
	handler.ServeHTTP(recBadAuth, reqBadAuth)
	assert.Equal(t, http.StatusUnauthorized, recBadAuth.Code)

	// 4. Valid Header credentials
	reqValid := httptest.NewRequest("GET", "/v1/accounts", nil)
	reqValid.Header.Set("APCA-API-KEY-ID", key)
	reqValid.Header.Set("APCA-API-SECRET-KEY", secret)
	recValid := httptest.NewRecorder()
	handler.ServeHTTP(recValid, reqValid)
	assert.Equal(t, http.StatusOK, recValid.Code)
	assert.Equal(t, "authenticated", recValid.Body.String())

	// 5. Valid Basic Auth credentials
	reqBasic := httptest.NewRequest("GET", "/v1/accounts", nil)
	basicAuth := base64.StdEncoding.EncodeToString([]byte(key + ":" + secret))
	reqBasic.Header.Set("Authorization", "Basic "+basicAuth)
	recBasic := httptest.NewRecorder()
	handler.ServeHTTP(recBasic, reqBasic)
	assert.Equal(t, http.StatusOK, recBasic.Code)
	assert.Equal(t, "authenticated", recBasic.Body.String())

	// 6. Empty expected credentials should not match fallback credentials
	emptyAuthHandler := AuthMiddleware("", "", okHandler)
	reqFallback := httptest.NewRequest("GET", "/v1/accounts", nil)
	reqFallback.Header.Set("APCA-API-KEY-ID", "CK_OCI_TEST_KEY_01")
	reqFallback.Header.Set("APCA-API-SECRET-KEY", "SK_OCI_TEST_SECRET_01")
	recFallback := httptest.NewRecorder()
	emptyAuthHandler.ServeHTTP(recFallback, reqFallback)
	assert.Equal(t, http.StatusUnauthorized, recFallback.Code)
}
