package simulation

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Client wraps HTTP communication with the OCI stack (Domain API server, Employee BFF, Customer BFF, Forgejo, Bank, and Broker).
type Client struct {
	serverURL       string
	bffURL          string
	custBFFURL      string
	brokerURL       string
	bankURL         string
	forgejoURL      string
	mode            string
	transport       http.RoundTripper
	httpClient      *http.Client
	jarsMu          sync.RWMutex
	personaJars     map[string]http.CookieJar
	personaCSRF     map[string]string
	activePersonaID string
}

// NewClient constructs a new simulation client configured to communicate with a running OCI stack.
func NewClient(cfg *Config) (*Client, error) {
	ingressIP := os.Getenv("OCI_INGRESS_IP")
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if ingressIP != "" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err == nil && (strings.HasSuffix(host, ".oci.local") || host == "oci.local") {
				return dialer.DialContext(ctx, network, net.JoinHostPort(ingressIP, port))
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	c := &Client{
		serverURL:   strings.TrimSuffix(cfg.ServerURL, "/"),
		bffURL:      strings.TrimSuffix(cfg.EmployeeBFFURL, "/"),
		custBFFURL:  strings.TrimSuffix(cfg.CustomerBFFURL, "/"),
		brokerURL:   strings.TrimSuffix(cfg.BrokerURL, "/"),
		bankURL:     strings.TrimSuffix(cfg.BankURL, "/"),
		forgejoURL:  strings.TrimSuffix(cfg.ForgejoURL, "/"),
		mode:        cfg.Mode,
		transport:   transport,
		personaJars: make(map[string]http.CookieJar),
		personaCSRF: make(map[string]string),
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}

	return c, nil
}

// Close cleans up client network resources.
func (c *Client) Close() {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}

// GetCookieJar returns the persona's cookie jar, creating one if not present.
func (c *Client) GetCookieJar(personaID string) http.CookieJar {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaJars == nil {
		c.personaJars = make(map[string]http.CookieJar)
	}
	jar, ok := c.personaJars[personaID]
	if !ok {
		jar, _ = cookiejar.New(nil)
		c.personaJars[personaID] = jar
	}
	return jar
}

// SetCookieJar associates a cookie jar with a specific persona.
func (c *Client) SetCookieJar(personaID string, jar http.CookieJar) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaJars == nil {
		c.personaJars = make(map[string]http.CookieJar)
	}
	c.personaJars[personaID] = jar
}

// GetCSRFToken returns the active cached CSRF token for a persona.
func (c *Client) GetCSRFToken(personaID string) string {
	c.jarsMu.RLock()
	defer c.jarsMu.RUnlock()
	if c.personaCSRF != nil {
		return c.personaCSRF[personaID]
	}
	return ""
}

// SetCSRFToken stores the active CSRF token for a persona.
func (c *Client) SetCSRFToken(personaID, token string) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaCSRF == nil {
		c.personaCSRF = make(map[string]string)
	}
	c.personaCSRF[personaID] = token
}

// ResetSession clears the persona's cookie jar and CSRF token cache.
func (c *Client) ResetSession(personaID string) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if c.personaJars != nil {
		jar, _ := cookiejar.New(nil)
		c.personaJars[personaID] = jar
	}
	if c.personaCSRF != nil {
		delete(c.personaCSRF, personaID)
	}
}

// SetActivePersona binds the active persona ID on the client.
func (c *Client) SetActivePersona(p *Persona) {
	c.jarsMu.Lock()
	defer c.jarsMu.Unlock()
	if p != nil {
		c.activePersonaID = p.ID
	} else {
		c.activePersonaID = ""
	}
}

// WithPersona sets the active persona and returns the client instance.
func (c *Client) WithPersona(p *Persona) *Client {
	c.SetActivePersona(p)
	return c
}

