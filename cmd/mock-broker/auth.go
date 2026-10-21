package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// AuthMiddleware validates Alpaca API credentials from request headers.
func AuthMiddleware(expectedKey, expectedSecret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass health check endpoints
		if r.URL.Path == "/healthz" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		keyID := r.Header.Get("APCA-API-KEY-ID")
		secretKey := r.Header.Get("APCA-API-SECRET-KEY")

		// Fallback to Basic Auth
		if keyID == "" && secretKey == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Basic ") {
				encoded := strings.TrimPrefix(authHeader, "Basic ")
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				if err == nil {
					parts := strings.SplitN(string(decoded), ":", 2)
					if len(parts) == 2 {
						keyID = parts[0]
						secretKey = parts[1]
					}
				}
			}
		}

		if keyID != expectedKey || secretKey != expectedSecret {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    40110000,
				"message": "unauthorized: invalid or missing APCA-API-KEY-ID / APCA-API-SECRET-KEY headers",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
