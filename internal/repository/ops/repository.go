package ops

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type ITTicket struct {
	TicketID         string     `json:"ticket_id"`
	ForgejoIssueID   *int64     `json:"forgejo_issue_id,omitempty"`
	ForgejoRepo      string     `json:"forgejo_repo"`
	Title            string     `json:"title"`
	Body             *string    `json:"body,omitempty"`
	State            string     `json:"state"`
	Severity         string     `json:"severity"`
	Category         string     `json:"category"`
	Source           string     `json:"source"`
	CustomerID       *string    `json:"customer_id,omitempty"`
	AssigneeUsername *string    `json:"assignee_username,omitempty"`
	AuthorUsername   string     `json:"author_username"`
	ResolutionNotes  *string    `json:"resolution_notes,omitempty"`
	IsUrgent         bool       `json:"is_urgent"`
	AcknowledgedAt   *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy   *string    `json:"acknowledged_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type TicketComment struct {
	CommentID      string    `json:"comment_id"`
	TicketID       string    `json:"ticket_id"`
	AuthorUsername string    `json:"author_username"`
	Comment        string    `json:"comment"`
	IsInternal     bool      `json:"is_internal"`
	CreatedAt      time.Time `json:"created_at"`
}

type Repository interface {
	GetITTicketByID(ctx context.Context, ticketID string) (*ITTicket, error)
	ListOpenITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error)
	ListITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error)
	ListITTicketsBySource(ctx context.Context, source string, limit, offset int32) ([]*ITTicket, error)
	ListITTicketsByCustomer(ctx context.Context, customerID string, limit, offset int32) ([]*ITTicket, error)
	CreateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error)
	UpdateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error)
	AcknowledgeITTicket(ctx context.Context, ticketID, acknowledgedBy string) (*ITTicket, error)
	CloseITTicket(ctx context.Context, ticketID string) (*ITTicket, error)
	CreateTicketComment(ctx context.Context, comment *TicketComment) (*TicketComment, error)
	ListTicketComments(ctx context.Context, ticketID string, includeInternal bool) ([]*TicketComment, error)
}

type DBExecutor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type pgxRepository struct {
	db DBExecutor
}

// NewRepository returns an ops.Repository backed by PostgreSQL. If db is nil, it delegates to NewMockRepository().
func NewRepository(db DBExecutor) Repository {
	if db == nil {
		return NewMockRepository()
	}
	return &pgxRepository{db: db}
}

const ticketColumns = `
	ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent, acknowledged_at, acknowledged_by, created_at, updated_at
`

func scanITTicket(row interface{ Scan(dest ...any) error }) (*ITTicket, error) {
	var t ITTicket
	err := row.Scan(
		&t.TicketID, &t.ForgejoIssueID, &t.ForgejoRepo, &t.Title, &t.Body,
		&t.State, &t.Severity, &t.Category, &t.Source, &t.CustomerID,
		&t.AssigneeUsername, &t.AuthorUsername, &t.ResolutionNotes, &t.IsUrgent,
		&t.AcknowledgedAt, &t.AcknowledgedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pgxRepository) GetITTicketByID(ctx context.Context, ticketID string) (*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		SELECT %s
		FROM ops.it_tickets
		WHERE ticket_id = $1`, ticketColumns)
	row := r.db.QueryRow(ctx, query, ticketID)
	t, err := scanITTicket(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ticket not found: %s", ticketID)
		}
		return nil, err
	}
	return t, nil
}

