package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	opssvc "github.com/orange-cat-investments/oci/internal/service/ops"
	"github.com/stretchr/testify/assert"
)

type mockOpsService struct{}

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

func (m *mockOpsService) CreateITTicket(ctx context.Context, forgejoRepo, title, body, authorUsername string, forgejoIssueID *int64) (*opsrepo.ITTicket, error) {
	bodyPtr := &body
	return &opsrepo.ITTicket{
		TicketID:       "created-ticket-123",
		ForgejoRepo:    forgejoRepo,
		Title:          title,
		Body:           bodyPtr,
		State:          "open",
		AuthorUsername: authorUsername,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

func (m *mockOpsService) GetITTicket(ctx context.Context, ticketID string) (*opsrepo.ITTicket, error) {
	return &opsrepo.ITTicket{
		TicketID:       ticketID,
		ForgejoRepo:    "infra/k3s",
		Title:          "Sample Ticket",
		State:          "open",
		AuthorUsername: "emp-human-frank",
	}, nil
}

func (m *mockOpsService) ListITTickets(ctx context.Context, limit, offset int32) ([]*opsrepo.ITTicket, error) {
	return []*opsrepo.ITTicket{
		{
			TicketID:       "ticket-test-123",
			ForgejoRepo:    "infra/k3s",
			Title:          "Pod OOMKilled Warning",
			State:          "open",
			AuthorUsername: "emp-human-frank",
		},
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

		var result map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.NotNil(t, result["items"])
	})

	t.Run("POST /api/v1/ops/tickets", func(t *testing.T) {
		body := map[string]any{
			"forgejo_repo":    "infra/k3s",
			"title":           "Node Out of Memory",
			"body":            "K3s worker node 2 OOM",
			"author_username": "emp-human-frank",
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/ops/tickets", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var ticket opsrepo.ITTicket
		err := json.Unmarshal(rec.Body.Bytes(), &ticket)
		assert.NoError(t, err)
		assert.Equal(t, "created-ticket-123", ticket.TicketID)
	})
}
