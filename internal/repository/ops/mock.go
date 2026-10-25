package ops

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type mockRepository struct {
	mu       sync.RWMutex
	tickets  map[string]*ITTicket
	comments map[string][]*TicketComment
}

// NewMockRepository returns an in-memory mock implementation of ops.Repository for testing and local development.
func NewMockRepository() Repository {
	repo := &mockRepository{
		tickets:  make(map[string]*ITTicket),
		comments: make(map[string][]*TicketComment),
	}
	repo.seedDefaults()
	return repo
}

func (r *mockRepository) seedDefaults() {
	now := time.Now().UTC()
	body001 := "Sample IT ticket body"
	r.tickets["ticket-001"] = &ITTicket{
		TicketID:       "ticket-001",
		ForgejoRepo:    "infra/microk8s-cluster",
		Title:          "MicroK8s Node High Memory Warning",
		Body:           &body001,
		State:          "open",
		Severity:       "high",
		Category:       "hardware",
		Source:         "internal",
		AuthorUsername: "emp-human-frank",
		IsUrgent:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	bodyOpen001 := "High priority memory alert on microk8s worker node 2"
	r.tickets["ticket-uuid-open-001"] = &ITTicket{
		TicketID:       "ticket-uuid-open-001",
		ForgejoRepo:    "infra/microk8s-cluster",
		Title:          "High Memory Usage Alert",
		Body:           &bodyOpen001,
		State:          "open",
		Severity:       "critical",
		Category:       "hardware",
		Source:         "internal",
		AuthorUsername: "emp-human-frank",
		IsUrgent:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	bodyArthur := "Can support verify the settlement status of my latest NVDA recurring allocation order?"
	custIDArthur := "cust-longterm-arthur"
	assigneeElise := "emp-human-elise"
	r.tickets["018e0000-0000-7000-8000-000000000005"] = &ITTicket{
		TicketID:         "018e0000-0000-7000-8000-000000000005",
		ForgejoRepo:      "oci-internal/customer-support",
		Title:            "Inquiry on Portfolio Trade Settlement Batch #412",
		Body:             &bodyArthur,
		State:            "in_progress",
		Severity:         "medium",
		Category:         "trading",
		Source:           "customer",
		CustomerID:       &custIDArthur,
		AssigneeUsername: &assigneeElise,
		AuthorUsername:   "cust-longterm-arthur",
		IsUrgent:         false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	r.comments["018e0000-0000-7000-8000-000000000005"] = []*TicketComment{
		{
			CommentID:      "018e0000-0000-7000-8000-000000000050",
			TicketID:       "018e0000-0000-7000-8000-000000000005",
			AuthorUsername: "emp-human-elise",
			Comment:        "INTERNAL ONLY: Reconciled clearing ledger with Mock Broker. Settlement batch matched successfully.",
			IsInternal:     true,
			CreatedAt:      now.Add(-10 * time.Minute),
		},
		{
			CommentID:      "018e0000-0000-7000-8000-000000000051",
			TicketID:       "018e0000-0000-7000-8000-000000000005",
			AuthorUsername: "emp-human-elise",
			Comment:        "Hello Arthur! Your NVDA order settlement has been verified and fully credited to your OCI Alpha Growth Fund portfolio.",
			IsInternal:     false,
			CreatedAt:      now.Add(-5 * time.Minute),
		},
	}
}

func (r *mockRepository) GetITTicketByID(ctx context.Context, ticketID string) (*ITTicket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ticket, ok := r.tickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("ticket not found or already closed: %s", ticketID)
	}
	cp := *ticket
	return &cp, nil
}

func (r *mockRepository) ListOpenITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error) {
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

func (r *mockRepository) ListITTickets(ctx context.Context, limit, offset int32) ([]*ITTicket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ITTicket
	for _, t := range r.tickets {
		cp := *t
		tickets = append(tickets, &cp)
	}
	return tickets, nil
}

func (r *mockRepository) ListITTicketsBySource(ctx context.Context, source string, limit, offset int32) ([]*ITTicket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ITTicket
	for _, t := range r.tickets {
		if t.Source == source {
			cp := *t
			tickets = append(tickets, &cp)
		}
	}
	return tickets, nil
}

func (r *mockRepository) ListITTicketsByCustomer(ctx context.Context, customerID string, limit, offset int32) ([]*ITTicket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ITTicket
	for _, t := range r.tickets {
		if t.CustomerID != nil && *t.CustomerID == customerID {
			cp := *t
			tickets = append(tickets, &cp)
		}
	}
	return tickets, nil
}

func (r *mockRepository) CreateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if ticket.TicketID == "" {
		ticket.TicketID = fmt.Sprintf("ticket-%d", time.Now().UnixNano())
	}
	if ticket.State == "" {
		ticket.State = "open"
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
	if ticket.Severity == "urgent" || ticket.Severity == "critical" {
		ticket.IsUrgent = true
	}
	if ticket.ForgejoRepo == "" {
		if ticket.Source == "customer" {
			ticket.ForgejoRepo = "oci-internal/customer-support"
		} else {
			ticket.ForgejoRepo = "oci-internal/helpdesk"
		}
	}
	ticket.CreatedAt = now
	ticket.UpdatedAt = now
	r.tickets[ticket.TicketID] = ticket
	cp := *ticket
	return &cp, nil
}

func (r *mockRepository) UpdateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.tickets[ticket.TicketID]
	if !ok {
		return nil, fmt.Errorf("ticket not found: %s", ticket.TicketID)
	}

	if ticket.State != "" {
		existing.State = ticket.State
	}
	if ticket.AssigneeUsername != nil {
		existing.AssigneeUsername = ticket.AssigneeUsername
	}
	if ticket.ResolutionNotes != nil {
		existing.ResolutionNotes = ticket.ResolutionNotes
	}
	existing.UpdatedAt = time.Now().UTC()
	cp := *existing
	return &cp, nil
}

func (r *mockRepository) AcknowledgeITTicket(ctx context.Context, ticketID, acknowledgedBy string) (*ITTicket, error) {
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

func (r *mockRepository) CloseITTicket(ctx context.Context, ticketID string) (*ITTicket, error) {
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

func (r *mockRepository) CreateTicketComment(ctx context.Context, comment *TicketComment) (*TicketComment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if comment.CommentID == "" {
		comment.CommentID = fmt.Sprintf("comm-%d", time.Now().UnixNano())
	}
	comment.CreatedAt = now

	cp := *comment
	r.comments[comment.TicketID] = append(r.comments[comment.TicketID], &cp)
	return &cp, nil
}

func (r *mockRepository) ListTicketComments(ctx context.Context, ticketID string, includeInternal bool) ([]*TicketComment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all, ok := r.comments[ticketID]
	if !ok {
		return []*TicketComment{}, nil
	}

	var result []*TicketComment
	for _, c := range all {
		if includeInternal || !c.IsInternal {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result, nil
}
