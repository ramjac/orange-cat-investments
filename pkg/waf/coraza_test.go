package waf

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(Middleware(DefaultConfig()))

	echoOK := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":   true,
			"body": string(body),
		})
	}

	r.Get("/api/v1/customer/search", echoOK)
	r.Post("/api/v1/customer/trading/orders", echoOK)
	r.Post("/nextcloud/api/v1/documents", echoOK)
	r.Get("/ops/tickets", echoOK)
	r.Get("/files/*", echoOK)
	r.Get("/api/v1/customer/ticker", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "flusher missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"ticker\":\"ZOM\",\"price\":101.5}\n\n"))
		flusher.Flush()
	})

	return r
}

func TestCorazaMiddlewareBlocksAttacksAndAllowsLegitimateRequests(t *testing.T) {
	router := newTestRouter()

	attackCases := []struct {
		name     string
		method   string
		target   string
		body     string
		headers  map[string]string
		wantRule string
	}{
		{
			name:     "SQLi tautology in query",
			method:   http.MethodGet,
			target:   "/ops/tickets?q=%27%20OR%201%3D1--",
			wantRule: "942100",
		},
		{
			name:     "SQLi UNION SELECT in query",
			method:   http.MethodGet,
			target:   "/api/v1/customer/search?q=foo%20UNION%20SELECT%20username%2Cpassword%20FROM%20users",
			wantRule: "942110",
		},
		{
			name:     "SQLi tautology in JSON body",
			method:   http.MethodPost,
			target:   "/api/v1/customer/trading/orders",
			body:     `{"symbol":"' OR 1=1--","side":"buy","qty":1}`,
			headers:  map[string]string{"Content-Type": "application/json"},
			wantRule: "942100",
		},
		{
			name:     "XSS script tag in query",
			method:   http.MethodGet,
			target:   "/api/v1/customer/search?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E",
			wantRule: "941100",
		},
		{
			name:     "XSS script tag in JSON body",
			method:   http.MethodPost,
			target:   "/nextcloud/api/v1/documents",
			body:     `{"title":"Note","content":"<script>alert(1)</script>"}`,
			headers:  map[string]string{"Content-Type": "application/json"},
			wantRule: "941100",
		},
		{
			name:     "XSS img onerror payload",
			method:   http.MethodPost,
			target:   "/nextcloud/api/v1/documents",
			body:     `{"title":"Note","content":"<img src=x onerror=alert(1)>"}`,
			headers:  map[string]string{"Content-Type": "application/json"},
			wantRule: "941110",
		},
		{
			name:     "Path traversal in URI path",
			method:   http.MethodGet,
			target:   "/files/../../etc/passwd",
			wantRule: "930100",
		},
		{
			name:     "URL-encoded path traversal in query",
			method:   http.MethodGet,
			target:   "/api/v1/customer/search?path=%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			wantRule: "930100",
		},
		{
			name:     "Command injection in Referer header",
			method:   http.MethodGet,
			target:   "/ops/tickets",
			headers:  map[string]string{"Referer": "https://evil.test/; cat /etc/passwd"},
			wantRule: "932100",
		},
	}

	for _, tc := range attackCases {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = bytes.NewBufferString(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.target, bodyReader)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code)
			assert.Equal(t, tc.wantRule, rec.Header().Get("X-OCI-WAF-Rule"))
			assert.Contains(t, rec.Body.String(), "WAF_RULE_VIOLATION")
			assert.NotContains(t, rec.Body.String(), "goroutine")
			assert.NotContains(t, rec.Body.String(), "runtime.")
		})
	}

	legitCases := []struct {
		name    string
		method  string
		target  string
		body    string
		headers map[string]string
	}{
		{
			name:   "Normal search query",
			method: http.MethodGet,
			target: "/api/v1/customer/search?q=quarterly+catnip+dividend",
		},
		{
			name:    "Legitimate trade order JSON",
			method:  http.MethodPost,
			target:  "/api/v1/customer/trading/orders",
			body:    `{"symbol":"AAPL","side":"buy","qty":10,"order_type":"market","take_profit_price":215.50}`,
			headers: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:    "Legitimate Nextcloud markdown document with apostrophes and dashes",
			method:  http.MethodPost,
			target:  "/nextcloud/api/v1/documents",
			body:    `{"title":"Orange Cat's Q3 Update","content":"All habitats green -- Garfield's nap score and Zoomie Index remain nominal."}`,
			headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	for _, tc := range legitCases {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = bytes.NewBufferString(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.target, bodyReader)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			if tc.body != "" {
				var parsed map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &parsed))
				assert.Equal(t, tc.body, parsed["body"], "WAF must restore r.Body for downstream handlers")
			}
		})
	}
}

func TestCorazaMiddlewarePreservesSSEStreaming(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer/ticker", nil)
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "data: {\"ticker\":\"ZOM\"")

	// Attack on SSE endpoint query string is still blocked.
	badReq := httptest.NewRequest(http.MethodGet, "/api/v1/customer/ticker?symbol=%27%20OR%201%3D1--", nil)
	badReq.Header.Set("Accept", "text/event-stream")
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	assert.Equal(t, http.StatusForbidden, badRec.Code)
}

func TestCorazaMiddlewareSubMillisecondOverhead(t *testing.T) {
	eng, err := NewEngine(DefaultConfig())
	require.NoError(t, err)

	payload := []byte(`{"symbol":"ZOM","side":"buy","qty":25,"limit_price":102.45}`)
	start := time.Now()
	const iterations = 200
	for i := 0; i < iterations; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/customer/trading/orders", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		v, inspectErr := eng.InspectRequest(req)
		require.NoError(t, inspectErr)
		require.Nil(t, v)
	}
	avg := time.Since(start) / iterations
	assert.Less(t, avg, time.Millisecond, "expected per-request WAF inspection overhead < 1ms, got %v", avg)
}
