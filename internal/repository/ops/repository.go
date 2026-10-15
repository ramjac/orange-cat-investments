package ops

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type ITTicket struct {
	TicketID       string     `json:"ticket_id"`
	ForgejoIssueID *int64     `json:"forgejo_issue_id,omitempty"`
	ForgejoRepo    string     `json:"forgejo_repo"`
	Title          string     `json:"title"`
	Body           *string    `json:"body,omitempty"`
	State          string     `json:"state"`
	AuthorUsername string     `json:"author_username"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *string    `json:"acknowledged_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Repository interface {
	GetITTicketByID(ctx context.Context, ticketID string) (*ITTicket, error)
	ListOpenITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error)
	ListITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error)
	CreateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error)
	AcknowledgeITTicket(ctx context.Context, ticketID, acknowledgedBy string) (*ITTicket, error)
	CloseITTicket(ctx context.Context, ticketID string) (*ITTicket, error)
}

type DBExecutor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type pgxRepository struct {
	db      DBExecutor
	mu      sync.RWMutex
	tickets map[string]*ITTicket
}

func NewRepository(db DBExecutor) Repository {
	repo := &pgxRepository{
		db:      db,
		tickets: make(map[string]*ITTicket),
	}
	repo.seedDefaults()
	return repo
}

func (r *pgxRepository) seedDefaults() {
	now := time.Now().UTC()
	body001 := "Sample IT ticket body"
	r.tickets["ticket-001"] = &ITTicket{
		TicketID:       "ticket-001",
		ForgejoRepo:    "infra/k3s-cluster",
		Title:          "K3s Node High Memory Warning",
		Body:           &body001,
		State:          "open",
		AuthorUsername: "emp-human-frank",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	bodyOpen001 := "High priority memory alert on k3s worker node 2"
	r.tickets["ticket-uuid-open-001"] = &ITTicket{
		TicketID:       "ticket-uuid-open-001",
		ForgejoRepo:    "infra/k3s-cluster",
		Title:          "High Memory Usage Alert",
		Body:           &bodyOpen001,
		State:          "open",
		AuthorUsername: "emp-human-frank",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (r *pgxRepository) GetITTicketByID(ctx context.Context, ticketID string) (*ITTicket, error) {
	if r.db != nil {
		row := r.db.QueryRow(ctx, `
			SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at
			FROM ops.it_tickets
			WHERE ticket_id = $1`, ticketID)
		var t ITTicket
		err := row.Scan(
			&t.TicketID, &t.ForgejoIssueID, &t.ForgejoRepo, &t.Title, &t.Body,
			&t.State, &t.AuthorUsername, &t.AcknowledgedAt, &t.AcknowledgedBy,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("ticket not found: %s", ticketID)
			}
			return nil, err
		}
		return &t, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	ticket, ok := r.tickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("ticket not found or already closed: %s", ticketID)
	}
	cp := *ticket
	return &cp, nil
}

func (r *pgxRepository) ListOpenITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error) {
	if r.db != nil {
		rows, err := r.db.Query(ctx, `
			SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at
			FROM ops.it_tickets
			WHERE state = 'open'
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var tickets []*ITTicket
		for rows.Next() {
			var t ITTicket
			err := rows.Scan(
				&t.TicketID, &t.ForgejoIssueID, &t.ForgejoRepo, &t.Title, &t.Body,
				&t.State, &t.AuthorUsername, &t.AcknowledgedAt, &t.AcknowledgedBy,
				&t.CreatedAt, &t.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
			tickets = append(tickets, &t)
		}
		return tickets, rows.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ITTicket
	for _, t := range r.tickets {
		if t.State == "open" {
			cp := *t
			tickets = append(tickets, &cp)
		}
	}
	return tickets, nil
}

func (r *pgxRepository) ListITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error) {
	if r.db != nil {
		rows, err := r.db.Query(ctx, `
			SELECT ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at
			FROM ops.it_tickets
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var tickets []*ITTicket
		for rows.Next() {
			var t ITTicket
			err := rows.Scan(
				&t.TicketID, &t.ForgejoIssueID, &t.ForgejoRepo, &t.Title, &t.Body,
				&t.State, &t.AuthorUsername, &t.AcknowledgedAt, &t.AcknowledgedBy,
				&t.CreatedAt, &t.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
			tickets = append(tickets, &t)
		}
		return tickets, rows.Err()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ITTicket
	for _, t := range r.tickets {
		cp := *t
		tickets = append(tickets, &cp)
	}
	return tickets, nil
}

func (r *pgxRepository) CreateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error) {
	if r.db != nil {
		row := r.db.QueryRow(ctx, `
			INSERT INTO ops.it_tickets (
				forgejo_issue_id, forgejo_repo, title, body, state, author_username
			) VALUES (
				$1, $2, $3, $4, 'open', $5
			) RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at`,
			ticket.ForgejoIssueID, ticket.ForgejoRepo, ticket.Title, ticket.Body, ticket.AuthorUsername,
		)
		var t ITTicket
		err := row.Scan(
			&t.TicketID, &t.ForgejoIssueID, &t.ForgejoRepo, &t.Title, &t.Body,
			&t.State, &t.AuthorUsername, &t.AcknowledgedAt, &t.AcknowledgedBy,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if ticket.TicketID == "" {
		ticket.TicketID = "ticket-uuid-created"
	}
	ticket.State = "open"
	ticket.CreatedAt = now
	ticket.UpdatedAt = now
	r.tickets[ticket.TicketID] = ticket
	cp := *ticket
	return &cp, nil
}

func (r *pgxRepository) AcknowledgeITTicket(ctx context.Context, ticketID, acknowledgedBy string) (*ITTicket, error) {
	if r.db != nil {
		row := r.db.QueryRow(ctx, `
			UPDATE ops.it_tickets
			SET state = 'acknowledged',
				acknowledged_at = CURRENT_TIMESTAMP,
				acknowledged_by = $2,
				updated_at = CURRENT_TIMESTAMP
			WHERE ticket_id = $1
			RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at`,
			ticketID, acknowledgedBy,
		)
		var t ITTicket
		err := row.Scan(
			&t.TicketID, &t.ForgejoIssueID, &t.ForgejoRepo, &t.Title, &t.Body,
			&t.State, &t.AuthorUsername, &t.AcknowledgedAt, &t.AcknowledgedBy,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("ticket not found or already acknowledged/closed: %s", ticketID)
			}
			return nil, err
		}
		return &t, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ticket, ok := r.tickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("ticket not found or already closed: %s", ticketID)
	}

	now := time.Now().UTC()
	ticket.State = "acknowledged"
	ticket.AcknowledgedBy = &acknowledgedBy
	ticket.AcknowledgedAt = &now
	ticket.UpdatedAt = now
	cp := *ticket
	return &cp, nil
}

func (r *pgxRepository) CloseITTicket(ctx context.Context, ticketID string) (*ITTicket, error) {
	if r.db != nil {
		row := r.db.QueryRow(ctx, `
			UPDATE ops.it_tickets
			SET state = 'closed',
				updated_at = CURRENT_TIMESTAMP
			WHERE ticket_id = $1
			RETURNING ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, author_username, acknowledged_at, acknowledged_by, created_at, updated_at`,
			ticketID,
		)
		var t ITTicket
		err := row.Scan(
			&t.TicketID, &t.ForgejoIssueID, &t.ForgejoRepo, &t.Title, &t.Body,
			&t.State, &t.AuthorUsername, &t.AcknowledgedAt, &t.AcknowledgedBy,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("ticket not found or already closed: %s", ticketID)
			}
			return nil, err
		}
		return &t, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ticket, ok := r.tickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("ticket not found or already closed: %s", ticketID)
	}

	now := time.Now().UTC()
	ticket.State = "closed"
	ticket.UpdatedAt = now
	cp := *ticket
	return &cp, nil
}
