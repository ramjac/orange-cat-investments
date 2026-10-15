package ops

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/ops"
	"github.com/orange-cat-investments/oci/pkg/cots/forgejo"
)

// AppMessage Dictionary Key Constants for Pebble Watch C SDK
const (
	PebbleKeyCmd       uint32 = 0 // Command Type (1 = ALERT, 2 = ACK_CONFIRM)
	PebbleKeyTicketID  uint32 = 1 // Ticket ID UUID
	PebbleKeyTitle     uint32 = 2 // Alert Title
	PebbleKeyBody      uint32 = 3 // Alert Body / Details
	PebbleKeyPriority  uint32 = 4 // Alert Priority
	PebbleKeyAuthor    uint32 = 5 // Author / Source
	PebbleKeyTimestamp uint32 = 6 // UTC Timestamp String
)

type PebbleAlertPayload struct {
	TicketID       string            `json:"ticket_id"`
	Title          string            `json:"title"`
	Priority       string            `json:"priority"`
	Body           string            `json:"body"`
	Author         string            `json:"author"`
	CreatedAt      time.Time         `json:"created_at"`
	AppMessageDict map[uint32]string `json:"app_message_dict"`
}

type PebbleAckRequest struct {
	TicketID       string `json:"ticket_id"`
	AcknowledgedBy string `json:"acknowledged_by"`
	WatchID        string `json:"watch_id,omitempty"`
	DeviceID       string `json:"device_id,omitempty"`
	ButtonID       uint8  `json:"button_id,omitempty"` // 1 = SELECT (ACK)
}

type PebbleAckResponse struct {
	TicketID       string    `json:"ticket_id"`
	State          string    `json:"state"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
	AcknowledgedBy string    `json:"acknowledged_by"`
	StatusMessage  string    `json:"status_message"`
}

type PushNotificationRequest struct {
	Topic    string `json:"topic"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority string `json:"priority"`
}

type PushNotificationResponse struct {
	Status       string    `json:"status"`
	Provider     string    `json:"provider"`
	Topic        string    `json:"topic"`
	Title        string    `json:"title"`
	Message      string    `json:"message"`
	DispatchedAt time.Time `json:"dispatched_at"`
}

type CreateITTicketRequest struct {
	ForgejoRepo    string `json:"forgejo_repo,omitempty"`
	Title          string `json:"title"`
	Body           string `json:"body,omitempty"`
	AuthorUsername string `json:"author_username"`
	ForgejoIssueID *int64 `json:"forgejo_issue_id,omitempty"`
	Severity       string `json:"severity,omitempty"`
	Category       string `json:"category,omitempty"`
	Source         string `json:"source,omitempty"`
	CustomerID     string `json:"customer_id,omitempty"`
	IsUrgent       bool   `json:"is_urgent,omitempty"`
}

type UpdateTicketRequest struct {
	State            string `json:"state,omitempty"`
	AssigneeUsername string `json:"assignee_username,omitempty"`
	ResolutionNotes  string `json:"resolution_notes,omitempty"`
}

type AddCommentRequest struct {
	AuthorUsername string `json:"author_username"`
	Comment        string `json:"comment"`
	IsInternal     bool   `json:"is_internal"`
}

