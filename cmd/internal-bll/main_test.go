package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestInternalBLL_Incident_CanonicalIDAndAsyncPublish(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	// Subscribe to incident created event
	eventChan, err := pubSub.Subscribe(context.Background(), "events.workforce.incident_created.v1")
	require.NoError(t, err)

	// Mock workforce API returning canonical incident_id (Thread 37)
	mockWorkforceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"incident_id": "inc-canonical-uuid-1234",
			"status":      "reported",
		})
	}))
	defer mockWorkforceServer.Close()

	bll := &InternalBLLServer{
		publisher:    pubSub,
		httpClient:   mockWorkforceServer.Client(),
		workforceURL: mockWorkforceServer.URL,
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
	err = json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Equal(t, "inc-canonical-uuid-1234", res.IncidentID, "should use canonical incident ID from workforce API")
	assert.Equal(t, "urgent", res.Severity)

	// Verify asynchronous event was published
	select {
	case msg := <-eventChan:
		msg.Ack()
		assert.Contains(t, string(msg.Payload), "inc-canonical-uuid-1234")
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for incident event publication")
	}
}

func TestInternalBLL_Incident_WorkforceFailure(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	mockWorkforceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal database failure", http.StatusInternalServerError)
	}))
	defer mockWorkforceServer.Close()

	bll := &InternalBLLServer{
		publisher:    pubSub,
		httpClient:   mockWorkforceServer.Client(),
		workforceURL: mockWorkforceServer.URL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/internal-bll/incidents", bll.handleCreateIncident)

	incReq := IncidentCreateRequest{
		Title:       "Hardware fault",
		Severity:    "urgent",
		Description: "Power fault",
	}
	body, _ := json.Marshal(incReq)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/internal-bll/incidents", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestInternalBLL_OTATrigger_PerDeviceJobsAndEvents(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	eventChan, err := pubSub.Subscribe(context.Background(), "events.facilities.ota_triggered.v1")
	require.NoError(t, err)

	var facilitiesCalled atomic.Bool
	mockFacilitiesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		facilitiesCalled.Store(true)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jobs": []map[string]interface{}{
				{"job_id": "job-cam-1", "asset_id": "asset-001", "release_id": "rel-v2.1"},
				{"job_id": "job-cam-2", "asset_id": "asset-002", "release_id": "rel-v2.1"},
			},
		})
	}))
	defer mockFacilitiesServer.Close()

	bll := &InternalBLLServer{
		publisher:     pubSub,
		httpClient:    mockFacilitiesServer.Client(),
		facilitiesURL: mockFacilitiesServer.URL,
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
	assert.True(t, facilitiesCalled.Load())

	// Verify two distinct worker-compatible events received (one per device) (Thread 14)
	var receivedCount int
	for i := 0; i < 2; i++ {
		select {
		case msg := <-eventChan:
			msg.Ack()
			receivedCount++
			assert.Contains(t, string(msg.Payload), "asset-00")
			assert.Contains(t, string(msg.Payload), "edge_camera")
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for OTA event %d", i+1)
		}
	}
	assert.Equal(t, 2, receivedCount)
}

func TestInternalBLL_TriggerReport(t *testing.T) {
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, watermillLogger)

	bll := &InternalBLLServer{
		publisher:  pubSub,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/internal-bll/reports/monthly", bll.handleTriggerReport)

	reqBody, _ := json.Marshal(ReportTriggerRequest{Period: "2026-10", GeneratedBy: "test-admin"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/internal-bll/reports/monthly", bytes.NewReader(reqBody))
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Contains(t, rec.Body.String(), "2026-10")
}
