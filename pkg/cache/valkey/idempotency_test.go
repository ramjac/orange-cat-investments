package valkey

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyMiddleware_RequiresValidUUID(t *testing.T) {
	store := NewIdempotencyStore(nil, nil)
	mw := IdempotencyMiddleware(store, func(r *http.Request) string { return "cust-1" })

	var calls atomic.Int32
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	// Missing header -> 400
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusBadRequest, rec1.Code)
	assert.Equal(t, int32(0), calls.Load())

	// Non-UUID header -> 400
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
	req2.Header.Set(IdempotencyHeader, "not-a-uuid")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
	assert.Equal(t, int32(0), calls.Load())
}

func TestIdempotencyMiddleware_ReplaysCompletedResponseExactlyOnce(t *testing.T) {
	store := NewIdempotencyStore(nil, nil)
	mw := IdempotencyMiddleware(store, func(r *http.Request) string { return "cust-chloe" })

	var calls atomic.Int32
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order_id":"ord-1","attempt":` + string(rune('0'+n)) + `}`))
	}))

	key := uuid.NewString()

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
	req1.Header.Set(IdempotencyHeader, key)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)
	assert.Empty(t, rec1.Header().Get(IdempotencyReplayedHeader))
	assert.Equal(t, `{"order_id":"ord-1","attempt":1}`, rec1.Body.String())

	// Duplicate retry with identical key -> replayed cached response, handler NOT invoked again.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
	req2.Header.Set(IdempotencyHeader, key)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, "true", rec2.Header().Get(IdempotencyReplayedHeader))
	assert.Equal(t, rec1.Body.String(), rec2.Body.String())
	assert.Equal(t, int32(1), calls.Load())
}

func TestIdempotencyMiddleware_InFlightReturns409Conflict(t *testing.T) {
	store := NewIdempotencyStore(nil, nil)
	mw := IdempotencyMiddleware(store, func(r *http.Request) string { return "cust-arthur" })

	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		close(started)
		<-release
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order_id":"ord-inflight"}`))
	}))

	key := uuid.NewString()
	var wg sync.WaitGroup
	rec1 := httptest.NewRecorder()
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
		req.Header.Set(IdempotencyHeader, key)
		h.ServeHTTP(rec1, req)
	}()

	<-started

	// Concurrent request with same key while first is still executing -> 409 Conflict
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
	req2.Header.Set(IdempotencyHeader, key)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusConflict, rec2.Code)

	close(release)
	wg.Wait()

	assert.Equal(t, http.StatusOK, rec1.Code)
	assert.Equal(t, int32(1), calls.Load())
}

func TestIdempotencyMiddleware_ReleasesLockOn5xx(t *testing.T) {
	store := NewIdempotencyStore(nil, nil)
	mw := IdempotencyMiddleware(store, func(r *http.Request) string { return "cust-retry" })

	var calls atomic.Int32
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream timeout"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order_id":"ord-recovered"}`))
	}))

	key := uuid.NewString()

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
	req1.Header.Set(IdempotencyHeader, key)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusBadGateway, rec1.Code)

	// Retry with same key succeeds because 5xx released the lock.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", nil)
	req2.Header.Set(IdempotencyHeader, key)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, `{"order_id":"ord-recovered"}`, rec2.Body.String())
	assert.Equal(t, int32(2), calls.Load())
}