type CreateCustomerTicketRequest struct {
	CustomerID string `json:"customer_id"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	Body       string `json:"body"`
}

// IssueTrackerClient defines generic issue tracking operations bridging with Forgejo.
type IssueTrackerClient interface {
	CreateIssue(ctx context.Context, owner, repo string, opt forgejo.CreateIssueOption) (*forgejo.Issue, error)
	CreateIssueComment(ctx context.Context, owner, repo string, index int64, body string) (*forgejo.Comment, error)
	UpdateIssue(ctx context.Context, owner, repo string, index int64, opt forgejo.UpdateIssueOption) (*forgejo.Issue, error)
}

type Service interface {
	GetPendingAlerts(ctx context.Context) ([]*PebbleAlertPayload, error)
	ProcessPebbleAck(ctx context.Context, req *PebbleAckRequest) (*PebbleAckResponse, error)
	CreateITTicket(ctx context.Context, req *CreateITTicketRequest) (*ops.ITTicket, error)
	GetITTicket(ctx context.Context, ticketID string) (*ops.ITTicket, error)
	ListITTickets(ctx context.Context, limit, offset int32) ([]*ops.ITTicket, error)
	ListITTicketsBySource(ctx context.Context, source string, limit, offset int32) ([]*ops.ITTicket, error)
	ListITTicketsByCustomer(ctx context.Context, customerID string, limit, offset int32) ([]*ops.ITTicket, error)
	UpdateITTicket(ctx context.Context, ticketID string, req *UpdateTicketRequest) (*ops.ITTicket, error)
	AddTicketComment(ctx context.Context, ticketID string, req *AddCommentRequest) (*ops.TicketComment, error)
	ListTicketComments(ctx context.Context, ticketID string, includeInternal bool) ([]*ops.TicketComment, error)
	CreateCustomerTicket(ctx context.Context, req *CreateCustomerTicketRequest) (*ops.ITTicket, error)
	SendPushNotification(ctx context.Context, req *PushNotificationRequest) (*PushNotificationResponse, error)
}

type opsService struct {
	repo         ops.Repository
	issueTracker IssueTrackerClient
}

func NewService(repo ops.Repository) Service {
	return &opsService{repo: repo}
}

func NewServiceWithIssueTracker(repo ops.Repository, issueTracker IssueTrackerClient) Service {
	return &opsService{repo: repo, issueTracker: issueTracker}
}

func (s *opsService) GetPendingAlerts(ctx context.Context) ([]*PebbleAlertPayload, error) {
	tickets, err := s.repo.ListOpenITTickets(ctx, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list open IT tickets: %w", err)
	}

	payloads := make([]*PebbleAlertPayload, 0, len(tickets))
	for _, t := range tickets {
		bodyStr := ""
		if t.Body != nil {
			bodyStr = *t.Body
		}

		priority := "HIGH"
		if t.IsUrgent || t.Severity == "urgent" || t.Severity == "critical" {
			priority = "CRITICAL"
		} else if t.Severity == "low" {
			priority = "LOW"
		}

		dict := map[uint32]string{
			PebbleKeyCmd:       "1",
			PebbleKeyTicketID:  t.TicketID,
			PebbleKeyTitle:     t.Title,
			PebbleKeyBody:      bodyStr,
			PebbleKeyPriority:  priority,
			PebbleKeyAuthor:    t.AuthorUsername,
			PebbleKeyTimestamp: t.CreatedAt.UTC().Format(time.RFC3339),
		}

		payloads = append(payloads, &PebbleAlertPayload{
			TicketID:       t.TicketID,
			Title:          t.Title,
			Priority:       priority,
			Body:           bodyStr,
			Author:         t.AuthorUsername,
			CreatedAt:      t.CreatedAt,
			AppMessageDict: dict,
		})
	}

	return payloads, nil
}

func (s *opsService) ProcessPebbleAck(ctx context.Context, req *PebbleAckRequest) (*PebbleAckResponse, error) {
	if req == nil || req.TicketID == "" {
		return nil, errors.New("ticket_id is required for Pebble ACK")
	}

	ackBy := req.AcknowledgedBy
	if ackBy == "" {
		if empID, ok := ctx.Value("employee_id").(string); ok && empID != "" {
			ackBy = empID
		} else if req.WatchID != "" {
			ackBy = req.WatchID
		} else if req.DeviceID != "" {
			ackBy = req.DeviceID
		} else if onCall := os.Getenv("DEFAULT_ON_CALL_ENGINEER"); onCall != "" {
			ackBy = onCall
		} else if os.Getenv("ENABLE_DEV_MOCK_AUTH") == "true" {
			ackBy = "emp-oncall-engineer"
		} else {
			return nil, errors.New("acknowledged_by is required for Pebble ACK when paired device identity is unavailable")
		}
	}

	updatedTicket, err := s.repo.AcknowledgeITTicket(ctx, req.TicketID, ackBy)
	if err != nil {
		return nil, fmt.Errorf("failed to acknowledge IT ticket: %w", err)
	}

	ackTime := time.Now().UTC()
	if updatedTicket.AcknowledgedAt != nil {
		ackTime = *updatedTicket.AcknowledgedAt
	}

	return &PebbleAckResponse{
		TicketID:       updatedTicket.TicketID,
		State:          updatedTicket.State,
		AcknowledgedAt: ackTime,
		AcknowledgedBy: ackBy,
		StatusMessage:  fmt.Sprintf("Ticket %s acknowledged by %s via Pebble Watch companion app", req.TicketID, ackBy),
	}, nil
}

func (s *opsService) CreateITTicket(ctx context.Context, req *CreateITTicketRequest) (*ops.ITTicket, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.Title == "" || req.AuthorUsername == "" {
		return nil, errors.New("title and author_username are required")
	}

	bodyPtr := &req.Body
	if req.Body == "" {
		bodyPtr = nil
	}

	repoName := req.ForgejoRepo
	if repoName == "" {
		if req.Source == "customer" {
			repoName = "oci-internal/customer-support"
		} else {
			repoName = "oci-internal/helpdesk"
		}
	}

	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}
	category := req.Category
	if category == "" {
		category = "software"
	}
	source := req.Source
	if source == "" {
		source = "internal"
	}

	isUrgent := req.IsUrgent || severity == "urgent" || severity == "critical"

	var custIDPtr *string
	if req.CustomerID != "" {
		custIDPtr = &req.CustomerID
	}

	// Forgejo Issue Bridge synchronization
	issueID := req.ForgejoIssueID
	if s.issueTracker != nil && issueID == nil {
		parts := strings.Split(repoName, "/")
		owner := "oci-internal"
		rName := repoName
		if len(parts) == 2 {
			owner = parts[0]
			rName = parts[1]
		}
		issue, err := s.issueTracker.CreateIssue(ctx, owner, rName, forgejo.CreateIssueOption{
			Title: req.Title,
			Body:  req.Body,
		})
		if err == nil && issue != nil {
			issueID = &issue.Index
		}
	}

	ticket := &ops.ITTicket{
		ForgejoIssueID: issueID,
		ForgejoRepo:    repoName,
		Title:          req.Title,
		Body:           bodyPtr,
		State:          "open",
		Severity:       severity,
		Category:       category,
		Source:         source,
		CustomerID:     custIDPtr,
		AuthorUsername: req.AuthorUsername,
		IsUrgent:       isUrgent,
	}

	created, err := s.repo.CreateITTicket(ctx, ticket)
	if err != nil {
		return nil, err
	}

	// Route urgent priority tickets immediately to on-call push notification channel
	if isUrgent {
		_, _ = s.SendPushNotification(ctx, &PushNotificationRequest{
			Topic:    "oncall-alerts",
			Title:    fmt.Sprintf("URGENT TICKET [%s]: %s", strings.ToUpper(category), req.Title),
			Message:  req.Body,
			Priority: "urgent",
		})
	}

	return created, nil
}

func (s *opsService) GetITTicket(ctx context.Context, ticketID string) (*ops.ITTicket, error) {
	if ticketID == "" {
		return nil, errors.New("ticket_id cannot be empty")
	}
	return s.repo.GetITTicketByID(ctx, ticketID)
}

func (s *opsService) ListITTickets(ctx context.Context, limit, offset int32) ([]*ops.ITTicket, error) {
	return s.repo.ListITTickets(ctx, limit, offset)
}

func (s *opsService) ListITTicketsBySource(ctx context.Context, source string, limit, offset int32) ([]*ops.ITTicket, error) {
	return s.repo.ListITTicketsBySource(ctx, source, limit, offset)
}

func (s *opsService) ListITTicketsByCustomer(ctx context.Context, customerID string, limit, offset int32) ([]*ops.ITTicket, error) {
	return s.repo.ListITTicketsByCustomer(ctx, customerID, limit, offset)
}

func (s *opsService) UpdateITTicket(ctx context.Context, ticketID string, req *UpdateTicketRequest) (*ops.ITTicket, error) {
	if ticketID == "" {
		return nil, errors.New("ticket_id cannot be empty")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	var assigneePtr *string
	if req.AssigneeUsername != "" {
		assigneePtr = &req.AssigneeUsername
	}
	var resNotesPtr *string
	if req.ResolutionNotes != "" {
		resNotesPtr = &req.ResolutionNotes
	}

	ticket := &ops.ITTicket{
		TicketID:         ticketID,
		State:            req.State,
		AssigneeUsername: assigneePtr,
		ResolutionNotes:  resNotesPtr,
	}

	updated, err := s.repo.UpdateITTicket(ctx, ticket)
	if err != nil {
		return nil, err
	}

	// If closing the ticket, sync state with Forgejo issue if configured
	if s.issueTracker != nil && updated.ForgejoIssueID != nil && req.State == "closed" {
		parts := strings.Split(updated.ForgejoRepo, "/")
		owner := "oci-internal"
		rName := updated.ForgejoRepo
		if len(parts) == 2 {
			owner = parts[0]
			rName = parts[1]
		}
		closedState := "closed"
		_, _ = s.issueTracker.UpdateIssue(ctx, owner, rName, *updated.ForgejoIssueID, forgejo.UpdateIssueOption{
			State: &closedState,
		})
	}

	return updated, nil
}

func (s *opsService) AddTicketComment(ctx context.Context, ticketID string, req *AddCommentRequest) (*ops.TicketComment, error) {
	if ticketID == "" {
		return nil, errors.New("ticket_id cannot be empty")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.Comment == "" {
		return nil, errors.New("comment text is required")
	}
	if req.AuthorUsername == "" {
		return nil, errors.New("author_username is required")
	}

	// Fetch existing ticket to check for Forgejo synchronization
	ticket, err := s.repo.GetITTicketByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	// If Forgejo Issue is linked and comment is non-internal, forward to Forgejo
	if s.issueTracker != nil && ticket.ForgejoIssueID != nil && !req.IsInternal {
		parts := strings.Split(ticket.ForgejoRepo, "/")
		owner := "oci-internal"
		rName := ticket.ForgejoRepo
		if len(parts) == 2 {
			owner = parts[0]
			rName = parts[1]
		}
		_, _ = s.issueTracker.CreateIssueComment(ctx, owner, rName, *ticket.ForgejoIssueID, req.Comment)
	}

	comment := &ops.TicketComment{
		TicketID:       ticketID,
		AuthorUsername: req.AuthorUsername,
		Comment:        req.Comment,
		IsInternal:     req.IsInternal,
	}

	return s.repo.CreateTicketComment(ctx, comment)
}

func (s *opsService) ListTicketComments(ctx context.Context, ticketID string, includeInternal bool) ([]*ops.TicketComment, error) {
	if ticketID == "" {
		return nil, errors.New("ticket_id cannot be empty")
	}
	return s.repo.ListTicketComments(ctx, ticketID, includeInternal)
}

func (s *opsService) CreateCustomerTicket(ctx context.Context, req *CreateCustomerTicketRequest) (*ops.ITTicket, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.CustomerID == "" || req.Title == "" {
		return nil, errors.New("customer_id and title are required")
	}

	category := req.Category
	if category == "" {
		category = "general"
	}

	return s.CreateITTicket(ctx, &CreateITTicketRequest{
		ForgejoRepo:    "oci-internal/customer-support",
		Title:          req.Title,
		Body:           req.Body,
		AuthorUsername: req.CustomerID,
		Severity:       "medium",
		Category:       category,
		Source:         "customer",
		CustomerID:     req.CustomerID,
		IsUrgent:       false,
	})
}

func (s *opsService) SendPushNotification(ctx context.Context, req *PushNotificationRequest) (*PushNotificationResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.Topic == "" {
		return nil, errors.New("notification topic is required")
	}
	if req.Message == "" {
		return nil, errors.New("notification message is required")
	}

	ntfyURL := os.Getenv("NTFY_URL")
	if ntfyURL == "" {
		ntfyURL = "http://ntfy.apps.svc.cluster.local"
	}

	target := fmt.Sprintf("%s/%s", strings.TrimRight(ntfyURL, "/"), req.Topic)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(req.Message))
	if err != nil {
		return nil, fmt.Errorf("failed to create notification request: %w", err)
	}
	if req.Title != "" {
		httpReq.Header.Set("Title", req.Title)
	}
	if req.Priority != "" {
		httpReq.Header.Set("Priority", req.Priority)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		if os.Getenv("ENV") == "test" || os.Getenv("MOCK_NOTIFICATIONS") == "true" {
			return &PushNotificationResponse{
				Status:       "dispatched",
				Provider:     "mock-ntfy",
				Topic:        req.Topic,
				Title:        req.Title,
				Message:      req.Message,
				DispatchedAt: time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("failed to dispatch push notification via ntfy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("notification provider returned status %d", resp.StatusCode)
	}

	return &PushNotificationResponse{
		Status:       "dispatched",
		Provider:     "ntfy",
		Topic:        req.Topic,
		Title:        req.Title,
		Message:      req.Message,
		DispatchedAt: time.Now().UTC(),
	}, nil
}
