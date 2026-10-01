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

type Service interface {
	GetPendingAlerts(ctx context.Context) ([]*PebbleAlertPayload, error)
	ProcessPebbleAck(ctx context.Context, req *PebbleAckRequest) (*PebbleAckResponse, error)
	CreateITTicket(ctx context.Context, forgejoRepo, title, body, authorUsername string, forgejoIssueID *int64) (*ops.ITTicket, error)
	GetITTicket(ctx context.Context, ticketID string) (*ops.ITTicket, error)
	ListITTickets(ctx context.Context, limit, offset int32) ([]*ops.ITTicket, error)
	SendPushNotification(ctx context.Context, req *PushNotificationRequest) (*PushNotificationResponse, error)
}

type opsService struct {
	repo ops.Repository
}

func NewService(repo ops.Repository) Service {
	return &opsService{repo: repo}
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

		dict := map[uint32]string{
			PebbleKeyCmd:       "1",
			PebbleKeyTicketID:  t.TicketID,
			PebbleKeyTitle:     t.Title,
			PebbleKeyBody:      bodyStr,
			PebbleKeyPriority:  "HIGH",
			PebbleKeyAuthor:    t.AuthorUsername,
			PebbleKeyTimestamp: t.CreatedAt.UTC().Format(time.RFC3339),
		}

		payloads = append(payloads, &PebbleAlertPayload{
			TicketID:       t.TicketID,
			Title:          t.Title,
			Priority:       "HIGH",
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
		ackBy = "emp-human-frank" // Default on-call engineer persona
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

func (s *opsService) CreateITTicket(ctx context.Context, forgejoRepo, title, body, authorUsername string, forgejoIssueID *int64) (*ops.ITTicket, error) {
	if title == "" || authorUsername == "" {
		return nil, errors.New("title and author_username are required")
	}

	bodyPtr := &body
	if body == "" {
		bodyPtr = nil
	}

	ticket := &ops.ITTicket{
		ForgejoIssueID: forgejoIssueID,
		ForgejoRepo:    forgejoRepo,
		Title:          title,
		Body:           bodyPtr,
		State:          "open",
		AuthorUsername: authorUsername,
	}

	return s.repo.CreateITTicket(ctx, ticket)
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

