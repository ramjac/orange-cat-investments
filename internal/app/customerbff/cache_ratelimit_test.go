package customerbff

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerBFF_PortfolioCachingAndConditionalETag(t *testing.T) {
	bff := NewCustomerBFF(newMockCustomerBLLClient(), nil)

	// Ensure in-memory cache and session store
	cache := cachevalkey.NewMemoryCache()
	defer cache.Close()
	bff.SetCache(cache)

	sessStore := sessionvalkey.NewMemoryCustomerSessionStore()
	defer sessStore.Close()
	bff.SetSessionStore(sessStore)

	sess, err := sessStore.Create("cust-cache-test", "Cache Tester", "cache@oci.local")
	require.NoError(t, err)

	mux := http.NewServeMux()
	bff.RegisterRoutes(mux)

	// Request 1: Fresh fetch, should populate cache and set ETag header
	req1 := httptest.NewRequest("GET", "/api/v1/customer/portfolio", nil)
	req1.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	assert.Equal(t, http.StatusOK, rec1.Code)
	etag := rec1.Header().Get("ETag")
	assert.NotEmpty(t, etag, "response must have ETag header")

	// Verify key is in cache
	var cachedData map[string]interface{}
	ok, err := cache.Get(context.Background(), cachevalkey.KeyPortfolio("cust-cache-test"), &cachedData)
	require.NoError(t, err)
	assert.True(t, ok, "portfolio must be present in Valkey/memory cache")

	// Request 2: Conditional request with If-None-Match matching ETag -> 304 Not Modified
	req2 := httptest.NewRequest("GET", "/api/v1/customer/portfolio", nil)
	req2.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusNotModified, rec2.Code, "server must respond with 304 Not Modified on matching ETag")
	assert.Equal(t, etag, rec2.Header().Get("ETag"))
	assert.Empty(t, rec2.Body.String(), "body must be empty on 304")

	// Request 3: Subsequent fetch without conditional header should return cached data with 200
	req3 := httptest.NewRequest("GET", "/api/v1/customer/portfolio", nil)
	req3.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)

	assert.Equal(t, http.StatusOK, rec3.Code)
	assert.Equal(t, etag, rec3.Header().Get("ETag"))

	// Eager Cache Invalidation via Order Execution
	csrfToken := "test-csrf-token"
	orderBody := bytes.NewBufferString(`{"symbol":"AAPL","side":"buy","quantity":5,"price":150.0}`)
	orderReq := httptest.NewRequest("POST", "/api/v1/customer/trading/orders", orderBody)
	orderReq.AddCookie(&http.Cookie{Name: "customer_session", Value: sess.SessionID})
	orderReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	orderReq.Header.Set("X-CSRF-Token", csrfToken)
	orderReq.Header.Set("Content-Type", "application/json")
	orderRec := httptest.NewRecorder()
	mux.ServeHTTP(orderRec, orderReq)

	assert.Equal(t, http.StatusOK, orderRec.Code)

	// Verify cache key was eagerly evicted
	ok, err = cache.Get(context.Background(), cachevalkey.KeyPortfolio("cust-cache-test"), &cachedData)
	require.NoError(t, err)
	assert.False(t, ok, "portfolio cache must be evicted immediately upon trade order placement")
}

func TestCustomerBFF_RateLimiting(t *testing.T) {
	bff := NewCustomerBFF(newMockCustomerBLLClient(), nil)
	limiter := cachevalkey.NewRateLimiter(nil, nil)
	bff.SetRateLimiter(limiter)

	mux := http.NewServeMux()
	bff.RegisterRoutes(mux)

	// 1. Test Login Rate Limiting (5 attempts allowed, 6th rejected)
	clientIP := "198.51.100.42:55432"
	loginPayload := `{"customer_id":"cust-rl-test","name":"RL Tester","email":"rl@oci.local"}`

	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/customer/auth/login", bytes.NewBufferString(loginPayload))
		req.RemoteAddr = clientIP
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "attempt %d should be allowed", i)
	}

	// 6th attempt should be blocked by RateLimitMiddleware with 429
	blockedReq := httptest.NewRequest("POST", "/api/v1/customer/auth/login", bytes.NewBufferString(loginPayload))
	blockedReq.RemoteAddr = clientIP
	blockedReq.Header.Set("Content-Type", "application/json")
	blockedRec := httptest.NewRecorder()
	mux.ServeHTTP(blockedRec, blockedReq)

	assert.Equal(t, http.StatusTooManyRequests, blockedRec.Code)
	assert.NotEmpty(t, blockedRec.Header().Get("Retry-After"))
	assert.Contains(t, blockedRec.Body.String(), "rate limit exceeded")
}

func TestCustomerBFF_MultiDeviceLogoutPurge(t *testing.T) {
	store := sessionvalkey.NewMemoryCustomerSessionStore()
	defer store.Close()

	sess1, err := store.Create("cust-multi", "Multi Tester", "multi@oci.local")
	require.NoError(t, err)
	sess2, err := store.Create("cust-multi", "Multi Tester", "multi@oci.local")
	require.NoError(t, err)

	_, ok1 := store.Get(sess1.SessionID)
	_, ok2 := store.Get(sess2.SessionID)
	assert.True(t, ok1)
	assert.True(t, ok2)

	purged, err := store.PurgeCustomerSessions("cust-multi")
	require.NoError(t, err)
	assert.Equal(t, 2, purged)

	_, ok1 = store.Get(sess1.SessionID)
	_, ok2 = store.Get(sess2.SessionID)
	assert.False(t, ok1, "session 1 must be revoked on multi-device logout")
	assert.False(t, ok2, "session 2 must be revoked on multi-device logout")
}
