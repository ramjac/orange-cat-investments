package ops

import (
	"context"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/ops"
	"github.com/stretchr/testify/assert"
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
		ticket, err := service.CreateITTicket(ctx, "infra/k3s", "Node Unreachable", "Worker node 1 down", "emp-human-frank", nil)
		assert.NoError(t, err)
		assert.Equal(t, "created-mock-id", ticket.TicketID)

		_, errInvalid := service.CreateITTicket(ctx, "infra/k3s", "", "", "", nil)
		assert.Error(t, errInvalid)
	})

	t.Run("GetITTicket", func(t *testing.T) {
		ticket, err := service.GetITTicket(ctx, "ticket-123")
		assert.NoError(t, err)
		assert.Equal(t, "ticket-123", ticket.TicketID)

		_, errEmpty := service.GetITTicket(ctx, "")
		assert.Error(t, errEmpty)
	})
}
