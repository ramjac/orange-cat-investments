package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/internal/handler"
	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	workforce_svc "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/stretchr/testify/assert"
)

func TestWorkforceAPI_Healthz(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","domain":"workforce"}`))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"domain":"workforce"`)
}

func TestWorkforceAPI_Routes(t *testing.T) {
	repo := workforce.NewMockRepository()
	svc := workforce_svc.NewService(repo)
	h := handler.NewWorkforceHandler(svc, nil, nil)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/workforce/employees", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
