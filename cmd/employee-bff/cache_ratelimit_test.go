package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/internal/handler"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	cachevalkey "github.com/orange-cat-investments/oci/pkg/cache/valkey"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmployeeBFF_CareScheduleCachingAndETag(t *testing.T) {
	repo := wfService.NewMockRepository()
	svc := wfService.NewService(repo)
	wfHandler := handler.NewWorkforceHandler(svc, nil, nil)

	cache := cachevalkey.NewMemoryCache()
	defer cache.Close()
	wfHandler.SetCache(cache)

	mux := http.NewServeMux()
	wfHandler.RegisterRoutes(mux)

	// Seed care schedule in mock repo
	_, err := svc.UpdateCareSchedule(context.Background(), &wfService.CareSchedule{
		FelineID:    "fel-tom",
		DietaryPlan: "100% Grain-Free Salmon",
	})
	require.NoError(t, err)

	// 1. Initial GET -> populates cache & returns ETag
	req1 := httptest.NewRequest("GET", "/api/v1/workforce/care-schedules/fel-tom", nil)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	assert.Equal(t, http.StatusOK, rec1.Code)
	etag := rec1.Header().Get("ETag")
	assert.NotEmpty(t, etag, "response must have ETag header")

	// Verify cached
	var cs wfService.CareSchedule
	ok, err := cache.Get(context.Background(), cachevalkey.KeyCareSchedule("fel-tom"), &cs)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "100% Grain-Free Salmon", cs.DietaryPlan)

	// 2. Conditional GET with matching If-None-Match -> 304 Not Modified
	req2 := httptest.NewRequest("GET", "/api/v1/workforce/care-schedules/fel-tom", nil)
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusNotModified, rec2.Code)
	assert.Equal(t, etag, rec2.Header().Get("ETag"))
	assert.Empty(t, rec2.Body.String())

	// 3. Update care schedule -> eager invalidation
	updateBody := `{"dietary_plan":"Turkey & Pumpkin Puree","feeding_times":["08:00","18:00"],"emergency_medical_hold":false}`
	updateReq := httptest.NewRequest("PUT", "/api/v1/workforce/care-schedules/fel-tom", bytes.NewBufferString(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	mux.ServeHTTP(updateRec, updateReq)

	assert.Equal(t, http.StatusOK, updateRec.Code)

	// Verify cache key was evicted
	ok, err = cache.Get(context.Background(), cachevalkey.KeyCareSchedule("fel-tom"), &cs)
	require.NoError(t, err)
	assert.False(t, ok, "care schedule cache must be eagerly evicted upon update")
}

func TestEmployeeBFF_NextcloudCOTSCaching(t *testing.T) {
	repo := wfService.NewMockRepository()
	svc := wfService.NewService(repo)
	wfHandler := handler.NewWorkforceHandler(svc, nil, nil)

	cache := cachevalkey.NewMemoryCache()
	defer cache.Close()
	wfHandler.SetCache(cache)

	mux := http.NewServeMux()
	wfHandler.RegisterRoutes(mux)

	// 1. Initial GET
	req1 := httptest.NewRequest("GET", "/api/v1/workforce/hub/folders", nil)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	assert.Equal(t, http.StatusOK, rec1.Code)
	etag := rec1.Header().Get("ETag")
	assert.NotEmpty(t, etag)

	// 2. Conditional GET -> 304 Not Modified
	req2 := httptest.NewRequest("GET", "/api/v1/workforce/hub/folders", nil)
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusNotModified, rec2.Code)
}

func TestEmployeeBFF_MultiDeviceLogout(t *testing.T) {
	store := sessionvalkey.NewMemoryEmployeeSessionStore()
	defer store.Close()

	sess1, err := store.Create("emp-marcus", "marcus@oci.local", "Marcus", "engineering", "Core")
	require.NoError(t, err)
	sess2, err := store.Create("emp-marcus", "marcus@oci.local", "Marcus", "engineering", "Core")
	require.NoError(t, err)

	_, ok1 := store.Get(sess1.SessionID)
	_, ok2 := store.Get(sess2.SessionID)
	assert.True(t, ok1)
	assert.True(t, ok2)

	purged, err := store.PurgeEmployeeSessions("emp-marcus")
	require.NoError(t, err)
	assert.Equal(t, 2, purged)

	_, ok1 = store.Get(sess1.SessionID)
	_, ok2 = store.Get(sess2.SessionID)
	assert.False(t, ok1, "session 1 must be revoked on multi-device logout")
	assert.False(t, ok2, "session 2 must be revoked on multi-device logout")
}

func TestEmployeeBFF_PebbleAckRateLimiting(t *testing.T) {
	mux := http.NewServeMux()
	limiter := cachevalkey.NewRateLimiter(nil, nil)

	// Dummy handler to test rate limiting wrapper
	count := 0
	wrapped := cachevalkey.RateLimitMiddleware(limiter, func(r *http.Request) string {
		return "pebble_ack:" + cachevalkey.ClientIP(r)
	}, 3, 100*time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))

	mux.Handle("POST /api/v1/ops/pebble/ack", wrapped)

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/ops/pebble/ack", nil)
		req.RemoteAddr = "10.0.0.42:12345"
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 4th should be 429
	req := httptest.NewRequest("POST", "/api/v1/ops/pebble/ack", nil)
	req.RemoteAddr = "10.0.0.42:12345"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
