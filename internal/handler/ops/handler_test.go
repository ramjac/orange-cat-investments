package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/orange-cat-investments/oci/pkg/pebble"
	"github.com/stretchr/testify/assert"
)

type mockInternalBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	lastLimit int32
}

func (m *mockInternalBLLClient) ListAlerts(ctx context.Context, in *opsv1.ListAlertsRequest) (*opsv1.ListAlertsResponse, error) {
	return &opsv1.ListAlertsResponse{
		Alerts: []*opsv1.PebbleAlert{
			{
				AlertID:  "ticket-test-123",
				TicketID: "ticket-test-123",
				Title:    "Pod OOMKilled Warning",
				Severity: "HIGH",
				Body:     "Memory usage exceeded threshold",
				Author:   "emp-human-frank",
			},
		},
	}, nil
}

func (m *mockInternalBLLClient) AcknowledgeAlert(ctx context.Context, in *opsv1.AcknowledgeAlertRequest) (*opsv1.AcknowledgeAlertResponse, error) {
	return &opsv1.AcknowledgeAlertResponse{
		AlertID:        in.AlertID,
		State:          "acknowledged",
		AcknowledgedBy: in.AcknowledgedBy,
		AcknowledgedAt: time.Now().UTC().Format(time.RFC3339),
		StatusMessage:  "Ticket acknowledged via Pebble Watch",
	}, nil
}

func (m *mockInternalBLLClient) CreateTicket(ctx context.Context, in *opsv1.CreateTicketRequest) (*opsv1.CreateTicketResponse, error) {
	return &opsv1.CreateTicketResponse{
		Ticket: &opsv1.ITTicket{
			TicketID:       "created-ticket-123",
			ForgejoRepo:    in.ForgejoRepo,
			Title:          in.Title,
			Body:           in.Body,
			State:          "open",
			AuthorUsername: in.AuthorUsername,
			Severity:       in.Severity,
			Category:       in.Category,
			Source:         in.Source,
		},
	}, nil
}

func (m *mockInternalBLLClient) GetTicket(ctx context.Context, in *opsv1.GetTicketRequest) (*opsv1.GetTicketResponse, error) {
	return &opsv1.GetTicketResponse{
		Ticket: &opsv1.ITTicket{
			TicketID:       in.TicketID,
			ForgejoRepo:    "infra/microk8s",
			Title:          "Sample Ticket",
			State:          "open",
			AuthorUsername: "emp-human-frank",
		},
	}, nil
}

func (m *mockInternalBLLClient) ListTickets(ctx context.Context, in *opsv1.ListTicketsRequest) (*opsv1.ListTicketsResponse, error) {
	m.lastLimit = in.Limit
	return &opsv1.ListTicketsResponse{
		Tickets: []*opsv1.ITTicket{
			{
				TicketID:       "ticket-test-123",
				ForgejoRepo:    "infra/microk8s",
				Title:          "Pod OOMKilled Warning",
				State:          "open",
				AuthorUsername: "emp-human-frank",
			},
		},
	}, nil
}

func (m *mockInternalBLLClient) ListCustomerTickets(ctx context.Context, in *opsv1.ListCustomerTicketsRequest) (*opsv1.ListCustomerTicketsResponse, error) {
	return &opsv1.ListCustomerTicketsResponse{
		Tickets: []*opsv1.ITTicket{
			{
				TicketID:       "cust-ticket-999",
				Title:          "Sample Customer Ticket",
				Source:         "customer",
				CustomerID:     in.CustomerID,
				AuthorUsername: in.CustomerID,
				State:          "open",
			},
		},
	}, nil
}

func (m *mockInternalBLLClient) UpdateTicket(ctx context.Context, in *opsv1.UpdateTicketRequest) (*opsv1.UpdateTicketResponse, error) {
	return &opsv1.UpdateTicketResponse{
		Ticket: &opsv1.ITTicket{
			TicketID:         in.TicketID,
			State:            in.State,
			AssigneeUsername: in.AssigneeUsername,
			ResolutionNotes:  in.ResolutionNotes,
		},
	}, nil
}

