package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncidentBroker_SSEStreamAndBroadcast(t *testing.T) {
	broker := NewIncidentBroker(nil)
	assert.Equal(t, 0, broker.ClientCount())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/employee/incidents/stream":
			broker.HandleStream(w, r)
		case "/api/v1/employee/incidents/broadcast":
			broker.HandleBroadcast(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Connect SSE client
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/employee/incidents/stream", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	reader := bufio.NewReader(resp.Body)

	// Read initial connected event
	line1, err := reader.ReadString('\n')
	require.NoError(t, err)
	assert.Contains(t, line1, "event: connected")

	line2, err := reader.ReadString('\n')
	require.NoError(t, err)
	assert.Contains(t, line2, "data: {\"status\":\"connected\"}")

	// Empty separator line
	_, _ = reader.ReadString('\n')

	// Verify client is registered
	assert.Equal(t, 1, broker.ClientCount())

	// 2. Broadcast incident payload
	incidentData := map[string]interface{}{
		"incident_id": "inc-stream-test-01",
		"title":       "Water Glass Knocked over in Server Room",
		"severity":    "critical",
		"room_zone":   "Zone Alpha",
		"description": "Feline knocked glass over rack 3",
	}
	bodyBytes, err := json.Marshal(incidentData)
	require.NoError(t, err)

	bcastReq, err := http.NewRequest("POST", server.URL+"/api/v1/employee/incidents/broadcast", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	bcastReq.Header.Set("Content-Type", "application/json")

	bcastResp, err := http.DefaultClient.Do(bcastReq)
	require.NoError(t, err)
	defer bcastResp.Body.Close()
	assert.Equal(t, http.StatusOK, bcastResp.StatusCode)

	// 3. Receive broadcast event on SSE connection
	done := make(chan struct{})
	var receivedEvent, receivedData string
	go func() {
		defer close(done)
		receivedEvent, _ = reader.ReadString('\n')
		receivedData, _ = reader.ReadString('\n')
	}()

	select {
	case <-done:
		assert.Contains(t, receivedEvent, "event: incident")
		assert.Contains(t, receivedData, "Water Glass Knocked over in Server Room")
		assert.Contains(t, receivedData, "critical")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for incident SSE message")
	}

	// 4. Disconnect client
	cancel()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, broker.ClientCount())
}
