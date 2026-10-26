package httputil

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSONAndError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]string{"id": "123"})
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), `"id":"123"`)

	rec2 := httptest.NewRecorder()
	RenderError(rec2, http.StatusBadRequest, errors.New("bad input"))
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
	assert.Contains(t, rec2.Body.String(), `"error":"bad input"`)
}

func TestDecodeJSON(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"garfield"}`))
	var dst sample
	require.NoError(t, DecodeJSON(req, &dst))
	assert.Equal(t, "garfield", dst.Name)
}

func TestHealthzHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	HealthzHandler(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
}