func (m *mockInternalBLLClient) AddTicketComment(ctx context.Context, in *opsv1.AddTicketCommentRequest) (*opsv1.AddTicketCommentResponse, error) {
	return &opsv1.AddTicketCommentResponse{
		Comment: &opsv1.TicketComment{
			CommentID:      "comm-test-1",
			TicketID:       in.TicketID,
			AuthorUsername: in.AuthorUsername,
			Comment:        in.Comment,
			IsInternal:     in.IsInternal,
		},
	}, nil
}

func (m *mockInternalBLLClient) ListTicketComments(ctx context.Context, in *opsv1.ListTicketCommentsRequest) (*opsv1.ListTicketCommentsResponse, error) {
	return &opsv1.ListTicketCommentsResponse{
		Comments: []*opsv1.TicketComment{
			{
				CommentID:      "comm-test-1",
				TicketID:       in.TicketID,
				AuthorUsername: "emp-human-elise",
				Comment:        "Sample comment",
				IsInternal:     false,
			},
		},
	}, nil
}

func (m *mockInternalBLLClient) CreateCustomerTicket(ctx context.Context, in *opsv1.CreateCustomerTicketRequest) (*opsv1.CreateCustomerTicketResponse, error) {
	return &opsv1.CreateCustomerTicketResponse{
		Ticket: &opsv1.ITTicket{
			TicketID:       "cust-ticket-999",
			Title:          in.Title,
			Source:         "customer",
			CustomerID:     in.CustomerID,
			AuthorUsername: in.CustomerID,
			State:          "open",
		},
	}, nil
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

func TestOpsHandler(t *testing.T) {
	mockBLL := &mockInternalBLLClient{}
	handler := NewHandler(mockBLL)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	t.Run("GET /api/v1/ops/pebble/alerts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ops/pebble/alerts", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var alerts []*opsv1.PebbleAlert
		err := json.Unmarshal(rec.Body.Bytes(), &alerts)
		assert.NoError(t, err)
		assert.Len(t, alerts, 1)
		assert.Equal(t, "ticket-test-123", alerts[0].TicketID)
	})

	t.Run("POST /api/v1/ops/pebble/ack", func(t *testing.T) {
		payload := pebble.PebbleAckRequest{
			TicketID:       "ticket-test-123",
			AcknowledgedBy: "emp-human-frank",
			ButtonID:       1,
		}
		data, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/ops/pebble/ack", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp opsv1.AcknowledgeAlertResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "ticket-test-123", resp.AlertID)
		assert.Equal(t, "acknowledged", resp.State)
	})

	t.Run("GET /api/v1/ops/tickets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ops/tickets?limit=10&offset=0", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, int32(10), mockBLL.lastLimit)

		var result map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.NotNil(t, result["items"])
	})

	t.Run("GET /api/v1/ops/tickets Limit Capped", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ops/tickets?limit=500&offset=0", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, int32(100), mockBLL.lastLimit)
	})

	t.Run("POST /api/v1/ops/tickets", func(t *testing.T) {
		body := map[string]any{
			"forgejo_repo":    "infra/microk8s",
			"title":           "Node Out of Memory",
			"body":            "MicroK8s worker node 2 OOM",
			"author_username": "emp-human-frank",
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/ops/tickets", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var ticket opsv1.ITTicket
		err := json.Unmarshal(rec.Body.Bytes(), &ticket)
		assert.NoError(t, err)
		assert.Equal(t, "created-ticket-123", ticket.TicketID)
	})

	t.Run("POST /api/v1/ops/notifications", func(t *testing.T) {
		body := map[string]any{
			"topic":    "alerts",
			"title":    "MicroK8s Node High Memory",
			"message":  "Node memory utilization at 92%",
			"priority": "high",
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/ops/notifications", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp opsv1.SendPushNotificationResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "dispatched", resp.Status)
		assert.Equal(t, "ntfy", resp.Provider)
		assert.Equal(t, "alerts", resp.Topic)
	})

	t.Run("GET & PUT /api/v1/ops/tickets/{id}", func(t *testing.T) {
		// GET ticket
		req := httptest.NewRequest("GET", "/api/v1/ops/tickets/ticket-test-123", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var ticket opsv1.ITTicket
		err := json.Unmarshal(rec.Body.Bytes(), &ticket)
		assert.NoError(t, err)
		assert.Equal(t, "ticket-test-123", ticket.TicketID)

		// PUT ticket update
		updateBody := map[string]any{
			"state":             "resolved",
			"assignee_username": "emp-human-bob",
			"resolution_notes":  "Restarted habitat temperature gateway",
		}
		updateData, _ := json.Marshal(updateBody)
		req2 := httptest.NewRequest("PUT", "/api/v1/ops/tickets/ticket-test-123", bytes.NewReader(updateData))
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusOK, rec2.Code)
		var updated opsv1.ITTicket
		err = json.Unmarshal(rec2.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.Equal(t, "resolved", updated.State)
	})

	t.Run("Comments Threading on IT Tickets", func(t *testing.T) {
		// POST Comment
		commBody := map[string]any{
			"author_username": "emp-human-frank",
			"comment":         "Sensor firmware updated to v2.1.0",
			"is_internal":     true,
		}
		data, _ := json.Marshal(commBody)
		req := httptest.NewRequest("POST", "/api/v1/ops/tickets/ticket-test-123/comments", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		// GET Comments
		req2 := httptest.NewRequest("GET", "/api/v1/ops/tickets/ticket-test-123/comments", nil)
		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusOK, rec2.Code)
		var listResp map[string]any
		err := json.Unmarshal(rec2.Body.Bytes(), &listResp)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), listResp["total"])
	})

	t.Run("Customer Support Intake & Comments", func(t *testing.T) {
		// POST Customer Ticket
		custBody := map[string]any{
			"customer_id": "cust-longterm-arthur",
			"category":    "trading",
			"title":       "Order settlement confirmation inquiry",
			"body":        "Please check if transaction batch #412 settled.",
		}
		data, _ := json.Marshal(custBody)
		req := httptest.NewRequest("POST", "/api/v1/customer/support/tickets", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Customer-ID", "cust-longterm-arthur")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var created opsv1.ITTicket
		err := json.Unmarshal(rec.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.Equal(t, "cust-ticket-999", created.TicketID)

		// GET Customer Tickets
		reqList := httptest.NewRequest("GET", "/api/v1/customer/support/tickets", nil)
		reqList.Header.Set("X-Customer-ID", "cust-longterm-arthur")
		recList := httptest.NewRecorder()
		mux.ServeHTTP(recList, reqList)

		assert.Equal(t, http.StatusOK, recList.Code)

		// GET Customer Ticket Comments (Scrubbed)
		reqComm := httptest.NewRequest("GET", "/api/v1/customer/support/tickets/cust-ticket-999/comments", nil)
		recComm := httptest.NewRecorder()
		mux.ServeHTTP(recComm, reqComm)

		assert.Equal(t, http.StatusOK, recComm.Code)

		// POST Customer Reply Comment
		replyBody := map[string]any{
			"comment": "Thank you for looking into this!",
		}
		dataReply, _ := json.Marshal(replyBody)
		reqReply := httptest.NewRequest("POST", "/api/v1/customer/support/tickets/cust-ticket-999/comments", bytes.NewReader(dataReply))
		reqReply.Header.Set("Content-Type", "application/json")
		reqReply.Header.Set("X-Customer-ID", "cust-longterm-arthur")
		recReply := httptest.NewRecorder()
		mux.ServeHTTP(recReply, reqReply)

		assert.Equal(t, http.StatusCreated, recReply.Code)
	})
}
