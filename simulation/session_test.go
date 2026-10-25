package simulation

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersonaCookieJarIsolation(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
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
	client, err := NewClient(&Config{Mode: "mock"})
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
	today := time.Now().UTC().Format("2006-01-02")
	resp, body, err := client.Do(personaCtx, true, "POST", "/api/v1/workforce/timesheets", map[string]any{
		"employee_id":   alice.ID,
		"shift_date":    today,
		"start_time":    fmt.Sprintf("%sT08:00:00Z", today),
		"end_time":      fmt.Sprintf("%sT16:30:00Z", today),
		"break_minutes": 30,
		"task_code":     "admin",
		"billable":      true,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected 201 Created with transparent CSRF, body: %s", string(body))

	// Direct request with mismatched CSRF header should return 403 Forbidden
	reqURL := client.bffURL + "/api/v1/workforce/timesheets"
	headers := map[string]string{
		"X-CSRF-Token": "tampered-csrf-token-xyz",
	}
	badResp, badBody, badErr := client.doRequest(personaCtx, reqURL, "POST", map[string]any{
		"employee_id":   alice.ID,
		"shift_date":    today,
		"start_time":    fmt.Sprintf("%sT08:00:00Z", today),
		"end_time":      fmt.Sprintf("%sT16:30:00Z", today),
		"break_minutes": 30,
	}, headers)
	require.NoError(t, badErr)
	assert.Equal(t, http.StatusForbidden, badResp.StatusCode, "expected 403 Forbidden for mismatched CSRF header")
	assert.Contains(t, string(badBody), "forbidden: invalid or missing CSRF token")
}

func TestMultiStepEmployeeLifecycleJourney(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := NewRegistry()
	bob, ok := r.Get("emp-human-bob")
	require.True(t, ok)

	// Step 1: Pre-login with strict authentication enforced fails
	client.SetRequireAuth(true)
	unauthResp, _, unauthErr := client.Do(ctx, true, "GET", "/api/v1/workforce/timesheets", nil)
	require.NoError(t, unauthErr)
	assert.Equal(t, http.StatusUnauthorized, unauthResp.StatusCode, "unauthenticated request must return 401 in strict mode")

	// Step 2: Automated login handshake
	require.NoError(t, client.AuthenticatePersona(ctx, bob))
	bobCtx := WithPersona(ctx, bob)

	// Step 3: Browse workforce timesheets
	resp, _, err := client.Do(bobCtx, true, "GET", "/api/v1/workforce/timesheets", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 4: Submit daily timesheet (mutating POST with auto CSRF injection)
	today := time.Now().UTC().Format("2006-01-02")
	createResp, createBytes, createErr := client.Do(bobCtx, true, "POST", "/api/v1/workforce/timesheets", map[string]any{
		"employee_id":   bob.ID,
		"shift_date":    today,
		"start_time":    fmt.Sprintf("%sT08:00:00Z", today),
		"end_time":      fmt.Sprintf("%sT16:30:00Z", today),
		"break_minutes": 30,
		"task_code":     "habitat_maintenance",
		"billable":      true,
	})
	require.NoError(t, createErr)
	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "submit timesheet failed: %s", string(createBytes))

	// Step 5: View paystubs / salary slips
	payResp, _, payErr := client.Do(bobCtx, true, "GET", "/api/v1/workforce/payroll/slips", nil)
	require.NoError(t, payErr)
	assert.Equal(t, http.StatusOK, payResp.StatusCode)

	// Step 6: Explicit logout
	logoutErr := client.LogoutPersona(bobCtx, bob)
	require.NoError(t, logoutErr)

	// Step 7: Negative verification — cached session cookie is rejected because session was invalidated in store
	postLogoutResp, _, postLogoutErr := client.Do(bobCtx, true, "GET", "/api/v1/workforce/timesheets", nil)
	require.NoError(t, postLogoutErr)
	assert.Equal(t, http.StatusUnauthorized, postLogoutResp.StatusCode, "request after logout must fail with 401 Unauthorized")
}

func TestMultiStepCustomerLifecycleJourney(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := NewRegistry()
	arthur, ok := r.Get("cust-longterm-arthur")
	require.True(t, ok)

	// Step 1: Pre-login with strict auth enforced fails
	client.SetRequireAuth(true)
	unauthResp, _, unauthErr := client.DoCustomer(ctx, "GET", "/api/v1/customer/portfolio", nil)
	require.NoError(t, unauthErr)
	assert.Equal(t, http.StatusUnauthorized, unauthResp.StatusCode, "unauthenticated customer request must return 401 in strict mode")

	// Step 2: Customer authentication handshake
	require.NoError(t, client.AuthenticatePersona(ctx, arthur))
	arthurCtx := WithPersona(ctx, arthur)

	// Step 3: Verify portfolio holdings
	portResp, portBytes, portErr := client.DoCustomer(arthurCtx, "GET", "/api/v1/customer/portfolio", nil)
	require.NoError(t, portErr)
	assert.Equal(t, http.StatusOK, portResp.StatusCode, "portfolio fetch failed: %s", string(portBytes))

	// Step 4: Submit trade order (mutating POST with auto CSRF injection)
	orderResp, orderBytes, orderErr := client.DoCustomer(arthurCtx, "POST", "/api/v1/customer/trading/orders", map[string]any{
		"portfolio_id": "018f3a9a-2222-7000-8000-000000000002",
		"symbol":       "SPY",
		"side":         "buy",
		"quantity":     5,
		"price":        512.0,
	})
	require.NoError(t, orderErr)
	assert.Equal(t, http.StatusCreated, orderResp.StatusCode, "create order failed: %s", string(orderBytes))

	// Step 5: Audit trade execution logs
	logResp, _, logErr := client.DoCustomer(arthurCtx, "GET", "/api/v1/customer/trade-logs", nil)
	require.NoError(t, logErr)
	assert.Equal(t, http.StatusOK, logResp.StatusCode)

	// Step 6: Explicit customer logout
	require.NoError(t, client.LogoutPersona(arthurCtx, arthur))

	// Step 7: Negative verification — cached session cookie rejected
	postLogoutResp, _, postLogoutErr := client.DoCustomer(arthurCtx, "GET", "/api/v1/customer/portfolio", nil)
	require.NoError(t, postLogoutErr)
	assert.Equal(t, http.StatusUnauthorized, postLogoutResp.StatusCode, "customer request after logout must fail with 401 Unauthorized")
}

func TestNegativeSecuritySessionInvalidationOnOffboarding(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := NewRegistry()
	alice, ok := r.Get("emp-human-alice")
	require.True(t, ok)
	rick, ok := r.Get("emp-human-rick")
	require.True(t, ok)

	// Authenticate both Alice (HR) and Rick (Employee)
	require.NoError(t, client.AuthenticatePersona(ctx, alice))
	require.NoError(t, client.AuthenticatePersona(ctx, rick))

	rickCtx := WithPersona(ctx, rick)
	aliceCtx := WithPersona(ctx, alice)

	// 1. Verify Rick has an active valid session
	initResp, _, initErr := client.Do(rickCtx, true, "GET", "/api/v1/workforce/timesheets", nil)
	require.NoError(t, initErr)
	assert.Equal(t, http.StatusOK, initResp.StatusCode)

	// 2. Alice triggers offboarding for Rick
	offboardResp, offboardBytes, offboardErr := client.Do(aliceCtx, true, "POST", "/api/v1/workforce/employees/"+rick.ID+"/offboard", map[string]any{
		"employee_id":   rick.ID,
		"reason":        "Separation of employment",
		"target_status": "separated",
	})
	require.NoError(t, offboardErr)
	assert.Equal(t, http.StatusOK, offboardResp.StatusCode, "offboarding failed: %s", string(offboardBytes))

	// 3. Negative security check: Rick makes a call using his cached session cookie
	// Invalidation in session store must cause the BFF to strictly reject the request with 401 Unauthorized
	subsequentResp, subsequentBytes, err := client.Do(rickCtx, true, "GET", "/api/v1/workforce/timesheets", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, subsequentResp.StatusCode, "offboarded employee session must return 401 Unauthorized")
	assert.Contains(t, string(subsequentBytes), "unauthorized: invalid or expired employee session")
}

func TestNegativeSecurityCustomerGDPRErasure(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	r := NewRegistry()
	chloe, ok := r.Get("cust-active-chloe")
	require.True(t, ok)

	// 1. Authenticate Chloe
	require.NoError(t, client.AuthenticatePersona(ctx, chloe))
	chloeCtx := WithPersona(ctx, chloe)

	// Verify active session works
	portResp, _, portErr := client.DoCustomer(chloeCtx, "GET", "/api/v1/customer/portfolio", nil)
	require.NoError(t, portErr)
	assert.Equal(t, http.StatusOK, portResp.StatusCode)

	// 2. Chloe requests GDPR erasure
	gdprResp, gdprBytes, gdprErr := client.DoCustomer(chloeCtx, "POST", "/api/v1/customer/privacy/gdpr-erasure", map[string]any{
		"customer_id":   chloe.ID,
		"email":         chloe.Email,
		"confirmation": "I UNDERSTAND THIS ACTION IS IRREVERSIBLE",
		"reason":        "User requested complete erasure under GDPR Article 17",
	})
	require.NoError(t, gdprErr)
	assert.Equal(t, http.StatusOK, gdprResp.StatusCode, "gdpr erasure failed: %s", string(gdprBytes))

	// 3. Verify session invalidation: subsequent call with cached cookie returns 401
	subsequentResp, subsequentBytes, err := client.DoCustomer(chloeCtx, "GET", "/api/v1/customer/portfolio", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, subsequentResp.StatusCode, "erased customer session must return 401 Unauthorized")
	assert.Contains(t, string(subsequentBytes), "unauthorized: invalid or expired customer session")
}
