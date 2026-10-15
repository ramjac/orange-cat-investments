package versioning

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVersioningMiddleware(t *testing.T) {
	cfg := DefaultConfig()
	middleware := Middleware(cfg)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := middleware(dummyHandler)

	t.Run("Default version when no header provided", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if v := rec.Header().Get("X-API-Version"); v != "v1" {
			t.Errorf("expected X-API-Version 'v1', got '%s'", v)
		}
	})

	t.Run("Explicit X-API-Version header v1", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("X-API-Version", "v1")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if v := rec.Header().Get("X-API-Version"); v != "v1" {
			t.Errorf("expected X-API-Version 'v1', got '%s'", v)
		}
	})

	t.Run("Accept vendor media type application/vnd.oci.v1+json", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("Accept", "application/vnd.oci.v1+json, application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if v := rec.Header().Get("X-API-Version"); v != "v1" {
			t.Errorf("expected X-API-Version 'v1', got '%s'", v)
		}
	})

	t.Run("Unsupported X-API-Version returns 406 Not Acceptable", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("X-API-Version", "v2")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotAcceptable {
			t.Fatalf("expected status 406 Not Acceptable, got %d", rec.Code)
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse json response: %v", err)
		}
		if errStr, ok := resp["error"].(string); !ok || errStr != "unsupported API version: v2" {
			t.Errorf("unexpected error payload: %v", resp)
		}
	})

	t.Run("Unsupported Accept vendor media type returns 406", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("Accept", "application/vnd.oci.v3+json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotAcceptable {
			t.Fatalf("expected status 406 Not Acceptable, got %d", rec.Code)
		}
	})
}

func TestDeprecateHelper(t *testing.T) {
	rec := httptest.NewRecorder()
	sunsetTime := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	successor := "/api/v2/facilities/assets"

	Deprecate(rec, sunsetTime, successor)

	if v := rec.Header().Get("Deprecation"); v != "true" {
		t.Errorf("expected Deprecation 'true', got '%s'", v)
	}
	if v := rec.Header().Get("Sunset"); v != "Fri, 01 Jan 2027 00:00:00 GMT" {
		t.Errorf("expected Sunset 'Fri, 01 Jan 2027 00:00:00 GMT', got '%s'", v)
	}
	expectedLink := "</api/v2/facilities/assets>; rel=\"successor-version\""
	if v := rec.Header().Get("Link"); v != expectedLink {
		t.Errorf("expected Link '%s', got '%s'", expectedLink, v)
	}
}
