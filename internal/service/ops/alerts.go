package ops

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/pebble"
)

// AppMessage Dictionary Key Constants for Pebble Watch C SDK
const (
	PebbleKeyCmd       = pebble.PebbleKeyCmd
	PebbleKeyTicketID  = pebble.PebbleKeyTicketID
	PebbleKeyTitle     = pebble.PebbleKeyTitle
	PebbleKeyBody      = pebble.PebbleKeyBody
	PebbleKeyPriority  = pebble.PebbleKeyPriority
	PebbleKeyAuthor    = pebble.PebbleKeyAuthor
	PebbleKeyTimestamp = pebble.PebbleKeyTimestamp
)

type PebbleAlertPayload = pebble.PebbleAlertPayload
type PebbleAckRequest = pebble.PebbleAckRequest
type PebbleAckResponse = pebble.PebbleAckResponse

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
