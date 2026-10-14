package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RequireBankAuth authenticates incoming requests using Plaid headers or Bearer tokens.
func RequireBankAuth(expectedClientID, expectedSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass auth for health probes
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			// If no credentials configured, allow all
			if expectedClientID == "" && expectedSecret == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Plaid-style headers: PLAID-CLIENT-ID and PLAID-SECRET
			clientID := r.Header.Get("PLAID-CLIENT-ID")
			secret := r.Header.Get("PLAID-SECRET")
			if clientID != "" && secret != "" {
				if clientID == expectedClientID && secret == expectedSecret {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 2. Fallback Bearer or API Key headers
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token == expectedSecret || token == expectedClientID {
					next.ServeHTTP(w, r)
					return
				}
			}

			apiKey := r.Header.Get("X-Bank-API-Key")
			if apiKey != "" && (apiKey == expectedSecret || apiKey == expectedClientID) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error_type":    "INVALID_REQUEST",
				"error_code":    "UNAUTHORIZED",
				"error_message": "invalid or missing banking API credentials",
			})
		})
	}
}
