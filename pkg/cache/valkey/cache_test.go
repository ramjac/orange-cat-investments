package valkey

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	type PortfolioSnapshot struct {
		CustomerID   string  `json:"customer_id"`
		TotalBalance float64 `json:"total_balance"`
	}

	key := KeyPortfolio("cust-101")

	// 1. Get non-existent
	var p PortfolioSnapshot
	ok, err := c.Get(ctx, key, &p)
	require.NoError(t, err)
	assert.False(t, ok)

	// 2. Set and Get
	orig := PortfolioSnapshot{CustomerID: "cust-101", TotalBalance: 12500.50}
	err = c.Set(ctx, key, orig, 100*time.Millisecond)
	require.NoError(t, err)

	ok, err = c.Get(ctx, key, &p)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, orig.CustomerID, p.CustomerID)
	assert.Equal(t, orig.TotalBalance, p.TotalBalance)

	// 3. Expiration
	time.Sleep(120 * time.Millisecond)
	ok, err = c.Get(ctx, key, &p)
	require.NoError(t, err)
	assert.False(t, ok, "key should have expired")

	// 4. Delete and DeletePrefix
	_ = c.Set(ctx, KeyQuotes("AAPL"), 150.0, time.Minute)
	_ = c.Set(ctx, KeyQuotes("MSFT"), 300.0, time.Minute)
	_ = c.Set(ctx, KeyHabitatTelemetry(), "active", time.Minute)

	err = c.Delete(ctx, KeyHabitatTelemetry())
	require.NoError(t, err)
	var str string
	ok, _ = c.Get(ctx, KeyHabitatTelemetry(), &str)
	assert.False(t, ok)

	err = c.DeletePrefix(ctx, "cache:quotes:")
	require.NoError(t, err)
	var price float64
	ok, _ = c.Get(ctx, KeyQuotes("AAPL"), &price)
	assert.False(t, ok)
	ok, _ = c.Get(ctx, KeyQuotes("MSFT"), &price)
	assert.False(t, ok)
}

func TestConditionalETag(t *testing.T) {
	data := []byte(`{"status":"success","balance":5000}`)
	etag := ComputeETag(data)
	assert.NotEmpty(t, etag)

	// 1. Initial request without If-None-Match
	req := httptest.NewRequest("GET", "/api/v1/customer/portfolio", nil)
	rec := httptest.NewRecorder()
	handled := HandleConditionalETag(rec, req, data)
	assert.False(t, handled)
	assert.Equal(t, etag, rec.Header().Get("ETag"))
	assert.Equal(t, http.StatusOK, rec.Code)

	// 2. Subsequent request with matching If-None-Match
	reqMatch := httptest.NewRequest("GET", "/api/v1/customer/portfolio", nil)
	reqMatch.Header.Set("If-None-Match", etag)
	recMatch := httptest.NewRecorder()
	handled = HandleConditionalETag(recMatch, reqMatch, data)
	assert.True(t, handled)
	assert.Equal(t, http.StatusNotModified, recMatch.Code)
	assert.Equal(t, etag, recMatch.Header().Get("ETag"))

	// 3. Request with weak ETag prefix matching
	reqWeak := httptest.NewRequest("GET", "/api/v1/customer/portfolio", nil)
	reqWeak.Header.Set("If-None-Match", "W/"+etag)
	recWeak := httptest.NewRecorder()
	handled = HandleConditionalETag(recWeak, reqWeak, data)
	assert.True(t, handled)
	assert.Equal(t, http.StatusNotModified, recWeak.Code)

	// 4. Request with stale ETag
	reqMismatch := httptest.NewRequest("GET", "/api/v1/customer/portfolio", nil)
	reqMismatch.Header.Set("If-None-Match", `"old-hash-12345"`)
	recMismatch := httptest.NewRecorder()
	handled = HandleConditionalETag(recMismatch, reqMismatch, data)
	assert.False(t, handled)
	assert.Equal(t, http.StatusOK, recMismatch.Code)
	assert.Equal(t, etag, recMismatch.Header().Get("ETag"))
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(nil, nil)
	ctx := context.Background()

	key := "test-customer:cust-101"
	limit := 3
	window := 100 * time.Millisecond

	// 1. First 3 requests allowed
	for i := 0; i < limit; i++ {
		allowed, rem, _, err := limiter.Allow(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, limit-1-i, rem)
	}

	// 2. 4th request within window rejected
	allowed, rem, retryAfter, err := limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, rem)
	assert.True(t, retryAfter > 0)

	// 3. After window passes, allowed again
	time.Sleep(120 * time.Millisecond)
	allowed, rem, _, err = limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 2, rem)
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(nil, nil)
	keyFunc := func(r *http.Request) string {
		return "ip:" + ClientIP(r)
	}

	limit := 2
	window := 150 * time.Millisecond

	handler := RateLimitMiddleware(limiter, keyFunc, limit, window)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/customer/trading/orders", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Request 1: OK
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req)
	assert.Equal(t, http.StatusOK, rec1.Code)
	assert.Equal(t, "1", rec1.Header().Get("X-RateLimit-Remaining"))

	// Request 2: OK
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, "0", rec2.Header().Get("X-RateLimit-Remaining"))

	// Request 3: 429
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req)
	assert.Equal(t, http.StatusTooManyRequests, rec3.Code)
	assert.NotEmpty(t, rec3.Header().Get("Retry-After"))
}

func TestInvalidationListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewMemoryCache()
	defer cache.Close()

	pubsub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermill.NewStdLogger(false, false))
	defer pubsub.Close()

	listener := NewInvalidationListener(cache, pubsub, nil)
	listener.Start(ctx)
	defer listener.Stop()

	// Seed cache entries
	_ = cache.Set(ctx, KeyPortfolio("cust-99"), "portfolio-data", time.Hour)
	_ = cache.Set(ctx, KeyCareSchedule("fel-42"), "care-data", time.Hour)
	_ = cache.Set(ctx, KeyActiveIncidents(), "incidents-data", time.Hour)

	// 1. Emit trade order placed -> evicts portfolio
	tradeEnv := envelope.NewEventEnvelope("evt-1", "events.investment.order_placed.v1", "ord-1", map[string]string{
		"customer_id": "cust-99",
	})
	tradeBytes, _ := json.Marshal(tradeEnv)
	err := pubsub.Publish("events.investment.order_placed.v1", message.NewMessage("msg-1", tradeBytes))
	require.NoError(t, err)

	// 2. Emit care schedule updated -> evicts care schedule
	careEnv := envelope.NewEventEnvelope("evt-2", "events.workforce.care_schedule_updated.v1", "cs-1", map[string]string{
		"feline_id": "fel-42",
	})
	careBytes, _ := json.Marshal(careEnv)
	err = pubsub.Publish("events.workforce.care_schedule_updated.v1", message.NewMessage("msg-2", careBytes))
	require.NoError(t, err)

	// 3. Emit incident created -> evicts active incidents
	incEnv := envelope.NewEventEnvelope("evt-3", "events.workforce.incident_created.v1", "inc-1", map[string]string{
		"incident_id": "inc-1",
	})
	incBytes, _ := json.Marshal(incEnv)
	err = pubsub.Publish("events.workforce.incident_created.v1", message.NewMessage("msg-3", incBytes))
	require.NoError(t, err)

	// Wait for background events to process
	time.Sleep(100 * time.Millisecond)

	var str string
	ok, _ := cache.Get(ctx, KeyPortfolio("cust-99"), &str)
	assert.False(t, ok, "portfolio cache should be evicted")

	ok, _ = cache.Get(ctx, KeyCareSchedule("fel-42"), &str)
	assert.False(t, ok, "care schedule cache should be evicted")

	ok, _ = cache.Get(ctx, KeyActiveIncidents(), &str)
	assert.False(t, ok, "incident cache should be evicted")
}
