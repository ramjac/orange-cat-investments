package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockInternalBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	mu       sync.Mutex
	tickets  map[string]*opsv1.ITTicket
	alerts   map[string]*opsv1.PebbleAlert
	comments map[string][]*opsv1.TicketComment
}

func newMockInternalBLLClient() *mockInternalBLLClient {
	return &mockInternalBLLClient{
		tickets:  make(map[string]*opsv1.ITTicket),
		alerts:   make(map[string]*opsv1.PebbleAlert),
		comments: make(map[string][]*opsv1.TicketComment),
	}
}

func (m *mockInternalBLLClient) CreateTicket(ctx context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ticketID := fmt.Sprintf("ticket-%d", time.Now().UnixNano())
	nowStr := time.Now().UTC().Format(time.RFC3339)

	ticket := &opsv1.ITTicket{
		TicketID:       ticketID,
		Title:          in.Title,
		Body:           in.Body,
		State:          "open",
		Severity:       in.Severity,
		Category:       in.Category,
		Source:         in.Source,
		AuthorUsername: in.AuthorUsername,
		IsUrgent:       in.IsUrgent,
		CreatedAt:      nowStr,
		UpdatedAt:      nowStr,
	}
	m.tickets[ticketID] = ticket

	return &opsv1.CreateTicketResponse{Ticket: ticket}, nil
}

func (m *mockInternalBLLClient) GetTicket(ctx context.Context, in *opsv1.GetTicketRequest) (*opsv1.GetTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ticket, ok := m.tickets[in.TicketID]
	if !ok {
		return nil, fmt.Errorf("ticket %s not found", in.TicketID)
	}
	return &opsv1.GetTicketResponse{Ticket: ticket}, nil
}

func (m *mockInternalBLLClient) ListTickets(ctx context.Context, in *opsv1.ListTicketsRequest) (*opsv1.ListTicketsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tickets := make([]*opsv1.ITTicket, 0, len(m.tickets))
	for _, t := range m.tickets {
		tickets = append(tickets, t)
	}
	return &opsv1.ListTicketsResponse{Tickets: tickets}, nil
}

func (m *mockInternalBLLClient) UpdateTicket(ctx context.Context, in *opsv1.UpdateTicketRequest) (*opsv1.UpdateTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ticket, ok := m.tickets[in.TicketID]
	if !ok {
		return nil, fmt.Errorf("ticket %s not found", in.TicketID)
	}
	if in.State != "" {
		ticket.State = in.State
	}
	if in.AssigneeUsername != "" {
		ticket.AssigneeUsername = in.AssigneeUsername
	}
	if in.ResolutionNotes != "" {
		ticket.ResolutionNotes = in.ResolutionNotes
	}
	ticket.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return &opsv1.UpdateTicketResponse{Ticket: ticket}, nil
}

func (m *mockInternalBLLClient) CreateAlert(ctx context.Context, in *opsv1.CreateAlertRequest) (*opsv1.CreateAlertResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alertID := in.TicketID
	if alertID == "" {
		alertID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}
	nowStr := time.Now().UTC().Format(time.RFC3339)

	alert := &opsv1.PebbleAlert{
		AlertID:      alertID,
		TicketID:     alertID,
		Title:        in.Title,
		Severity:     in.Severity,
		RoomZone:     in.RoomZone,
		Acknowledged: false,
		CreatedAt:    nowStr,
	}
	m.alerts[alertID] = alert
	return &opsv1.CreateAlertResponse{Alert: alert}, nil
}

func (m *mockInternalBLLClient) ListAlerts(ctx context.Context, in *opsv1.ListAlertsRequest) (*opsv1.ListAlertsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var alerts []*opsv1.PebbleAlert
	for _, a := range m.alerts {
		if in.UnacknowledgedOnly && a.Acknowledged {
			continue
		}
		alerts = append(alerts, a)
	}
	return &opsv1.ListAlertsResponse{Alerts: alerts}, nil
}

func (m *mockInternalBLLClient) AcknowledgeAlert(ctx context.Context, in *opsv1.AcknowledgeAlertRequest) (*opsv1.AcknowledgeAlertResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, ok := m.alerts[in.AlertID]
	if !ok {
		return nil, fmt.Errorf("alert %s not found", in.AlertID)
	}
	alert.Acknowledged = true
	nowStr := time.Now().UTC().Format(time.RFC3339)

	if t, ok := m.tickets[in.AlertID]; ok {
		t.State = "acknowledged"
		t.AcknowledgedBy = in.AcknowledgedBy
		t.AcknowledgedAt = nowStr
	}

	return &opsv1.AcknowledgeAlertResponse{
		AlertID:        in.AlertID,
		State:          "acknowledged",
		AcknowledgedBy: in.AcknowledgedBy,
		AcknowledgedAt: nowStr,
		StatusMessage:  fmt.Sprintf("Ticket %s acknowledged by %s via Pebble Watch companion app", in.AlertID, in.AcknowledgedBy),
	}, nil
}