func (c *Client) doRequest(ctx context.Context, fullURL, method string, body any, headers map[string]string) (*http.Response, []byte, error) {
	var personaID string
	if p := PersonaFromContext(ctx); p != nil {
		personaID = p.ID
	} else if c.activePersonaID != "" {
		personaID = c.activePersonaID
	}

	jar := c.GetCookieJar(personaID)

	var bodyReader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, nil, fmt.Errorf("failed to encode JSON request body: %w", err)
		}
		bodyReader = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Transparent Double-Submit CSRF Header Injection on mutating requests
	if (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete) && req.Header.Get("X-CSRF-Token") == "" {
		var csrfToken string
		if parsedURL, parseErr := url.Parse(fullURL); parseErr == nil && jar != nil {
			cookies := jar.Cookies(parsedURL)
			for _, ck := range cookies {
				if ck.Name == "csrf_token" || ck.Name == "employee_csrf_token" ||
					ck.Name == "__Host-csrf-token" || ck.Name == "__Host-employee-csrf-token" {
					csrfToken = ck.Value
					break
				}
			}
		}
		if csrfToken == "" && personaID != "" {
			csrfToken = c.GetCSRFToken(personaID)
		}
		if csrfToken != "" {
			req.Header.Set("X-CSRF-Token", csrfToken)

			if parsedURL, parseErr := url.Parse(fullURL); parseErr == nil && jar != nil {
				hasCookie := false
				for _, ck := range jar.Cookies(parsedURL) {
					if ck.Name == "csrf_token" || ck.Name == "employee_csrf_token" {
						hasCookie = true
						break
					}
				}
				if !hasCookie {
					cookieName := "employee_csrf_token"
					if strings.Contains(fullURL, "customer") {
						cookieName = "csrf_token"
					}
					jar.SetCookies(parsedURL, []*http.Cookie{
						{
							Name:  cookieName,
							Value: csrfToken,
							Path:  "/",
						},
					})
				}
			}
		}
	}

	tr := c.transport
	if tr == nil && c.httpClient != nil {
		tr = c.httpClient.Transport
	}
	if tr == nil {
		tr = http.DefaultTransport
	}

	httpClient := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
		Jar:       jar,
	}
	if c.httpClient != nil && c.httpClient.CheckRedirect != nil {
		httpClient.CheckRedirect = c.httpClient.CheckRedirect
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed (%s %s): %w", method, fullURL, err)
	}
	defer resp.Body.Close()

	if token := resp.Header.Get("X-CSRF-Token"); token != "" && personaID != "" {
		c.SetCSRFToken(personaID, token)
	}
	for _, ck := range resp.Cookies() {
		if (ck.Name == "csrf_token" || ck.Name == "employee_csrf_token") && ck.Value != "" && personaID != "" {
			c.SetCSRFToken(personaID, ck.Value)
		}
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	return resp, respBytes, nil
}

// Do executes an HTTP request against the Employee BFF target URL.
func (c *Client) Do(ctx context.Context, useBFF bool, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.bffURL
	if baseURL == "" {
		baseURL = c.serverURL
	}
	return c.doRequest(ctx, baseURL+path, method, body, nil)
}

// DoCustomer executes an HTTP request against the Customer BFF target URL.
func (c *Client) DoCustomer(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.custBFFURL
	if baseURL == "" {
		baseURL = c.serverURL
	}
	return c.doRequest(ctx, baseURL+path, method, body, nil)
}

// DoBroker executes an HTTP request directly against the Mock Broker API.
func (c *Client) DoBroker(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.brokerURL
	if baseURL == "" {
		baseURL = "http://localhost:8089"
	}
	headers := map[string]string{
		"APCA-API-KEY-ID":     "CK_OCI_TEST_KEY_01",
		"APCA-API-SECRET-KEY": "SK_OCI_TEST_SECRET_01",
	}
	return c.doRequest(ctx, baseURL+path, method, body, headers)
}

// DoBank executes an HTTP request directly against the Mock Bank API.
func (c *Client) DoBank(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.bankURL
	if baseURL == "" {
		baseURL = "http://localhost:8088"
	}
	headers := map[string]string{
		"PLAID-CLIENT-ID": "client_oci_sim_01",
		"PLAID-SECRET":    "sec_oci_sim_test_8829",
	}
	return c.doRequest(ctx, baseURL+path, method, body, headers)
}

// DoForgejo executes an HTTP request directly against the Forgejo REST API.
func (c *Client) DoForgejo(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	return c.DoForgejoAs(ctx, "", method, path, body)
}

// DoForgejoAs executes an HTTP request directly against the Forgejo REST API with a specific user token.
func (c *Client) DoForgejoAs(ctx context.Context, username, method, path string, body any) (*http.Response, []byte, error) {
	baseURL := c.forgejoURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	token := os.Getenv("FORGEJO_TOKEN")
	if username == "emp-human-elise" {
		if t := os.Getenv("FORGEJO_ELISE_TOKEN"); t != "" {
			token = t
		}
	} else if username == "emp-human-marcus" {
		if t := os.Getenv("FORGEJO_MARCUS_TOKEN"); t != "" {
			token = t
		}
	}
	var headers map[string]string
	if token != "" {
		headers = map[string]string{
			"Authorization": "token " + token,
		}
	}
	return c.doRequest(ctx, baseURL+path, method, body, headers)
}

// timesheetRef is the minimal workforce timesheet projection used for
// same-day idempotency checks in the simulation actions.
type timesheetRef struct {
	TimesheetID string `json:"timesheet_id"`
	ShiftDate   string `json:"shift_date"`
	Status      string `json:"status"`
}

// findTimesheetByShiftDate queries the workforce timesheet list for a timesheet
// matching the given employee and shift date. It returns the timesheet and true
// when one exists, allowing daily actions to stay idempotent across re-runs.
func (c *Client) findTimesheetByShiftDate(ctx context.Context, employeeID, shiftDate string) (*timesheetRef, bool, error) {
	resp, bytes, err := c.Do(ctx, true, "GET", "/api/v1/workforce/timesheets?employee_id="+employeeID, nil)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("timesheet list request failed: status %d", resp.StatusCode)
	}
	var listResp struct {
		Items []timesheetRef `json:"items"`
	}
	if err := json.Unmarshal(bytes, &listResp); err != nil {
		return nil, false, err
	}
	for i := range listResp.Items {
		if listResp.Items[i].ShiftDate == shiftDate {
			return &listResp.Items[i], true, nil
		}
	}
	return nil, false, nil
}
