package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
	"github.com/stretchr/testify/assert"
)

type mockOpsService struct {
	lastLimit int32
}

func (m *mockOpsService) GetPendingAlerts(ctx context.Context) ([]*opssvc.PebbleAlertPayload, error) {
	return []*opssvc.PebbleAlertPayload{
		{
			TicketID:  "ticket-test-123",
			Title:     "Pod OOMKilled Warning",
			Priority:  "HIGH",
			Body:      "Memory usage exceeded threshold",
			Author:    "emp-human-frank",
			CreatedAt: time.Now().UTC(),
			AppMessageDict: map[uint32]string{
				opssvc.PebbleKeyCmd:      "1",
				opssvc.PebbleKeyTicketID: "ticket-test-123",
				opssvc.PebbleKeyTitle:    "Pod OOMKilled Warning",
			},
		},
	}, nil
}

func (m *mockOpsService) ProcessPebbleAck(ctx context.Context, req *opssvc.PebbleAckRequest) (*opssvc.PebbleAckResponse, error) {
	return &opssvc.PebbleAckResponse{
		TicketID:       req.TicketID,
		State:          "acknowledged",
		AcknowledgedAt: time.Now().UTC(),
		AcknowledgedBy: req.AcknowledgedBy,
		StatusMessage:  "Ticket acknowledged via Pebble Watch",
	}, nil
}

func (m *mockOpsService) CreateITTicket(ctx context.Context, req *opssvc.CreateITTicketRequest) (*opssvc.ITTicket, error) {
	bodyPtr := &req.Body
	return &opssvc.ITTicket{
		TicketID:       "created-ticket-123",
		ForgejoRepo:    req.ForgejoRepo,
		Title:          req.Title,
		Body:           bodyPtr,
		State:          "open",
		AuthorUsername: req.AuthorUsername,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

func (m *mockOpsService) GetITTicket(ctx context.Context, ticketID string) (*opssvc.ITTicket, error) {
	return &opssvc.ITTicket{
		TicketID:       ticketID,
		ForgejoRepo:    "infra/microk8s",
		Title:          "Sample Ticket",
		State:          "open",
		AuthorUsername: "emp-human-frank",
	}, nil
}

func (m *mockOpsService) ListITTickets(ctx context.Context, limit, offset int32) ([]*opssvc.ITTicket, error) {
	m.lastLimit = limit
	return []*opssvc.ITTicket{
		{
			TicketID:       "ticket-test-123",
			ForgejoRepo:    "infra/microk8s",
			Title:          "Pod OOMKilled Warning",
			State:          "open",
			AuthorUsername: "emp-human-frank",
		},
	}, nil
}

func (m *mockOpsService) ListITTicketsBySource(ctx context.Context, source string, limit, offset int32) ([]*opssvc.ITTicket, error) {
	return m.ListITTickets(ctx, limit, offset)
}

func (m *mockOpsService) ListITTicketsByCustomer(ctx context.Context, customerID string, limit, offset int32) ([]*opssvc.ITTicket, error) {
	return m.ListITTickets(ctx, limit, offset)
}

func (m *mockOpsService) UpdateITTicket(ctx context.Context, ticketID string, req *opssvc.UpdateTicketRequest) (*opssvc.ITTicket, error) {
	return &opssvc.ITTicket{
		TicketID: ticketID,
		State:    req.State,
	}, nil
}

func (m *mockOpsService) AddTicketComment(ctx context.Context, ticketID string, req *opssvc.AddCommentRequest) (*opssvc.TicketComment, error) {
	return &opssvc.TicketComment{
		CommentID:      "comm-test-1",
		TicketID:       ticketID,
		AuthorUsername: req.AuthorUsername,
		Comment:        req.Comment,
		IsInternal:     req.IsInternal,
	}, nil
}

func (m *mockOpsService) ListTicketComments(ctx context.Context, ticketID string, includeInternal bool) ([]*opssvc.TicketComment, error) {
	return []*opssvc.TicketComment{
		{
			CommentID:      "comm-test-1",
			TicketID:       ticketID,
			AuthorUsername: "emp-human-elise",
			Comment:        "Sample comment",
			IsInternal:     false,
		},
	}, nil
}

func (m *mockOpsService) CreateCustomerTicket(ctx context.Context, req *opssvc.CreateCustomerTicketRequest) (*opssvc.ITTicket, error) {
	return &opssvc.ITTicket{
		TicketID:       "cust-ticket-999",
		Title:          req.Title,
		Source:         "customer",
		CustomerID:     &req.CustomerID,
		AuthorUsername: req.CustomerID,
		State:          "open",
	}, nil
}

func (m *mockOpsService) SendPushNotification(ctx context.Context, req *opssvc.PushNotificationRequest) (*opssvc.PushNotificationResponse, error) {
	return &opssvc.PushNotificationResponse{
		Status:       "dispatched",
		Provider:     "ntfy",
		Topic:        req.Topic,
		Title:        req.Title,
		Message:      req.Message,
		DispatchedAt: time.Now().UTC(),
	}, nil
}

func TestOpsHandler(t *testing.T) {
	svc := &mockOpsService{}
	handler := NewHandler(svc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	t.Run("GET /api/v1/ops/pebble/alerts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ops/pebble/alerts", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var alerts []*opssvc.PebbleAlertPayload
		err := json.Unmarshal(rec.Body.Bytes(), &alerts)
		assert.NoError(t, err)
		assert.Len(t, alerts, 1)
		assert.Equal(t, "ticket-test-123", alerts[0].TicketID)
	})

	t.Run("POST /api/v1/ops/pebble/ack", func(t *testing.T) {
		payload := opssvc.PebbleAckRequest{
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

		var resp opssvc.PebbleAckResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "ticket-test-123", resp.TicketID)
		assert.Equal(t, "acknowledged", resp.State)
	})

	t.Run("GET /api/v1/ops/tickets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ops/tickets?limit=10&offset=0", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, int32(10), svc.lastLimit)

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
		assert.Equal(t, int32(100), svc.lastLimit)
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

		var ticket opssvc.ITTicket
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

		var resp opssvc.PushNotificationResponse
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
		var ticket opssvc.ITTicket
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
		var updated opssvc.ITTicket
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
		var created opssvc.ITTicket
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
