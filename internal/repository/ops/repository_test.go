package ops

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

type mockErrDB struct {
	err error
}

func (m *mockErrDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &mockRow{err: m.err}
}

func (m *mockErrDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, m.err
}

type mockRow struct {
	err error
}

func (r *mockRow) Scan(dest ...any) error {
	return r.err
}

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

func TestOpsPgxRepositoryDBErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("AcknowledgeITTicket - pgx.ErrNoRows", func(t *testing.T) {
		noRowsRepo := NewRepository(&mockErrDB{err: pgx.ErrNoRows})
		ack, err := noRowsRepo.AcknowledgeITTicket(ctx, "non-existent-id", "emp-human-frank")
		assert.Error(t, err)
		assert.Nil(t, ack)
		assert.EqualError(t, err, "ticket not found or already acknowledged/closed: non-existent-id")
	})

	t.Run("AcknowledgeITTicket - General DB Error", func(t *testing.T) {
		generalErr := errors.New("db connection failure")
		errRepo := NewRepository(&mockErrDB{err: generalErr})
		ack, err := errRepo.AcknowledgeITTicket(ctx, "ticket-001", "emp-human-frank")
		assert.Error(t, err)
		assert.Nil(t, ack)
		assert.Equal(t, generalErr, err)
	})

	t.Run("GetITTicketByID - pgx.ErrNoRows", func(t *testing.T) {
		noRowsRepo := NewRepository(&mockErrDB{err: pgx.ErrNoRows})
		ticket, err := noRowsRepo.GetITTicketByID(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, ticket)
		assert.EqualError(t, err, "ticket not found: non-existent-id")
	})

	t.Run("CloseITTicket - pgx.ErrNoRows", func(t *testing.T) {
		noRowsRepo := NewRepository(&mockErrDB{err: pgx.ErrNoRows})
		ticket, err := noRowsRepo.CloseITTicket(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, ticket)
		assert.EqualError(t, err, "ticket not found or already closed: non-existent-id")
	})
}
