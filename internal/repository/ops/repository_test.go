package ops

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpsPgxRepository(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	t.Run("GetITTicketByID", func(t *testing.T) {
		ticket, err := repo.GetITTicketByID(ctx, "ticket-001")
		assert.NoError(t, err)
		assert.NotNil(t, ticket)
		assert.Equal(t, "ticket-001", ticket.TicketID)
		assert.Equal(t, "open", ticket.State)
	})

	t.Run("ListOpenITTickets", func(t *testing.T) {
		tickets, err := repo.ListOpenITTickets(ctx, 10, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, tickets)
		assert.Equal(t, "open", tickets[0].State)
	})

	t.Run("ListITTickets", func(t *testing.T) {
		tickets, err := repo.ListITTickets(ctx, 10, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, tickets)
	})

	t.Run("CreateITTicket", func(t *testing.T) {
		body := "Disk space low"
		ticket := &ITTicket{
			ForgejoRepo:    "infra/k3s-cluster",
			Title:          "Disk Space Warning",
			Body:           &body,
			AuthorUsername: "emp-human-frank",
		}

		created, err := repo.CreateITTicket(ctx, ticket)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.TicketID)
		assert.Equal(t, "open", created.State)
	})

	t.Run("AcknowledgeITTicket", func(t *testing.T) {
		ack, err := repo.AcknowledgeITTicket(ctx, "ticket-001", "emp-human-frank")
		assert.NoError(t, err)
		assert.Equal(t, "acknowledged", ack.State)
		assert.NotNil(t, ack.AcknowledgedBy)
		assert.Equal(t, "emp-human-frank", *ack.AcknowledgedBy)
		assert.NotNil(t, ack.AcknowledgedAt)
	})

	t.Run("CloseITTicket", func(t *testing.T) {
		closed, err := repo.CloseITTicket(ctx, "ticket-001")
		assert.NoError(t, err)
		assert.Equal(t, "closed", closed.State)
	})
}
