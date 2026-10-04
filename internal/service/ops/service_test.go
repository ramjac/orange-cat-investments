package ops

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/ops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOpsRepo struct{}

func (m *mockOpsRepo) GetITTicketByID(ctx context.Context, ticketID string) (*ops.ITTicket, error) {
	body := "Mock ticket body"
	return &ops.ITTicket{
		TicketID:       ticketID,
		ForgejoRepo:    "infra/k3s",
		Title:          "Mock Title",
		Body:           &body,
		State:          "open",
		AuthorUsername: "emp-human-frank",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

func (m *mockOpsRepo) ListOpenITTickets(ctx context.Context, limit, offset int32) ([]*ops.ITTicket, error) {
	body := "Mock ticket body for open ticket"
	return []*ops.ITTicket{
		{
			TicketID:       "ticket-mock-123",
			ForgejoRepo:    "infra/k3s",
			Title:          "K3s Pod CrashLoopBackOff",
			Body:           &body,
			State:          "open",
			AuthorUsername: "emp-human-frank",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		},
	}, nil
}

func (m *mockOpsRepo) ListITTickets(ctx context.Context, limit, offset int32) ([]*ops.ITTicket, error) {
	return m.ListOpenITTickets(ctx, limit, offset)
}

func (m *mockOpsRepo) CreateITTicket(ctx context.Context, ticket *ops.ITTicket) (*ops.ITTicket, error) {
	ticket.TicketID = "created-mock-id"
	ticket.State = "open"
	return ticket, nil
}

func (m *mockOpsRepo) AcknowledgeITTicket(ctx context.Context, ticketID, acknowledgedBy string) (*ops.ITTicket, error) {
	now := time.Now().UTC()
	body := "Mock acknowledged ticket body"
	return &ops.ITTicket{
		TicketID:       ticketID,
		ForgejoRepo:    "infra/k3s",
		Title:          "K3s Pod CrashLoopBackOff",
		Body:           &body,
		State:          "acknowledged",
		AuthorUsername: "emp-human-frank",
		AcknowledgedAt: &now,
		AcknowledgedBy: &acknowledgedBy,
		CreatedAt:      now.Add(-10 * time.Minute),
		UpdatedAt:      now,
	}, nil
}

func (m *mockOpsRepo) CloseITTicket(ctx context.Context, ticketID string) (*ops.ITTicket, error) {
	return &ops.ITTicket{
		TicketID: ticketID,
		State:    "closed",
	}, nil
}

func TestOpsService(t *testing.T) {
	repo := &mockOpsRepo{}
	service := NewService(repo)
	ctx := context.Background()

	t.Run("GetPendingAlerts", func(t *testing.T) {
		alerts, err := service.GetPendingAlerts(ctx)
		assert.NoError(t, err)
		assert.Len(t, alerts, 1)
		assert.Equal(t, "ticket-mock-123", alerts[0].TicketID)
		assert.Equal(t, "K3s Pod CrashLoopBackOff", alerts[0].Title)
		assert.NotNil(t, alerts[0].AppMessageDict)
		assert.Equal(t, "ticket-mock-123", alerts[0].AppMessageDict[PebbleKeyTicketID])
	})

	t.Run("ProcessPebbleAck", func(t *testing.T) {
		req := &PebbleAckRequest{
			TicketID:       "ticket-mock-123",
			AcknowledgedBy: "emp-human-frank",
			ButtonID:       1,
		}

		res, err := service.ProcessPebbleAck(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "ticket-mock-123", res.TicketID)
		assert.Equal(t, "acknowledged", res.State)
		assert.Equal(t, "emp-human-frank", res.AcknowledgedBy)
		assert.Contains(t, res.StatusMessage, "acknowledged by emp-human-frank")
	})

	t.Run("ProcessPebbleAck Validation Error", func(t *testing.T) {
		_, err := service.ProcessPebbleAck(ctx, nil)
		assert.Error(t, err)

		_, err2 := service.ProcessPebbleAck(ctx, &PebbleAckRequest{TicketID: ""})
		assert.Error(t, err2)
	})

	t.Run("CreateITTicket", func(t *testing.T) {
		ticket, err := service.CreateITTicket(ctx, &CreateITTicketRequest{
			ForgejoRepo:    "infra/k3s",
			Title:          "Node Unreachable",
			Body:           "Worker node 1 down",
			AuthorUsername: "emp-human-frank",
		})
		assert.NoError(t, err)
		assert.Equal(t, "created-mock-id", ticket.TicketID)

		_, errNil := service.CreateITTicket(ctx, nil)
		assert.Error(t, errNil)

		_, errInvalid := service.CreateITTicket(ctx, &CreateITTicketRequest{
			ForgejoRepo:    "infra/k3s",
			Title:          "",
			Body:           "",
			AuthorUsername: "",
		})
		assert.Error(t, errInvalid)
	})

	t.Run("GetITTicket", func(t *testing.T) {
		ticket, err := service.GetITTicket(ctx, "ticket-123")
		assert.NoError(t, err)
		assert.Equal(t, "ticket-123", ticket.TicketID)

		_, errEmpty := service.GetITTicket(ctx, "")
		assert.Error(t, errEmpty)
	})

	t.Run("ListITTickets", func(t *testing.T) {
		tickets, err := service.ListITTickets(ctx, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, tickets, 1)
		assert.Equal(t, "ticket-mock-123", tickets[0].TicketID)
	})

	t.Run("SendPushNotification Validation Errors", func(t *testing.T) {
		_, err := service.SendPushNotification(ctx, nil)
		assert.EqualError(t, err, "request cannot be nil")

		_, errTopic := service.SendPushNotification(ctx, &PushNotificationRequest{
			Message: "Test Message",
		})
		assert.EqualError(t, errTopic, "notification topic is required")

		_, errMsg := service.SendPushNotification(ctx, &PushNotificationRequest{
			Topic: "alerts",
		})
		assert.EqualError(t, errMsg, "notification message is required")
	})

	t.Run("SendPushNotification Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/alerts", r.URL.Path)
			assert.Equal(t, "Critical Alert", r.Header.Get("Title"))
			assert.Equal(t, "high", r.Header.Get("Priority"))

			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, "Database down", string(body))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		origURL := os.Getenv("NTFY_URL")
		t.Setenv("NTFY_URL", server.URL)
		defer os.Setenv("NTFY_URL", origURL)

		req := &PushNotificationRequest{
			Topic:    "alerts",
			Title:    "Critical Alert",
			Message:  "Database down",
			Priority: "high",
		}

		res, err := service.SendPushNotification(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "dispatched", res.Status)
		assert.Equal(t, "ntfy", res.Provider)
		assert.Equal(t, "alerts", res.Topic)
		assert.Equal(t, "Critical Alert", res.Title)
		assert.Equal(t, "Database down", res.Message)
		assert.False(t, res.DispatchedAt.IsZero())
	})

	t.Run("SendPushNotification Provider Error (HTTP >= 400)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		t.Setenv("NTFY_URL", server.URL)

		req := &PushNotificationRequest{
			Topic:   "alerts",
			Message: "Database down",
		}

		res, err := service.SendPushNotification(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "notification provider returned status 500")
	})

	t.Run("SendPushNotification Network Failure Test Fallback Mode", func(t *testing.T) {
		t.Setenv("NTFY_URL", "http://127.0.0.1:0") // closed/invalid port
		t.Setenv("ENV", "test")

		req := &PushNotificationRequest{
			Topic:   "alerts",
			Message: "Test mock fallback",
		}

		res, err := service.SendPushNotification(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "dispatched", res.Status)
		assert.Equal(t, "mock-ntfy", res.Provider)
		assert.Equal(t, "alerts", res.Topic)
		assert.Equal(t, "Test mock fallback", res.Message)

		t.Setenv("ENV", "")
		t.Setenv("MOCK_NOTIFICATIONS", "true")

		res2, err2 := service.SendPushNotification(ctx, req)
		require.NoError(t, err2)
		require.NotNil(t, res2)
		assert.Equal(t, "mock-ntfy", res2.Provider)
	})

	t.Run("SendPushNotification Network Failure Production Mode", func(t *testing.T) {
		t.Setenv("NTFY_URL", "http://127.0.0.1:0")
		t.Setenv("ENV", "")
		t.Setenv("MOCK_NOTIFICATIONS", "")

		req := &PushNotificationRequest{
			Topic:   "alerts",
			Message: "Test failure",
		}

		res, err := service.SendPushNotification(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "failed to dispatch push notification via ntfy")
	})
}