func (m *mockInternalBLLClient) AddTicketComment(ctx context.Context, in *opsv1.AddTicketCommentRequest) (*opsv1.AddTicketCommentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	comment := &opsv1.TicketComment{
		CommentID:      fmt.Sprintf("comm-%d", time.Now().UnixNano()),
		TicketID:       in.TicketID,
		AuthorUsername: in.AuthorUsername,
		Comment:        in.Comment,
		IsInternal:     in.IsInternal,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	m.comments[in.TicketID] = append(m.comments[in.TicketID], comment)
	return &opsv1.AddTicketCommentResponse{Comment: comment}, nil
}

func (m *mockInternalBLLClient) ListTicketComments(ctx context.Context, in *opsv1.ListTicketCommentsRequest) (*opsv1.ListTicketCommentsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &opsv1.ListTicketCommentsResponse{Comments: m.comments[in.TicketID]}, nil
}

func (m *mockInternalBLLClient) SendPushNotification(ctx context.Context, in *opsv1.SendPushNotificationRequest) (*opsv1.SendPushNotificationResponse, error) {
	return &opsv1.SendPushNotificationResponse{
		Status:       "dispatched",
		Provider:     "ntfy",
		Topic:        in.Topic,
		Title:        in.Title,
		Message:      in.Message,
		DispatchedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (m *mockInternalBLLClient) CreateCustomerTicket(ctx context.Context, in *opsv1.CreateCustomerTicketRequest) (*opsv1.CreateCustomerTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ticketID := fmt.Sprintf("cust-ticket-%d", time.Now().UnixNano())
	ticket := &opsv1.ITTicket{
		TicketID:   ticketID,
		Title:      in.Title,
		Body:       in.Body,
		Category:   in.Category,
		CustomerID: in.CustomerID,
		State:      "open",
		Source:     "customer",
	}
	m.tickets[ticketID] = ticket
	return &opsv1.CreateCustomerTicketResponse{Ticket: ticket}, nil
}

func (m *mockInternalBLLClient) ListCustomerTickets(ctx context.Context, in *opsv1.ListCustomerTicketsRequest) (*opsv1.ListCustomerTicketsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var tickets []*opsv1.ITTicket
	for _, t := range m.tickets {
		if in.CustomerID == "" || t.CustomerID == in.CustomerID {
			tickets = append(tickets, t)
		}
	}
	return &opsv1.ListCustomerTicketsResponse{Tickets: tickets}, nil
}

func setupTestPebbleProxyServer(t *testing.T) (*httptest.Server, *mockInternalBLLClient) {
	mockClient := newMockInternalBLLClient()
	router := NewRouter(mockClient, nil)

	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})
	return server, mockClient
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
	err = json.NewDecoder(ackResp.Body).Decode(&ackResult)
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

func TestPebbleProxy_TicketsRoutes(t *testing.T) {
	server, _ := setupTestPebbleProxyServer(t)

	// 1. Create IT ticket
	createTicketPayload := map[string]interface{}{
		"title":           "Core Switch Port 12 Flapping",
		"body":            "Link flaps every 30 seconds",
		"severity":        "high",
		"category":        "hardware",
		"source":          "pebble-proxy",
		"author_username": "emp-frank",
	}
	ticketBytes, err := json.Marshal(createTicketPayload)
	require.NoError(t, err)

	createResp, err := http.Post(server.URL+"/ops/tickets", "application/json", bytes.NewReader(ticketBytes))
	require.NoError(t, err)
	defer createResp.Body.Close()
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	var createdTicket opsv1.ITTicket
	err = json.NewDecoder(createResp.Body).Decode(&createdTicket)
	require.NoError(t, err)
	assert.NotEmpty(t, createdTicket.TicketID)
	assert.Equal(t, "Core Switch Port 12 Flapping", createdTicket.Title)

	// 2. List tickets
	listResp, err := http.Get(server.URL + "/ops/tickets")
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var tickets []*opsv1.ITTicket
	err = json.NewDecoder(listResp.Body).Decode(&tickets)
	require.NoError(t, err)
	assert.NotEmpty(t, tickets)

	// 3. Add ticket comment
	commentPayload := map[string]interface{}{
		"author_username": "emp-elise",
		"comment":         "Replaced SFP+ transceiver",
		"is_internal":     false,
	}
	commBytes, err := json.Marshal(commentPayload)
	require.NoError(t, err)

	commResp, err := http.Post(server.URL+"/ops/tickets/"+createdTicket.TicketID+"/comments", "application/json", bytes.NewReader(commBytes))
	require.NoError(t, err)
	defer commResp.Body.Close()
	assert.Equal(t, http.StatusCreated, commResp.StatusCode)

	// 4. List comments
	listCommResp, err := http.Get(server.URL + "/ops/tickets/" + createdTicket.TicketID + "/comments")
	require.NoError(t, err)
	defer listCommResp.Body.Close()
	assert.Equal(t, http.StatusOK, listCommResp.StatusCode)

	var commList map[string]interface{}
	err = json.NewDecoder(listCommResp.Body).Decode(&commList)
	require.NoError(t, err)
	assert.Equal(t, float64(1), commList["total"])
}