func (r *pgxRepository) ListOpenITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		SELECT %s
		FROM ops.it_tickets
		WHERE state = 'open'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, ticketColumns)
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*ITTicket
	for rows.Next() {
		t, err := scanITTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *pgxRepository) ListITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		SELECT %s
		FROM ops.it_tickets
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, ticketColumns)
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*ITTicket
	for rows.Next() {
		t, err := scanITTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *pgxRepository) ListITTicketsBySource(ctx context.Context, source string, limit, offset int32) ([]*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		SELECT %s
		FROM ops.it_tickets
		WHERE source = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, ticketColumns)
	rows, err := r.db.Query(ctx, query, source, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*ITTicket
	for rows.Next() {
		t, err := scanITTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *pgxRepository) ListITTicketsByCustomer(ctx context.Context, customerID string, limit, offset int32) ([]*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		SELECT %s
		FROM ops.it_tickets
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, ticketColumns)
	rows, err := r.db.Query(ctx, query, customerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*ITTicket
	for rows.Next() {
		t, err := scanITTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *pgxRepository) CreateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if ticket.Severity == "" {
		ticket.Severity = "medium"
	}
	if ticket.Category == "" {
		ticket.Category = "software"
	}
	if ticket.Source == "" {
		ticket.Source = "internal"
	}
	if ticket.ForgejoRepo == "" {
		if ticket.Source == "customer" {
			ticket.ForgejoRepo = "oci-internal/customer-support"
		} else {
			ticket.ForgejoRepo = "oci-internal/helpdesk"
		}
	}

	query := fmt.Sprintf(`
		INSERT INTO ops.it_tickets (
			forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, resolution_notes, is_urgent
		) VALUES (
			$1, $2, $3, $4, 'open', $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING %s`, ticketColumns)
	row := r.db.QueryRow(ctx, query,
		ticket.ForgejoIssueID, ticket.ForgejoRepo, ticket.Title, ticket.Body,
		ticket.Severity, ticket.Category, ticket.Source, ticket.CustomerID,
		ticket.AssigneeUsername, ticket.AuthorUsername, ticket.ResolutionNotes, ticket.IsUrgent,
	)
	t, err := scanITTicket(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *pgxRepository) UpdateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		UPDATE ops.it_tickets
		SET state = COALESCE($2, state),
			assignee_username = COALESCE($3, assignee_username),
			resolution_notes = COALESCE($4, resolution_notes),
			updated_at = CURRENT_TIMESTAMP
		WHERE ticket_id = $1
		RETURNING %s`, ticketColumns)
	row := r.db.QueryRow(ctx, query,
		ticket.TicketID, ticket.State, ticket.AssigneeUsername, ticket.ResolutionNotes,
	)
	t, err := scanITTicket(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ticket not found: %s", ticket.TicketID)
		}
		return nil, err
	}
	return t, nil
}

func (r *pgxRepository) AcknowledgeITTicket(ctx context.Context, ticketID, acknowledgedBy string) (*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		UPDATE ops.it_tickets
		SET state = 'acknowledged',
			acknowledged_at = CURRENT_TIMESTAMP,
			acknowledged_by = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE ticket_id = $1
		RETURNING %s`, ticketColumns)
	row := r.db.QueryRow(ctx, query, ticketID, acknowledgedBy)
	t, err := scanITTicket(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ticket not found or already acknowledged/closed: %s", ticketID)
		}
		return nil, err
	}
	return t, nil
}

func (r *pgxRepository) CloseITTicket(ctx context.Context, ticketID string) (*ITTicket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := fmt.Sprintf(`
		UPDATE ops.it_tickets
		SET state = 'closed',
			updated_at = CURRENT_TIMESTAMP
		WHERE ticket_id = $1
		RETURNING %s`, ticketColumns)
	row := r.db.QueryRow(ctx, query, ticketID)
	t, err := scanITTicket(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ticket not found or already closed: %s", ticketID)
		}
		return nil, err
	}
	return t, nil
}

func (r *pgxRepository) CreateTicketComment(ctx context.Context, comment *TicketComment) (*TicketComment, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO ops.ticket_comments (
			ticket_id, author_username, comment, is_internal
		) VALUES (
			$1, $2, $3, $4
		) RETURNING comment_id, ticket_id, author_username, comment, is_internal, created_at`,
		comment.TicketID, comment.AuthorUsername, comment.Comment, comment.IsInternal,
	)
	var c TicketComment
	err := row.Scan(&c.CommentID, &c.TicketID, &c.AuthorUsername, &c.Comment, &c.IsInternal, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *pgxRepository) ListTicketComments(ctx context.Context, ticketID string, includeInternal bool) ([]*TicketComment, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var rows pgx.Rows
	var err error
	if includeInternal {
		rows, err = r.db.Query(ctx, `
			SELECT comment_id, ticket_id, author_username, comment, is_internal, created_at
			FROM ops.ticket_comments
			WHERE ticket_id = $1
			ORDER BY created_at ASC`, ticketID)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT comment_id, ticket_id, author_username, comment, is_internal, created_at
			FROM ops.ticket_comments
			WHERE ticket_id = $1 AND is_internal = FALSE
			ORDER BY created_at ASC`, ticketID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*TicketComment
	for rows.Next() {
		var c TicketComment
		if err := rows.Scan(&c.CommentID, &c.TicketID, &c.AuthorUsername, &c.Comment, &c.IsInternal, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	return comments, rows.Err()
}
