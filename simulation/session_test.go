package simulation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersonaCookieJarIsolation(t *testing.T) {
	empServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-employee-session",
			Value:    "sess-emp-12345",
			Path:     "/",
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "employee_csrf_token",
			Value: "csrf-emp-token",
			Path:  "/",
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "authenticated",
			"csrf_token": "csrf-emp-token",
		})
	}))
	defer empServer.Close()

	custServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-customer-session",
			Value:    "sess-cust-67890",
			Path:     "/",
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "csrf_token",
			Value: "csrf-cust-token",
			Path:  "/",
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "authenticated",
			"csrf_token": "csrf-cust-token",
		})
	}))
	defer custServer.Close()

	client, err := NewClient(&Config{
		EmployeeBFFURL: empServer.URL,
		CustomerBFFURL: custServer.URL,
	})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := NewRegistry()
	alice, ok := r.Get("emp-human-alice")
	require.True(t, ok)
	arthur, ok := r.Get("cust-longterm-arthur")
	require.True(t, ok)

	// Authenticate both personas
	require.NoError(t, client.AuthenticatePersona(ctx, alice))
	require.NoError(t, client.AuthenticatePersona(ctx, arthur))

	// Verify Alice's jar contains employee session but NOT customer session
	aliceJar := client.GetCookieJar(alice.ID)
	require.NotNil(t, aliceJar)
	bffParsed, err := url.Parse(client.bffURL)
	require.NoError(t, err)
	aliceCookies := aliceJar.Cookies(bffParsed)

	hasEmployeeSession := false
	hasCustomerSession := false
	for _, c := range aliceCookies {
		if c.Name == "employee_session" || c.Name == "__Host-employee-session" {
			hasEmployeeSession = true
		}
		if c.Name == "customer_session" || c.Name == "__Host-customer-session" {
			hasCustomerSession = true
		}
	}
	assert.True(t, hasEmployeeSession, "Alice should have employee session cookie")
	assert.False(t, hasCustomerSession, "Alice should not have customer session cookie")

	// Verify Arthur's jar contains customer session but NOT employee session
	arthurJar := client.GetCookieJar(arthur.ID)
	require.NotNil(t, arthurJar)
	custParsed, err := url.Parse(client.custBFFURL)
	require.NoError(t, err)
	arthurCookies := arthurJar.Cookies(custParsed)

	hasArthurCustSession := false
	hasArthurEmpSession := false
	for _, c := range arthurCookies {
		if c.Name == "customer_session" || c.Name == "__Host-customer-session" {
			hasArthurCustSession = true
		}
		if c.Name == "employee_session" || c.Name == "__Host-employee-session" {
			hasArthurEmpSession = true
		}
	}
	assert.True(t, hasArthurCustSession, "Arthur should have customer session cookie")
	assert.False(t, hasArthurEmpSession, "Arthur should not have employee session cookie")
}

func TestTransparentDoubleSubmitCSRFInjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/employee/auth/login" {
			http.SetCookie(w, &http.Cookie{
				Name:     "__Host-employee-session",
				Value:    "sess-emp-12345",
				Path:     "/",
				HttpOnly: true,
			})
			http.SetCookie(w, &http.Cookie{
				Name:  "employee_csrf_token",
				Value: "valid-csrf-token",
				Path:  "/",
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":     "authenticated",
				"csrf_token": "valid-csrf-token",
			})
			return
		}

		if r.Method == http.MethodPost {
			csrfHeader := r.Header.Get("X-CSRF-Token")
			if csrfHeader == "" || csrfHeader != "valid-csrf-token" {
				http.Error(w, `{"error":"forbidden: invalid or missing CSRF token"}`, http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status":"created"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(&Config{
		EmployeeBFFURL: server.URL,
	})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := NewRegistry()
	alice, ok := r.Get("emp-human-alice")
	require.True(t, ok)

	// Authenticate Alice
	require.NoError(t, client.AuthenticatePersona(ctx, alice))
	personaCtx := WithPersona(ctx, alice)

	// Mutating POST request with transparent CSRF injection should succeed
	resp, body, err := client.Do(personaCtx, "POST", "/api/v1/workforce/timesheets", map[string]any{
		"employee_id": alice.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected 201 Created with transparent CSRF, body: %s", string(body))

	// Direct request with mismatched CSRF header should return 403 Forbidden
	reqURL := client.bffURL + "/api/v1/workforce/timesheets"
	headers := map[string]string{
		"X-CSRF-Token": "tampered-csrf-token-xyz",
	}
	badResp, badBody, badErr := client.doRequest(personaCtx, reqURL, "POST", map[string]any{
		"employee_id": alice.ID,
	}, headers)
	require.NoError(t, badErr)
	assert.Equal(t, http.StatusForbidden, badResp.StatusCode, "expected 403 Forbidden for mismatched CSRF header")
	assert.Contains(t, string(badBody), "forbidden: invalid or missing CSRF token")
}
