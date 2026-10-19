package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	opshandler "github.com/orange-cat-investments/oci/internal/handler/ops"
	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestPebbleProxyServer(t *testing.T) (*httptest.Server, opsrepo.Repository) {
	repo := opsrepo.NewMockRepository()
	svc := opssvc.NewService(repo)
	handler := opshandler.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"pebble-proxy"}`))
	})
	handler.RegisterRoutes(mux)

	incidentHandler := requireServiceAuth(handleIncidentAlertBroadcast(repo, nil))
	mux.HandleFunc("POST /api/v1/ops/pebble/incidents/broadcast", incidentHandler)
	mux.HandleFunc("POST /ops/pebble/incidents/broadcast", incidentHandler)
	mux.HandleFunc("POST /ops/pebble/alerts/incident", incidentHandler)

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})
	return server, repo
}

func TestPebbleProxy_HealthCheck(t *testing.T) {
	server, _ := setupTestPebbleProxyServer(t)

	resp, err := http.Get(server.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "pebble-proxy", body["service"])
}

func TestPebbleProxy_IncidentAlertAndPebbleAckLifecycle(t *testing.T) {
	server, _ := setupTestPebbleProxyServer(t)

	// 1. Post workplace safety incident to pebble-proxy broadcast endpoint
	incidentPayload := map[string]interface{}{
		"incident_id": "inc-pebble-test-001",
		"title":       "Coffee Spilled on Core Switch",
		"category":    "facilities",
		"severity":    "critical",
		"room_zone":   "Zone Alpha Server Perch",
		"description": "Feline knocked mug onto 48-port switch stack",
	}
	bodyBytes, err := json.Marshal(incidentPayload)
	require.NoError(t, err)

	// Unauthenticated request should be rejected with 401 Unauthorized
	unauthResp, err := http.Post(server.URL+"/ops/pebble/alerts/incident", "application/json", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	_ = unauthResp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, unauthResp.StatusCode)

	// Authenticated request with internal service key succeeds
	authReq, err := http.NewRequest("POST", server.URL+"/ops/pebble/alerts/incident", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	authReq.Header.Set("Content-Type", "application/json")
	authReq.Header.Set("X-Internal-Service-Key", "oci-internal-service-secret")

	ingestResp, err := http.DefaultClient.Do(authReq)
	require.NoError(t, err)
	defer ingestResp.Body.Close()

	assert.Equal(t, http.StatusCreated, ingestResp.StatusCode)

	// 2. Query Pebble watch alerts: GET /ops/pebble/alerts
	alertsResp, err := http.Get(server.URL + "/ops/pebble/alerts")
	require.NoError(t, err)
	defer alertsResp.Body.Close()

	assert.Equal(t, http.StatusOK, alertsResp.StatusCode)
	var alerts []*opssvc.PebbleAlertPayload
	err = json.NewDecoder(alertsResp.Body).Decode(&alerts)
	require.NoError(t, err)

	// Find the created alert in pending alerts
	var targetAlert *opssvc.PebbleAlertPayload
	for _, a := range alerts {
		if a.TicketID == "inc-pebble-test-001" {
			targetAlert = a
			break
		}
	}
	require.NotNil(t, targetAlert, "expected incident alert to be present in pending pebble alerts")
	assert.Equal(t, "CRITICAL", targetAlert.Priority)
	assert.Contains(t, targetAlert.Title, "Coffee Spilled on Core Switch")
	assert.NotNil(t, targetAlert.AppMessageDict)
	assert.Equal(t, "inc-pebble-test-001", targetAlert.AppMessageDict[opssvc.PebbleKeyTicketID])
	assert.Equal(t, "CRITICAL", targetAlert.AppMessageDict[opssvc.PebbleKeyPriority])

	// 3. Smartwatch Physical Button ACK: POST /ops/pebble/ack
	ackPayload := map[string]string{
		"ticket_id":       "inc-pebble-test-001",
		"acknowledged_by": "frank_ops_watch",
	}
	ackBytes, err := json.Marshal(ackPayload)
	require.NoError(t, err)

	ackResp, err := http.Post(server.URL+"/ops/pebble/ack", "application/json", bytes.NewReader(ackBytes))
	require.NoError(t, err)
	defer ackResp.Body.Close()

	assert.Equal(t, http.StatusOK, ackResp.StatusCode)
	var ackResult opssvc.PebbleAckResponse
	err = json.NewDecoder(ackResultAckRespBody(ackResp)).Decode(&ackResult)
	require.NoError(t, err)
	assert.Equal(t, "inc-pebble-test-001", ackResult.TicketID)
	assert.Equal(t, "acknowledged", ackResult.State)

	// 4. Query Pebble alerts again: verify it is no longer pending
	alertsResp2, err := http.Get(server.URL + "/ops/pebble/alerts")
	require.NoError(t, err)
	defer alertsResp2.Body.Close()

	var alerts2 []*opssvc.PebbleAlertPayload
	err = json.NewDecoder(alertsResp2.Body).Decode(&alerts2)
	require.NoError(t, err)

	for _, a := range alerts2 {
		assert.NotEqual(t, "inc-pebble-test-001", a.TicketID, "acknowledged alert should no longer appear in pending alerts")
	}
}

func ackResultAckRespBody(resp *http.Response) *bytes.Buffer {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return buf
}
