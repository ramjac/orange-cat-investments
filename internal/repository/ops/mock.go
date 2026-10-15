package ops

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type mockRepository struct {
	mu      sync.RWMutex
	tickets map[string]*ITTicket
}

// NewMockRepository returns an in-memory mock implementation of ops.Repository for testing and local development.
func NewMockRepository() Repository {
	repo := &mockRepository{
		tickets: make(map[string]*ITTicket),
	}
	repo.seedDefaults()
	return repo
}

func (r *mockRepository) seedDefaults() {
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

func (r *mockRepository) CreateITTicket(ctx context.Context, ticket *ITTicket) (*ITTicket, error) {
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
