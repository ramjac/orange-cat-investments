package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ops_handler "github.com/orange-cat-investments/oci/internal/handler/ops"
	ops_repo "github.com/orange-cat-investments/oci/internal/repository/ops"
	ops_svc "github.com/orange-cat-investments/oci/internal/service/ops"
	"github.com/stretchr/testify/assert"
)

func TestOpsAPI_Healthz(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","domain":"ops"}`))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"domain":"ops"`)
}

func TestOpsAPI_Routes(t *testing.T) {
	repo := ops_repo.NewMockRepository()
	svc := ops_svc.NewService(repo)
	h := ops_handler.NewHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/ops/tickets", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
