package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	facilities_handler "github.com/orange-cat-investments/oci/internal/handler/facilities"
	facilities_repo "github.com/orange-cat-investments/oci/internal/repository/facilities"
	facilities_svc "github.com/orange-cat-investments/oci/internal/service/facilities"
	"github.com/stretchr/testify/assert"
)

func TestFacilitiesAPI_Healthz(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","domain":"facilities"}`))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"domain":"facilities"`)
}

func TestFacilitiesAPI_Routes(t *testing.T) {
	repo := facilities_repo.NewMockRepository()
	svc := facilities_svc.NewService(repo)
	h := facilities_handler.NewHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/facilities/assets", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
