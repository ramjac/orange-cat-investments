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

	t.Run("Accept header with q=0 on v2 and supported v1 succeeds with v1", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("Accept", "application/vnd.oci.v2+json;q=0, application/vnd.oci.v1+json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", rec.Code)
		}
		if v := rec.Header().Get("X-API-Version"); v != "v1" {
			t.Errorf("expected X-API-Version 'v1', got '%s'", v)
		}
	})

	t.Run("Accept header negotiates acceptable version when preferred version unsupported", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("Accept", "application/vnd.oci.v2+json;q=0.9, application/vnd.oci.v1+json;q=0.5")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", rec.Code)
		}
		if v := rec.Header().Get("X-API-Version"); v != "v1" {
			t.Errorf("expected X-API-Version 'v1', got '%s'", v)
		}
	})

	t.Run("Normalize version handling without 'v' prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("X-API-Version", "1")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", rec.Code)
		}
		if v := rec.Header().Get("X-API-Version"); v != "v1" {
			t.Errorf("expected X-API-Version 'v1', got '%s'", v)
		}
	})

	t.Run("Invalid quality parameter defaults to q=1.0", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
		req.Header.Set("Accept", "application/vnd.oci.v1+json;q=invalid")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK, got %d", rec.Code)
		}
		if v := rec.Header().Get("X-API-Version"); v != "v1" {
			t.Errorf("expected X-API-Version 'v1', got '%s'", v)
		}
	})
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1", "v1"},
		{"v1", "v1"},
		{"V2", "v2"},
		{"  2  ", "v2"},
		{"v10", "v10"},
	}

	for _, tt := range tests {
		if got := normalizeVersion(tt.input); got != tt.expected {
			t.Errorf("normalizeVersion(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestDeprecateZeroTime(t *testing.T) {
	rec := httptest.NewRecorder()
	Deprecate(rec, time.Time{}, "")

	if v := rec.Header().Get("Deprecation"); v != "true" {
		t.Errorf("expected Deprecation 'true', got '%s'", v)
	}
	if v := rec.Header().Get("Sunset"); v != "" {
		t.Errorf("expected empty Sunset header, got '%s'", v)
	}
	if v := rec.Header().Get("Link"); v != "" {
		t.Errorf("expected empty Link header, got '%s'", v)
	}
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
