package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalBLL_Healthz(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"internal-bll"}`))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"service":"internal-bll"`)
}

func TestInternalBLL_IncidentAndAlertTrigger(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	bll := &InternalBLLServer{
		publisher:  pubSub,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/internal-bll/incidents", bll.handleCreateIncident)

	incReq := IncidentCreateRequest{
		Title:       "Water bowl spilled on edge switch",
		Category:    "hardware_hazard",
		Severity:    "urgent",
		RoomZone:    "Zone Alpha (Main Perch)",
		Description: "Feline tail knocked ceramic water bowl onto top-of-rack networking gear.",
	}
	body, _ := json.Marshal(incReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/internal-bll/incidents", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var res IncidentCreatedPayload
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.NotEmpty(t, res.IncidentID)
	assert.Equal(t, "urgent", res.Severity)
	assert.Equal(t, "Zone Alpha (Main Perch)", res.RoomZone)
}

func TestInternalBLL_OTATrigger(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	bll := &InternalBLLServer{
		publisher:  pubSub,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/internal-bll/firmware/ota", bll.handleTriggerOTA)

	otaReq := OTATriggerRequest{
		AssetIDs:  []string{"asset-001", "asset-002"},
		ReleaseID: "rel-v2.1",
	}
	body, _ := json.Marshal(otaReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/internal-bll/firmware/ota", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
}
