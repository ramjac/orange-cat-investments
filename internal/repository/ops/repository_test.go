package ops

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type mockCapturedDB struct {
	lastSQL  string
	lastArgs []any
}

func (m *mockCapturedDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	m.lastSQL = sql
	m.lastArgs = args
	return &mockRow{err: pgx.ErrNoRows}
}

func (m *mockCapturedDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	m.lastSQL = sql
	m.lastArgs = args
	return nil, nil
}

type mockRow struct {
	err error
}

func (r *mockRow) Scan(dest ...any) error {
	return r.err
}

func TestOpsPgxRepository(t *testing.T) {
	repo := NewMockRepository()
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
			ForgejoRepo:    "infra/microk8s-cluster",
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
		t.Run("Success Path", func(t *testing.T) {
			closed, err := repo.CloseITTicket(ctx, "ticket-001")
			assert.NoError(t, err)
			assert.Equal(t, "closed", closed.State)
		})

		t.Run("Errors for Non-Existent ID", func(t *testing.T) {
			closed, err := repo.CloseITTicket(ctx, "non-existent")
			assert.Error(t, err)
			assert.Nil(t, closed)
			assert.EqualError(t, err, "ticket not found or already closed: non-existent")
		})
	})

	t.Run("ListITTicketsBySource and Customer", func(t *testing.T) {
		internalTickets, err := repo.ListITTicketsBySource(ctx, "internal", 10, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, internalTickets)

		custTickets, err := repo.ListITTicketsByCustomer(ctx, "cust-longterm-arthur", 10, 0)
		assert.NoError(t, err)
		assert.Len(t, custTickets, 1)
		assert.Equal(t, "Inquiry on Portfolio Trade Settlement Batch #412", custTickets[0].Title)
	})

	t.Run("UpdateITTicket", func(t *testing.T) {
		assignee := "emp-human-bob"
		resNotes := "Perch thermostat rebooted and recalibrated."
		updated, err := repo.UpdateITTicket(ctx, &ITTicket{
			TicketID:         "018e0000-0000-7000-8000-000000000005",
			State:            "resolved",
			AssigneeUsername: &assignee,
			ResolutionNotes:  &resNotes,
		})
		assert.NoError(t, err)
		assert.Equal(t, "resolved", updated.State)
		assert.Equal(t, "emp-human-bob", *updated.AssigneeUsername)
	})

	t.Run("TicketComments and Scrubbing", func(t *testing.T) {
		ticketID := "018e0000-0000-7000-8000-000000000005"

		// Query all comments (internal staff view)
		staffComments, err := repo.ListTicketComments(ctx, ticketID, true)
		assert.NoError(t, err)
		assert.Len(t, staffComments, 2)

		// Query sanitized comments (customer view)
		custComments, err := repo.ListTicketComments(ctx, ticketID, false)
		assert.NoError(t, err)
		assert.Len(t, custComments, 1)
		assert.False(t, custComments[0].IsInternal)

		// Create a new comment
		newComment, err := repo.CreateTicketComment(ctx, &TicketComment{
			TicketID:       ticketID,
			AuthorUsername: "cust-longterm-arthur",
			Comment:        "Thank you for the quick confirmation!",
			IsInternal:     false,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, newComment.CommentID)

		// Query again sanitized
		custCommentsAfter, err := repo.ListTicketComments(ctx, ticketID, false)
		assert.NoError(t, err)
		assert.Len(t, custCommentsAfter, 2)
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

	t.Run("PgxRepository DB error handling", func(t *testing.T) {
		// Pass a closed pool or invalid pool to trigger DB query errors
		pool, err := pgxpool.New(ctx, "postgres://invalid:invalid@127.0.0.1:1/nonexistent?sslmode=disable")
		require.NoError(t, err)
		pool.Close()

		dbRepo := NewRepository(pool)

		_, err = dbRepo.ListOpenITTickets(ctx, 10, 0)
		assert.Error(t, err)

		_, err = dbRepo.GetITTicketByID(ctx, "ticket-001")
		assert.Error(t, err)

		_, err = dbRepo.ListITTickets(ctx, 10, 0)
		assert.Error(t, err)

		_, err = dbRepo.CreateITTicket(ctx, &ITTicket{Title: "Test"})
		assert.Error(t, err)

		_, err = dbRepo.AcknowledgeITTicket(ctx, "ticket-001", "user")
		assert.Error(t, err)

		_, err = dbRepo.CloseITTicket(ctx, "ticket-001")
		assert.Error(t, err)
	})

	t.Run("PgxRepository DB Error Paths", func(t *testing.T) {
		pool, _ := pgxpool.New(ctx, "postgres://invalid:invalid@127.0.0.1:54322/invalid?sslmode=disable")
		pgxRepo := NewRepository(pool)

		_, err := pgxRepo.GetITTicketByID(ctx, "ticket-001")
		assert.Error(t, err)

		_, err = pgxRepo.ListOpenITTickets(ctx, 10, 0)
		assert.Error(t, err)

		_, err = pgxRepo.ListITTickets(ctx, 10, 0)
		assert.Error(t, err)

		body := "Disk space low"
		_, err = pgxRepo.CreateITTicket(ctx, &ITTicket{
			ForgejoRepo:    "infra/microk8s-cluster",
			Title:          "Disk Space Warning",
			Body:           &body,
			AuthorUsername: "emp-human-frank",
		})
		assert.Error(t, err)

		_, err = pgxRepo.ListITTicketsBySource(ctx, "internal", 10, 0)
		assert.Error(t, err)

		_, err = pgxRepo.ListITTicketsByCustomer(ctx, "cust-1", 10, 0)
		assert.Error(t, err)

		_, err = pgxRepo.UpdateITTicket(ctx, &ITTicket{TicketID: "ticket-001"})
		assert.Error(t, err)

		_, err = pgxRepo.AcknowledgeITTicket(ctx, "ticket-001", "emp-human-frank")
		assert.Error(t, err)

		_, err = pgxRepo.CloseITTicket(ctx, "ticket-001")
		assert.Error(t, err)
	})
}

func TestOpsPgxRepositoryParameterizedQueries(t *testing.T) {
	ctx := context.Background()
	capturedDB := &mockCapturedDB{}
	repo := NewRepository(capturedDB)

	injectionPayloads := []string{
		"' OR '1'='1",
		"'; DROP TABLE ops.it_tickets; --",
		"admin'--",
		"ticket-001' UNION SELECT 1,2,3--",
	}

	for _, payload := range injectionPayloads {
		t.Run("AcknowledgeITTicket with payload: "+payload, func(t *testing.T) {
			_, err := repo.AcknowledgeITTicket(ctx, payload, payload)
			assert.Error(t, err) // Expect pgx.ErrNoRows wrapped error from mockRow
			assert.Equal(t, acknowledgeITTicketSQL, capturedDB.lastSQL)
			require.Len(t, capturedDB.lastArgs, 2)
			assert.Equal(t, payload, capturedDB.lastArgs[0])
			assert.Equal(t, payload, capturedDB.lastArgs[1])
		})

		t.Run("GetITTicketByID with payload: "+payload, func(t *testing.T) {
			_, err := repo.GetITTicketByID(ctx, payload)
			assert.Error(t, err)
			assert.Equal(t, getITTicketByIDSQL, capturedDB.lastSQL)
			require.Len(t, capturedDB.lastArgs, 1)
			assert.Equal(t, payload, capturedDB.lastArgs[0])
		})

		t.Run("CloseITTicket with payload: "+payload, func(t *testing.T) {
			_, err := repo.CloseITTicket(ctx, payload)
			assert.Error(t, err)
			assert.Equal(t, closeITTicketSQL, capturedDB.lastSQL)
			require.Len(t, capturedDB.lastArgs, 1)
			assert.Equal(t, payload, capturedDB.lastArgs[0])
		})
	}
}
