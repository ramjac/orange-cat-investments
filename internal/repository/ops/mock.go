package ops

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/orange-cat-investments/oci/pkg/outbox"
)

type mockRepository struct {
	mu          sync.RWMutex
	tickets     map[string]*ITTicket
	comments    map[string][]*TicketComment
	outboxStore *outbox.MemoryStore
	auditLogs   []*AuditLog
	nextAuditID int64
}

// NewMockRepository returns an in-memory mock implementation of ops.Repository for testing and local development.
func NewMockRepository() Repository {
	repo := &mockRepository{
		tickets:     make(map[string]*ITTicket),
		comments:    make(map[string][]*TicketComment),
		outboxStore: outbox.NewMemoryStore(outbox.SchemaOps),
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
	if ticket.IsUrgent && r.outboxStore != nil {
		if evID, raw, err := outbox.BuildEnvelopeBytes(ctx, "", envelope.TopicIncidentAlerted, cp.TicketID, cp); err == nil {
			r.outboxStore.Enqueue(evID, envelope.TopicIncidentAlerted, cp.TicketID, raw)
		}
	}
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

func (r *mockRepository) ScrubCustomerGDPRData(ctx context.Context, customerID string) (int, error) {
	if customerID == "" {
		return 0, fmt.Errorf("customer_id cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	redacted := "[Content redacted pursuant to GDPR Article 17 erasure request]"
	count := 0
	for _, t := range r.tickets {
		if t.AuthorUsername == customerID || (t.CustomerID != nil && *t.CustomerID == customerID) {
			t.AuthorUsername = "ANONYMIZED_GDPR_USER"
			t.CustomerID = nil
			t.Body = &redacted
			t.UpdatedAt = now
			count++
		}
	}
	for _, list := range r.comments {
		for _, c := range list {
			if c.AuthorUsername == customerID {
				c.AuthorUsername = "ANONYMIZED_GDPR_USER"
				c.Comment = redacted
				count++
			}
		}
	}
	return count, nil
}

func (r *mockRepository) ClaimOutboxEvents(ctx context.Context, limit int32, lease time.Duration) ([]*outbox.Record, error) {
	if r.outboxStore == nil {
		return []*outbox.Record{}, nil
	}
	return r.outboxStore.ClaimPending(limit, lease), nil
}

func (r *mockRepository) AckOutboxEvents(ctx context.Context, publishedIDs []int64, failures []outbox.Failure) (int, error) {
	if r.outboxStore == nil {
		return 0, nil
	}
	r.outboxStore.MarkPublished(publishedIDs)
	for _, f := range failures {
		r.outboxStore.MarkFailed(f.ID, f.Error, 0)
	}
	return len(publishedIDs) + len(failures), nil
}

func (r *mockRepository) AppendAuditLog(ctx context.Context, entry *AuditLog) (*AuditLog, error) {
	if entry == nil || entry.ActorID == "" || entry.Action == "" {
		return nil, fmt.Errorf("actor_id and action are required for audit logs")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextAuditID++
	now := time.Now().UTC().Truncate(time.Microsecond)
	prevHash := AuditGenesisHash
	if len(r.auditLogs) > 0 {
		prevHash = r.auditLogs[len(r.auditLogs)-1].RecordHash
	}
	meta := normalizeAuditMetadata(entry.MetadataJSON)
	canonicalTS := FormatCanonicalAuditTimestamp(now)
	recHash := ComputeAuditHash(
		prevHash,
		entry.ActorID,
		entry.ActorRole,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		meta,
		canonicalTS,
	)

	rec := &AuditLog{
		ID:           r.nextAuditID,
		ActorID:      entry.ActorID,
		ActorRole:    entry.ActorRole,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		MetadataJSON: meta,
		CreatedAt:    now,
		CanonicalTS:  canonicalTS,
		PreviousHash: prevHash,
		RecordHash:   recHash,
	}
	r.auditLogs = append(r.auditLogs, rec)
	cp := *rec
	return &cp, nil
}

func (r *mockRepository) VerifyAuditChain(ctx context.Context) (*AuditChainVerification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cp := make([]*AuditLog, len(r.auditLogs))
	for i, rec := range r.auditLogs {
		c := *rec
		cp[i] = &c
	}
	return verifyAuditRecords(cp), nil
}

func (r *mockRepository) ListAuditLogs(ctx context.Context, actorID, action string, limit, offset int32) ([]*AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var filtered []*AuditLog
	for i := len(r.auditLogs) - 1; i >= 0; i-- {
		rec := r.auditLogs[i]
		if actorID != "" && rec.ActorID != actorID {
			continue
		}
		if action != "" && rec.Action != action {
			continue
		}
		c := *rec
		filtered = append(filtered, &c)
	}
	if int(offset) >= len(filtered) {
		return []*AuditLog{}, nil
	}
	end := int(offset + limit)
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], nil
}

// TamperAuditLogForTest simulates out-of-band storage corruption for unit testing chain verification.
func TamperAuditLogForTest(repo Repository, id int64, mutatedMetadata string) bool {
	m, ok := repo.(*mockRepository)
	if !ok {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rec := range m.auditLogs {
		if rec.ID == id {
			rec.MetadataJSON = mutatedMetadata
			return true
		}
	}
	return false
}
